package awsverifier

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"

	awsTools "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/go-playground/validator"
	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/clients/aws"
	"github.com/openshift/osd-network-verifier/pkg/data/cloud"
	"github.com/openshift/osd-network-verifier/pkg/data/cpu"
	handledErrors "github.com/openshift/osd-network-verifier/pkg/errors"
	"github.com/openshift/osd-network-verifier/pkg/helpers"
	"github.com/openshift/osd-network-verifier/pkg/output"
	"github.com/openshift/osd-network-verifier/pkg/probes"
	"github.com/openshift/osd-network-verifier/pkg/probes/curl"
)

// defaultIpPermissions contains the base set of ipPermissions (egress rules)
// allowed on the verifier's temporary security group (only created when the
// user doesn't provide their own security group)
var defaultIpPermissions = []ec2Types.IpPermission{
	{
		FromPort:   awsTools.Int32(80),
		ToPort:     awsTools.Int32(80),
		IpProtocol: awsTools.String("tcp"),
		IpRanges: []ec2Types.IpRange{
			{
				CidrIp: awsTools.String("0.0.0.0/0"),
			},
		},
	},
	{
		FromPort:   awsTools.Int32(443),
		ToPort:     awsTools.Int32(443),
		IpProtocol: awsTools.String("tcp"),
		IpRanges: []ec2Types.IpRange{
			{
				CidrIp: awsTools.String("0.0.0.0/0"),
			},
		},
	},
	{
		FromPort:   awsTools.Int32(9997),
		ToPort:     awsTools.Int32(9997),
		IpProtocol: awsTools.String("tcp"),
		IpRanges: []ec2Types.IpRange{
			{
				CidrIp: awsTools.String("0.0.0.0/0"),
			},
		},
	},
}

const (
	instanceCount int32 = 1

	// TODO find a location for future docker images
	// This corresponds with the quay tag: v0.1.90-f2e86a9
	networkValidatorImage = "quay.io/app-sre/osd-network-verifier@sha256:137bf177c2e87732b2692c1af39d3b79b2f84c7f0ee9254df4ea4412dddfab1e"
	networkValidatorRepo  = "quay.io/app-sre/osd-network-verifier"
	invalidKMSCode        = "Client.InvalidKMSKey.InvalidState"
)

// AwsVerifier holds an aws client and knows how to fulfill the VerifierService which contains all functions needed for verifier
type AwsVerifier struct {
	AwsClient *aws.Client
	Logger    ocmlog.Logger
	Output    output.Output
	// This cache is only to be used inside describeInstanceType() to minimize nil ptr error risk
	cachedInstanceTypeInfo *ec2Types.InstanceTypeInfo
}

// GetAMIForRegion returns the default X86 AWS AMI for the CurlJSONProbe given a region. This is unused within this codebase,
// but it's exported so that consumers can access this data
//
// Deprecated: GetAMIForRegion doesn't provide a way to check machine image IDs for platforms other than AWS, architectures
// other than X86, or probes other than CurlJSONProbe. It also doesn't return detailed errors. Instead, use:
// [probe_package].[ProbeName].GetMachineImageID(platformType, cpuArch, region)
func GetAMIForRegion(region string) string {
	ami, err := curl.Probe{}.GetMachineImageID(cloud.AWSClassic, cpu.ArchX86, region)
	if err != nil {
		return ""
	}
	return ami
}

// NewAwsVerifierFromConfig assembles an AwsVerifier given an aws-sdk-go-v2 config and an ocm logger
func NewAwsVerifierFromConfig(cfg awsTools.Config, logger ocmlog.Logger) (*AwsVerifier, error) {
	awsClient, err := aws.NewClientFromConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &AwsVerifier{
		AwsClient: awsClient,
		Logger:    logger,
	}, nil
}

