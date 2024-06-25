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
	handledErrors "github.com/openshift/osd-network-verifier/pkg/errors"
	"github.com/openshift/osd-network-verifier/pkg/helpers"
	"github.com/openshift/osd-network-verifier/pkg/output"
	"github.com/openshift/osd-network-verifier/pkg/probes"
	"github.com/openshift/osd-network-verifier/pkg/probes/curl_json"
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
	userdataEndVerifier   = "USERDATA END"
	prepulledImageMessage = "Warning: could not pull the specified docker image, will try to use the prepulled one"
)

// AwsVerifier holds an aws client and knows how to fulfill the VerifierService which contains all functions needed for verifier
type AwsVerifier struct {
	AwsClient *aws.Client
	Logger    ocmlog.Logger
	Output    output.Output
}

// GetAMIForRegion returns the default X86 AWS AMI for the CurlJSONProbe given a region. This is unused within this codebase,
// but it's exported so that consumers can access this data
//
// Deprecated: GetAMIForRegion doesn't provide a way to check machine image IDs for platforms other than AWS, architectures
// other than X86, or probes other than CurlJSONProbe. It also doesn't return detailed errors. Instead, use:
// [probe_package].[ProbeName].GetMachineImageID(platformType string, cpuArchitecture string, region string)
func GetAMIForRegion(region string) string {
	ami, err := curl_json.CurlJSONProbe{}.GetMachineImageID(helpers.PlatformAWS, helpers.ArchX86, region)
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

	return &AwsVerifier{awsClient, logger, output.Output{}}, nil
}

func (a *AwsVerifier) validateInstanceType(ctx context.Context, instanceType string) error {
	descInput := ec2.DescribeInstanceTypesInput{
		InstanceTypes: []ec2Types.InstanceType{ec2Types.InstanceType(instanceType)},
	}

	a.writeDebugLogs(ctx, fmt.Sprintf("Gathering description of instance type %s from EC2", instanceType))
	descOut, err := a.AwsClient.DescribeInstanceTypes(ctx, &descInput)
	if err != nil {
		return handledErrors.NewGenericError(err)
	}

	// Effectively guaranteed to only have one match since we are casting c.instanceType into ec2Types.InstanceType
	// and placing it as the only InstanceType filter. Otherwise, ec2:DescribeInstanceTypes also accepts multiple as
	// an array of InstanceTypes which could return multiple matches.
	if len(descOut.InstanceTypes) != 1 {
		a.writeDebugLogs(ctx, fmt.Sprintf("matched instance types: %v", descOut.InstanceTypes))
		return fmt.Errorf("expected one instance type match for %s, got %d", instanceType, len(descOut.InstanceTypes))
	}

	if string(descOut.InstanceTypes[0].InstanceType) == instanceType {
		if descOut.InstanceTypes[0].Hypervisor != ec2Types.InstanceTypeHypervisorNitro {
			return fmt.Errorf("probe instances must use 'nitro' hypervisor to support reliable result collection, but %s uses '%s'", instanceType, descOut.InstanceTypes[0].Hypervisor)
		}
	}

	return nil
}

type createEC2InstanceInput struct {
	amiID               string
	SubnetID            string
	userdata            string
	KmsKeyID            string
	securityGroupId     string // Deprecated: prefer securityGroupIDs
	securityGroupIDs    []string
	tempSecurityGroupID string
	instanceCount       int32
	instanceType        string
	tags                map[string]string
	ctx                 context.Context
	keyPair             string
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

	// An empty string does not default to the default security group, and returns this error:
	// error performing ec2:RunInstances: Value () for parameter groupId is invalid. The value cannot be empty
	if input.securityGroupId != "" {
		eniSpecification.Groups = append(eniSpecification.Groups, input.securityGroupId)
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
		instanceReq.KeyName = awsTools.String(DEBUG_KEY_NAME)
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
	if err := waiter.Wait(input.ctx, &ec2.DescribeInstancesInput{InstanceIds: []string{instanceID}}, 2*time.Minute); err != nil {
		if err := a.AwsClient.TerminateEC2Instance(input.ctx, instanceID); err != nil {
			return instanceID, handledErrors.NewGenericError(err)
		}
		return "", fmt.Errorf("%s: terminated %s after timing out waiting for instance to be running", err, instanceID)
	}

	return instanceID, nil
}

func (a *AwsVerifier) findUnreachableEndpoints(ctx context.Context, instanceID string, probe probes.Probe) error {
	var consoleOutput string

	a.writeDebugLogs(ctx, "Scraping console output and waiting for user data script to complete...")

	// Periodically scrape console output and analyze the logs for any errors or a successful completion
	err := helpers.PollImmediate(30*time.Second, 4*time.Minute, func() (bool, error) {
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
		probe.ParseProbeOutput(rawProbeOutput, &a.Output)

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
	input := &ec2.CreateSecurityGroupInput{
		GroupName:   awsTools.String(name + "-" + helpers.RandSeq(5)),
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

	input_rules := &ec2.AuthorizeSecurityGroupEgressInput{
		GroupId:       output.GroupId,
		IpPermissions: defaultIpPermissions,
	}

	if _, err := a.AwsClient.AuthorizeSecurityGroupEgress(ctx, input_rules); err != nil {
		return &ec2.CreateSecurityGroupOutput{}, err
	}

	revoke_default_egress := &ec2.RevokeSecurityGroupEgressInput{
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

	if _, err := a.AwsClient.RevokeSecurityGroupEgress(ctx, revoke_default_egress); err != nil {
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