func NewAwsVerifier(accessID, accessSecret, sessionToken, region, profile string, debug bool) (*AwsVerifier, error) {
	// Create logger
	builder := ocmlog.NewStdLoggerBuilder()
	builder.Debug(debug)
	logger, err := builder.Build()
	if err != nil {
		return &AwsVerifier{}, fmt.Errorf("unable to build logger: %s", err.Error())
	}

	awsClient, err := aws.NewClient(context.TODO(), accessID, accessSecret, sessionToken, region, profile)
	if err != nil {
		return &AwsVerifier{}, err
	}

	return &AwsVerifier{
		AwsClient: awsClient,
		Logger:    logger,
		Output:    output.Output{},
	}, nil
}

// describeInstanceType calls the AWS EC2 API's "DescribeInstanceTypes" endpoint for the given
// instanceType and caches the answer in a.cachedInstanceTypeInfo. Subsequent
func (a *AwsVerifier) describeInstanceType(ctx context.Context, instanceType string) (*ec2Types.InstanceTypeInfo, error) {
	// Make API request if cache is empty or doesn't match requested instanceType
	if a.cachedInstanceTypeInfo == nil || string(a.cachedInstanceTypeInfo.InstanceType) != instanceType {
		a.writeDebugLogs(ctx, fmt.Sprintf("Gathering description of instance type %s from EC2", instanceType))
		descInput := ec2.DescribeInstanceTypesInput{
			InstanceTypes: []ec2Types.InstanceType{ec2Types.InstanceType(instanceType)},
		}
		descOut, err := a.AwsClient.DescribeInstanceTypes(ctx, &descInput)
		if err != nil {
			return nil, handledErrors.NewGenericError(err)
		}

		// Effectively guaranteed to only have one match since we are casting c.instanceType into ec2Types.InstanceType
		// and placing it as the only InstanceType filter. Otherwise, ec2:DescribeInstanceTypes also accepts multiple as
		// an array of InstanceTypes which could return multiple matches.
		if len(descOut.InstanceTypes) != 1 || string(descOut.InstanceTypes[0].InstanceType) != instanceType {
			return nil, fmt.Errorf("unexpected instance type matches for %s, got %v", instanceType, descOut.InstanceTypes)
		}

		a.cachedInstanceTypeInfo = &descOut.InstanceTypes[0]
	}

	return a.cachedInstanceTypeInfo, nil
}

// instanceTypeUsesNitro asks the AWS API whether the provided instanceType uses the "Nitro"
// hypervisor. Nitro is the only hypervisor supporting serial console output, which we need to
// collect in order to gather probe results
func (a *AwsVerifier) instanceTypeUsesNitro(ctx context.Context, instanceType string) (bool, error) {
	// Fetch instance type info
	instanceTypeInfo, err := a.describeInstanceType(ctx, instanceType)
	if err != nil {
		return false, err
	}

	// Return true if instance type uses nitro
	return instanceTypeInfo.Hypervisor == ec2Types.InstanceTypeHypervisorNitro, nil
}

// instanceTypeArchitecture asks the AWS API about the CPU architecture(s) supported by the provided
// instanceType and returns the first answer matching a cpu.Architecture known to the verifier. An
// error is returned if the API call fails or if the verifier has no support for the instanceType's CPU
func (a *AwsVerifier) instanceTypeArchitecture(ctx context.Context, instanceType string) (cpu.Architecture, error) {
	// Fetch instance type info
	instanceTypeInfo, err := a.describeInstanceType(ctx, instanceType)
	if err != nil {
		return cpu.Architecture{}, err
	}

	// Iterate over SupportedArchitectures until a matching cpu.Architecture is found
	for _, instanceArch := range instanceTypeInfo.ProcessorInfo.SupportedArchitectures {
		if arch := cpu.ArchitectureByName(string(instanceArch)); arch.IsValid() {
			return arch, nil
		}
	}

	return cpu.Architecture{}, fmt.Errorf("instance type %s doesn't support any of our supported architectures", instanceType)
}

// selectInstanceType selects an approriate EC2 instance type based on the caller's preferences and
// the verifier's limitations. All parameters are optional: leaving one or more "empty" (i.e.,
// passing "zero values") will result in a value being inferred from other parameters or from a
// pre-programmed default. Any provided parameters may be overridden (e.g., if an unsupported
// non-Nitro instance type is requested) with the "next best" supported alternative. For example,
// calling selectInstanceType(ctx, "c4.large", cpu.ArchX86) will return ("t3.micro", cpu.ArchX86,
// nil) because c4-type instances do not use Nitro hypervisors and t3.micro is a suitable
// alternative that uses the same CPU architecture as c4.large.
func (a *AwsVerifier) selectInstanceType(ctx context.Context, instanceType string, cpuArchitecture cpu.Architecture) (string, cpu.Architecture, error) {
	var err error

	// Validate any requested instance type
	validInstanceTypeRequested := false
	if instanceType != "" {
		// Derive CPU arch from requested InstanceType so that we can pick an appropriate AMI, and
		// if necessary (e.g., because given type is non-Nitro), an alternative instance type with
		// the same CPU arch
		cpuArchitecture, err = a.instanceTypeArchitecture(ctx, instanceType)
		if err != nil {
			return "", cpu.Architecture{}, fmt.Errorf("failed to validate CPU architecture of instance type %s: %w", instanceType, err)
		}

		// Determine if given InstanceType uses the required Nitro hypervisor
		validInstanceTypeRequested, err = a.instanceTypeUsesNitro(ctx, instanceType)
		if err != nil {
			return "", cpu.Architecture{}, fmt.Errorf("failed to determine hypervisor of instance type %s: %w", instanceType, err)
		}
	}

	// Ensure we have a valid CPU arch beyond this point, defaulting to X86 if necessary
	if !cpuArchitecture.IsValid() {
		cpuArchitecture = cpu.ArchX86
		a.writeDebugLogs(ctx, fmt.Sprintf("defaulted to %s CPU architecture", cpuArchitecture))
	}

	// If no instance type was requested (or if instance type  is invalid), select one based on CPU arch
	if !validInstanceTypeRequested {
		if instanceType != "" {
			// Warn user that we're ignoring their invalid requested instance type
			a.writeDebugLogs(ctx, fmt.Sprintf("ignoring requested instance type %s because it uses a non-Nitro hypervisor", instanceType))
		}

		instanceType, err = cpuArchitecture.DefaultInstanceType(cloud.AWSClassic)
		if err != nil {
			return "", cpu.Architecture{}, fmt.Errorf("failed to determine default instance type for CPU architecture %s: %w", cpuArchitecture, err)
		}
		a.writeDebugLogs(ctx, fmt.Sprintf("defaulted to instance type %s", instanceType))
	}

	return instanceType, cpuArchitecture, nil
}

type createEC2InstanceInput struct {
	amiID               string
	SubnetID            string
	userdata            string
	KmsKeyID            string
	securityGroupIDs    []string
	tempSecurityGroupID string
	instanceCount       int32
	instanceType        string
	tags                map[string]string
	ctx                 context.Context
	keyPair             string
	vpcID               string
}

func (a *AwsVerifier) createEC2Instance(input createEC2InstanceInput) (string, error) {
	ebsBlockDevice := &ec2Types.EbsBlockDevice{
		VolumeSize:          awsTools.Int32(10),
		DeleteOnTermination: awsTools.Bool(true),
		Encrypted:           awsTools.Bool(true),
	}
	// Check if KMS key was specified for root volume encryption
	if input.KmsKeyID != "" {
		ebsBlockDevice.KmsKeyId = awsTools.String(input.KmsKeyID)
	}

	eniSpecification := ec2Types.InstanceNetworkInterfaceSpecification{
		DeviceIndex: awsTools.Int32(0),
		SubnetId:    awsTools.String(input.SubnetID),
	}

	if len(input.securityGroupIDs) > 0 {
		eniSpecification.Groups = input.securityGroupIDs
	}

	if input.tempSecurityGroupID != "" {
		eniSpecification.Groups = append(eniSpecification.Groups, input.tempSecurityGroupID)
	}

	// Build our request, converting the go base types into the pointers required by the SDK
	instanceReq := ec2.RunInstancesInput{
		ImageId:      awsTools.String(input.amiID),
		MaxCount:     awsTools.Int32(input.instanceCount),
		MinCount:     awsTools.Int32(input.instanceCount),
		InstanceType: ec2Types.InstanceType(input.instanceType),
		// Tell EC2 to delete this instance if it shuts itself down, in case explicit instance deletion fails
		InstanceInitiatedShutdownBehavior: ec2Types.ShutdownBehaviorTerminate,
		// Because we're making this VPC aware, we also have to include a network interface specification
		NetworkInterfaces: []ec2Types.InstanceNetworkInterfaceSpecification{eniSpecification},
		// We specify block devices mainly to enable EBS encryption
		BlockDeviceMappings: []ec2Types.BlockDeviceMapping{
			{
				DeviceName: awsTools.String("/dev/sda1"),
				Ebs:        ebsBlockDevice,
			},
		},
		TagSpecifications: []ec2Types.TagSpecification{
			{
				ResourceType: ec2Types.ResourceTypeInstance,
				Tags:         buildTags(input.tags),
			},
			{
				ResourceType: ec2Types.ResourceTypeVolume,
				Tags:         buildTags(input.tags),
			},
		},
		//enable IMDSv2 for instances
		MetadataOptions: &ec2Types.InstanceMetadataOptionsRequest{
			HttpTokens:   ec2Types.HttpTokensStateRequired,
			HttpEndpoint: ec2Types.InstanceMetadataEndpointStateEnabled,
		},
		UserData: awsTools.String(input.userdata),
	}

	if input.keyPair != "" {
		instanceReq.KeyName = awsTools.String(DebugKeyName)
	}
	// Finally, we make our request
	instanceResp, err := a.AwsClient.RunInstances(input.ctx, &instanceReq)
	if err != nil {
		return "", handledErrors.NewGenericError(err)
	}

	for _, i := range instanceResp.Instances {
		a.Logger.Info(input.ctx, "Created instance with ID: %s", *i.InstanceId)
	}

	if len(instanceResp.Instances) == 0 {
		// Shouldn't happen, but ensure safety of the following logic
		return "", handledErrors.NewGenericError(errors.New("unexpectedly found 0 instances after creation, please try again"))
	}

	instanceID := *instanceResp.Instances[0].InstanceId

	// Wait up to 5 minutes for the instance to be running
	waiter := ec2.NewInstanceRunningWaiter(a.AwsClient)
	if err := waiter.Wait(input.ctx, &ec2.DescribeInstancesInput{InstanceIds: []string{instanceID}}, 5*time.Minute); err != nil {
		resp, err := a.AwsClient.DescribeInstances(input.ctx, &ec2.DescribeInstancesInput{
			InstanceIds: []string{instanceID},
		})
		if err != nil {
			fmt.Println("Warning: Waiter Describe instances failure.")
		}

		var stateCode string
		if resp != nil && resp.Reservations[0].Instances[0].StateReason.Code != nil {
			stateCode = *resp.Reservations[0].Instances[0].StateReason.Code
		}

		waiterErr := fmt.Errorf("%s: terminated %s after timing out waiting for instance to be running", err, instanceID)
		if stateCode == invalidKMSCode {
			waiterErr = handledErrors.NewKmsError("encountered issue accessing KMS key when launching instance.")
		}

		// Switch the instance SecurityGroup to the default before terminating to avoid a cleanup race condition. This is
		// handled by the normal cleanup process, except in this specific case where we fail early because of KMS issues.
		defaultSecurityGroupID := a.fetchVpcDefaultSecurityGroup(input.ctx, input.vpcID)
		if defaultSecurityGroupID != "" {
			// Replace the SecurityGroup attached to the instance with the default one for the VPC to allow for graceful
			// termination of the network-verifier created temporary SecurityGroup. If we hit an error, we ignore it
			// and continue with normal termination of the instance.
			_ = a.modifyInstanceSecurityGroup(input.ctx, instanceID, defaultSecurityGroupID)
			a.Logger.Info(input.ctx, "Modified the instance to use the default security group")
		}

		if err := a.AwsClient.TerminateEC2Instance(input.ctx, instanceID); err != nil {
			return instanceID, handledErrors.NewGenericError(err)
		}

		return "", waiterErr
	}

	return instanceID, nil
}

func (a *AwsVerifier) findUnreachableEndpoints(ctx context.Context, instanceID string, probe probes.Probe, ensurePrivate bool) error {
	var consoleOutput string

	a.writeDebugLogs(ctx, "Scraping console output and waiting for user data script to complete...")

	// Periodically scrape console output and analyze the logs for any errors or a successful completion
	err := helpers.PollImmediate(10*time.Second, 270*time.Second, func() (bool, error) {
		b64EncodedConsoleOutput, err := a.AwsClient.GetConsoleOutput(ctx, &ec2.GetConsoleOutputInput{
			InstanceId: awsTools.String(instanceID),
			Latest:     awsTools.Bool(true),
		})
		if err != nil {
			return false, handledErrors.NewGenericError(err)
		}

		// Return and resume waiting if console output is still nil
		if b64EncodedConsoleOutput.Output == nil {
			return false, nil
		}

		// In the early stages, an ec2 instance may be running but the console is not populated with any data
		if len(*b64EncodedConsoleOutput.Output) == 0 {
			a.writeDebugLogs(ctx, "EC2 console consoleOutput not yet populated with data, continuing to wait...")
			return false, nil
		}

		// Decode base64-encoded console output
		consoleOutputBytes, err := base64.StdEncoding.DecodeString(*b64EncodedConsoleOutput.Output)
		if err != nil {
			a.writeDebugLogs(ctx, fmt.Sprintf("Error decoding console consoleOutput, will retry on next check interval: %s", err))
			return false, nil
		}
		consoleOutput = string(consoleOutputBytes)

		// Check for startingToken and endingToken
		startingTokenSeen := strings.Contains(consoleOutput, probe.GetStartingToken())
		endingTokenSeen := strings.Contains(consoleOutput, probe.GetEndingToken())
		if !startingTokenSeen {
			if endingTokenSeen {
				a.writeDebugLogs(ctx, fmt.Sprintf("raw console logs:\n---\n%s\n---", consoleOutput))
				return false, handledErrors.NewGenericError(fmt.Errorf("probe output corrupted: endingToken encountered before startingToken"))
			}
			a.writeDebugLogs(ctx, "consoleOutput contains data, but probe has not yet printed startingToken, continuing to wait...")
			return false, nil
		}
		if !endingTokenSeen {
			a.writeDebugLogs(ctx, "consoleOutput contains startingToken, but probe has not yet printed endingToken, continuing to wait...")
			return false, nil
		}

		// If we make it this far, we know that both startingTokenSeen and endingTokenSeen are true

		// Separate the probe's output from the rest of the console output (using startingToken and endingToken)
		rawProbeOutput := strings.TrimSpace(helpers.CutBetween(consoleOutput, probe.GetStartingToken(), probe.GetEndingToken()))
		if len(rawProbeOutput) < 1 {
			a.writeDebugLogs(ctx, fmt.Sprintf("raw console logs:\n---\n%s\n---", consoleOutput))
			return false, handledErrors.NewGenericError(fmt.Errorf("probe output corrupted: no data between startingToken and endingToken"))
		}

		// Send probe's output off to the Probe interface for parsing
		a.writeDebugLogs(ctx, fmt.Sprintf("probe output:\n---\n%s\n---", rawProbeOutput))
		probe.ParseProbeOutput(ensurePrivate, rawProbeOutput, &a.Output)
		return true, nil
	})

	return err
}

func buildTags(tags map[string]string) []ec2Types.Tag {
	tagList := make([]ec2Types.Tag, 0, len(tags))
	for k, v := range tags {
		t := ec2Types.Tag{
			Key:   awsTools.String(k),
			Value: awsTools.String(v),
		}
		tagList = append(tagList, t)
	}

	return tagList
}

func (a *AwsVerifier) writeDebugLogs(ctx context.Context, log string) {
	a.Output.AddDebugLogs(log)
	a.Logger.Debug(ctx, log)
}

// CreateSecurityGroup creates a security group with the specified name and cluster tag key in a specified VPC
func (a *AwsVerifier) CreateSecurityGroup(ctx context.Context, tags map[string]string, name, vpcId string) (*ec2.CreateSecurityGroupOutput, error) {
	seq, err := helpers.RandSeq(5)
	if err != nil {
		return &ec2.CreateSecurityGroupOutput{}, err
	}

	input := &ec2.CreateSecurityGroupInput{
		GroupName:   awsTools.String(name + "-" + seq),
		VpcId:       &vpcId,
		Description: awsTools.String("osd-network-verifier security group"),
		TagSpecifications: []ec2Types.TagSpecification{
			{
				ResourceType: ec2Types.ResourceTypeSecurityGroup,
				Tags:         buildTags(tags),
			},
		},
	}
	a.writeDebugLogs(ctx, "Creating a Security group")
	output, err := a.AwsClient.CreateSecurityGroup(ctx, input)
	if err != nil {
		return &ec2.CreateSecurityGroupOutput{}, err
	}

	a.writeDebugLogs(ctx, fmt.Sprintf("Waiting for the Security Group to exist: %s", *output.GroupId))
	// Wait up to 1 minutes for the security group to exist
	waiter := ec2.NewSecurityGroupExistsWaiter(a.AwsClient)
	if err := waiter.Wait(ctx, &ec2.DescribeSecurityGroupsInput{GroupIds: []string{*output.GroupId}}, 1*time.Minute); err != nil {
		a.writeDebugLogs(ctx, fmt.Sprintf("Error waiting for the security group to exist: %s, attempting to delete the Security Group", *output.GroupId))
		_, err := a.AwsClient.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{GroupId: output.GroupId})
		if err != nil {
			return &ec2.CreateSecurityGroupOutput{}, handledErrors.NewGenericError(err)
		}
		return &ec2.CreateSecurityGroupOutput{}, fmt.Errorf("deleted %s after timing out waiting for security group to exist", *output.GroupId)
	}

	a.Logger.Info(ctx, "Created security group with ID: %s", *output.GroupId)

	inputRules := &ec2.AuthorizeSecurityGroupEgressInput{
		GroupId:       output.GroupId,
		IpPermissions: defaultIpPermissions,
	}

	if _, err := a.AwsClient.AuthorizeSecurityGroupEgress(ctx, inputRules); err != nil {
		return &ec2.CreateSecurityGroupOutput{}, err
	}

	revokeDefaultEgress := &ec2.RevokeSecurityGroupEgressInput{
		GroupId: output.GroupId,
		IpPermissions: []ec2Types.IpPermission{
			{
				FromPort:   awsTools.Int32(-1),
				ToPort:     awsTools.Int32(-1),
				IpProtocol: awsTools.String("-1"),
				IpRanges: []ec2Types.IpRange{
					{
						CidrIp: awsTools.String("0.0.0.0/0"),
					},
				},
			},
		},
	}

	if _, err := a.AwsClient.RevokeSecurityGroupEgress(ctx, revokeDefaultEgress); err != nil {
		return &ec2.CreateSecurityGroupOutput{}, err
	}

	return output, nil
}

// ipPermissionFromURL generates an EC2 IpPermission (for use in sec. group rules) from a given http(s)
// URL (e.g., "http://10.0.8.1:8080" or "https://proxy.example.com:1234") with the given human-readable
// description
func ipPermissionFromURL(urlStr string, description string) (*ec2Types.IpPermission, error) {

	// Validate URL by parsing it
	parsedUrl, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	// Extract the hostName from the URL
	parsedUrlHostnameStr := parsedUrl.Hostname()

	// Use a validator to check if parsedUrlHostnameStr is a fully qualified domain
	// name (FQDN, e.g., "example.com") or an IP address
	validate := validator.New()
	err = validate.Var(parsedUrlHostnameStr, "fqdn")
	if err == nil {
		// If parsedUrlHostnameStr is an FQDN, set the ip to 0.0.0.0 in order to
		// create an outbound SG rule to all IPs. Ref: OSD-20562
		parsedUrlHostnameStr = "0.0.0.0"
	}

	// Ensure parsedUrlHostnameStr is a valid IP address at this point
	ipAddr, err := netip.ParseAddr(parsedUrlHostnameStr)
	if err != nil {
		return nil, errors.New("URL must be valid IP address or FQDN (fully qualified domain name)")
	}

	// Then attempt to extract port number and cast to int32
	parsedUrlPortStr := parsedUrl.Port()
	if parsedUrlPortStr == "" {
		// Infer port from URL scheme (http/https)
		switch parsedUrl.Scheme {
		case "http":
			parsedUrlPortStr = "80"
		case "https":
			parsedUrlPortStr = "443"
		default:
			return nil, errors.New("unsupported URL scheme")
		}
	}
	parsedUrlPortInt64, err := strconv.ParseInt(parsedUrlPortStr, 10, 32)
	if err != nil {
		return nil, errors.New("invalid port")
	}
	parsedUrlPortInt32 := int32(parsedUrlPortInt64)

	// Construct egress rule (ipPermission) and add to array
	ipPerm := &ec2Types.IpPermission{
		FromPort:   awsTools.Int32(parsedUrlPortInt32),
		ToPort:     awsTools.Int32(parsedUrlPortInt32),
		IpProtocol: awsTools.String("tcp"),
	}
	// Set CIDR range based on IP version (/0 for 0.0.0.0 or ::, /32 for
	// specific IPv4, /128 for specific IPv6)
	if ipAddr.Is4() {
		cidrPrefixLength := "/32"
		if ipAddr.String() == "0.0.0.0" {
			cidrPrefixLength = "/0"
		}
		ipPerm.IpRanges = []ec2Types.IpRange{
			{
				CidrIp:      awsTools.String(ipAddr.String() + cidrPrefixLength),
				Description: awsTools.String(description),
			},
		}
	}
	if ipAddr.Is6() {
		cidrPrefixLength := "/128"
		if ipAddr.String() == "::" {
			cidrPrefixLength = "/0"
		}
		ipPerm.Ipv6Ranges = []ec2Types.Ipv6Range{
			{
				CidrIpv6:    awsTools.String(ipAddr.String() + cidrPrefixLength),
				Description: awsTools.String(description),
			},
		}
	}

	return ipPerm, nil
}

// ipPermissionSetFromURLs wraps ipPermissionFromURL() with deduplication logic. I.e.,
// for each URL string given in urlStrs, an IpPermission (with a description
// based on the provided descriptionPrefix) will be generated and added to the
// returned slice of IpPermissions UNLESS that slice (or defaultIpPermissions) already
// contains an equivalent IpPermission (which would cause an API call using that slice
// to be rejected by AWS). It may return an empty slice if no additional IpPermissions
// (beyond defaultIpPermissions) are needed to allow egress to the provided urlStrs
func ipPermissionSetFromURLs(urlStrs []string, descriptionPrefix string) ([]ec2Types.IpPermission, error) {
	// Create zero-length slice of ipPermissionSet with a capacity equal to the quantity
	// of proxy URLs provided
	var ipPermissionSet = make([]ec2Types.IpPermission, 0, len(urlStrs))

	// Iterate over provided proxy URLs, converting each to an IpPermission
	for _, urlStr := range urlStrs {
		ipPerm, err := ipPermissionFromURL(urlStr, descriptionPrefix+urlStr)
		if err != nil {
			return nil, fmt.Errorf("unable to create security group rule from URL '%s': %w", urlStr, err)
		}
		// Add ipPerm to ipPermissions only if not already there (AWS will reject duplicates)
		ipPermAlreadyExists := false
		for _, existingIPPerm := range ipPermissionSet {
			ipPermAlreadyExists = ipPermAlreadyExists || helpers.IPPermissionsEquivalent(*ipPerm, existingIPPerm)
		}
		// Also check against defaultIpPermissions
		for _, defaultIPPerm := range defaultIpPermissions {
			ipPermAlreadyExists = ipPermAlreadyExists || helpers.IPPermissionsEquivalent(*ipPerm, defaultIPPerm)
		}
		if !ipPermAlreadyExists {
			ipPermissionSet = append(ipPermissionSet, *ipPerm)
		}
	}

	return ipPermissionSet, nil
}

// AllowSecurityGroupProxyEgress adds rules to an existing security group that allow
// egress to the specified proxies. It returns nil if the necessary rules already exist
// in defaultIpPermissions
func (a *AwsVerifier) AllowSecurityGroupProxyEgress(ctx context.Context, securityGroupID string, proxyURLs []string) (*ec2.AuthorizeSecurityGroupEgressOutput, error) {
	// Generate a deduplicated set of IpPermissions from the given proxy URLs
	ipPermissions, err := ipPermissionSetFromURLs(proxyURLs, "Egress to user-provided proxy ")
	if err != nil {
		return nil, handledErrors.NewGenericError(fmt.Errorf("error occurred while authorizing egress to proxy: %w", err))
	}

	// Make AWS call to add rule to security group
	if len(ipPermissions) > 0 {
		authSecGrpIngInput := &ec2.AuthorizeSecurityGroupEgressInput{
			GroupId:       awsTools.String(securityGroupID),
			IpPermissions: ipPermissions,
		}
		out, err := a.AwsClient.AuthorizeSecurityGroupEgress(ctx, authSecGrpIngInput)
		if err != nil {
			return nil, handledErrors.NewGenericError(err)
		}

		return out, nil
	}
	return nil, nil
}

// GetVpcIdFromSubnetId takes in a subnet id and returns the associated VPC id
func (a *AwsVerifier) GetVpcIdFromSubnetId(ctx context.Context, vpcSubnetID string) (string, error) {
	input := &ec2.DescribeSubnetsInput{

		SubnetIds: []string{vpcSubnetID},
	}

	output, err := a.AwsClient.DescribeSubnets(ctx, input)
	if err != nil {
		return "", err
	}

	// What if we get an empty vpc-id for a returned subnet
	if len(output.Subnets) == 0 {
		return "", fmt.Errorf("no subnets returned for subnet id: %s", vpcSubnetID)
	}

	// What if the Subnets array has 0 entries
	vpcId := *output.Subnets[0].VpcId
	if vpcId == "" {
		// return "", errors.New("Empty VPCId for the returned subnet")
		return "", fmt.Errorf("empty vpc id for the returned subnet: %s", vpcSubnetID)
	}
	return vpcId, nil
}

// fetchVpcDefaultSecurityGroup will return either the 'default' SG ID, or an empty string if not found/an error is hit
func (a *AwsVerifier) fetchVpcDefaultSecurityGroup(ctx context.Context, vpcId string) string {
	describeSGOutput, err := a.AwsClient.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		Filters: []ec2Types.Filter{
			{
				Name:   awsTools.String("vpc-id"),
				Values: []string{vpcId},
			},
			{
				Name:   awsTools.String("group-name"),
				Values: []string{"default"},
			},
		},
	})

	if err != nil {
		return ""
	}

	for _, SG := range describeSGOutput.SecurityGroups {
		if *SG.GroupName == "default" {
			return *SG.GroupId
		}
	}

	return ""
}

func (a *AwsVerifier) modifyInstanceSecurityGroup(ctx context.Context, instanceID string, securityGroupID string) error {
	_, err := a.AwsClient.ModifyInstanceAttribute(ctx, &ec2.ModifyInstanceAttributeInput{
		InstanceId: &instanceID,
		Groups:     []string{securityGroupID},
	})

	return err
}
