package awsverifier

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"regexp"
	"time"

	awsTools "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/clients/aws"
	handledErrors "github.com/openshift/osd-network-verifier/pkg/errors"
	"github.com/openshift/osd-network-verifier/pkg/helpers"
	"github.com/openshift/osd-network-verifier/pkg/output"
)

var (
	defaultAmi = map[string]string{
		"af-south-1":     "ami-0496b860bd457437f",
		"ap-east-1":      "ami-0a4acc6e2240c28c0",
		"ap-northeast-1": "ami-0c803c5048783502d",
		"ap-northeast-2": "ami-097bfe6c51df21ff2",
		"ap-northeast-3": "ami-0f9fd8814f605fced",
		"ap-south-1":     "ami-0680fbc5b2a9a8a27",
		"ap-south-2":     "ami-0b17f7be910efe584",
		"ap-southeast-1": "ami-03a9fb3956e5eaa85",
		"ap-southeast-2": "ami-0d1cfea2e07d4a8ba",
		"ap-southeast-3": "ami-03b618de9a400b62e",
		"ap-southeast-4": "ami-091b4c35a00bf0c0c",
		"ca-central-1":   "ami-03f4c87d571a1d38c",
		"eu-central-1":   "ami-04210135cdf44ea1c",
		"eu-central-2":   "ami-0c9cd51945a26d2d7",
		"eu-north-1":     "ami-05de3c8fc38d53b0e",
		"eu-south-1":     "ami-02f3c2b3f95ec5471",
		"eu-west-1":      "ami-04faa6d68d9fda4dd",
		"eu-west-2":      "ami-01c4c11d06660214c",
		"eu-west-3":      "ami-0ab155e4e470cf8ba",
		"me-central-1":   "ami-071ca4418c87c0a0f",
		"me-south-1":     "ami-0194a03c002b56920",
		"sa-east-1":      "ami-08c2432c9373602a5",
		"us-east-1":      "ami-00cd2c6d77d317c06",
		"us-east-2":      "ami-01187490fcef7e973",
		"us-west-1":      "ami-08013983a3c8324c9",
		"us-west-2":      "ami-033cd6d1c118adec6",
	}
)

const (
	instanceCount int32 = 1

	// TODO find a location for future docker images
	// This corresponds with the tag: v0.1.73-b096215
	networkValidatorImage = "quay.io/app-sre/osd-network-verifier@sha256:55d74f02bc46673d9d0f7f1511a3fd4be2e5cf0a773501a23a87b85dd8ab233f"
	networkValidatorRepo  = "quay.io/app-sre/osd-network-verifier"
	userdataEndVerifier   = "USERDATA END"
	prepulledImageMessage = "Warning: could not pull the specified docker image, will try to use the prepulled one"
)

// AwsVerifier holds an aws client and knows how to fuifill the VerifierSerice which contains all functions needed for verifier
type AwsVerifier struct {
	AwsClient *aws.Client
	Logger    ocmlog.Logger
	Output    output.Output
}

// GetAMIForRegion returns the default AMI given a region.
// This is unused within this codebase, but exported so that consumers can access the values of defaultAmi
func GetAMIForRegion(region string) string {
	return defaultAmi[region]
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
			return fmt.Errorf("instance type %s must use hypervisor type 'nitro' to support reliable result collection, using %s", instanceType, descOut.InstanceTypes[0].Hypervisor)
		}
	}

	return nil
}

type createEC2InstanceInput struct {
	amiID           string
	SubnetID        string
	userdata        string
	KmsKeyID        string
	securityGroupId string
	instanceCount   int32
	instanceType    string
	tags            map[string]string
	ctx             context.Context
	keyPair         string
}

func (a *AwsVerifier) createEC2Instance(input createEC2InstanceInput) (string, error) {
	ebsBlockDevice := &ec2Types.EbsBlockDevice{
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
		eniSpecification.Groups = []string{input.securityGroupId}
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
				DeviceName: awsTools.String("/dev/xvda"),
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

func (a *AwsVerifier) findUnreachableEndpoints(ctx context.Context, instanceID string) error {
	var (
		b64ConsoleLogs string
		consoleLogs    string
	)

	// reUserDataComplete indicates that the network validation completed
	reUserDataComplete := regexp.MustCompile(userdataEndVerifier)
	// reSuccess indicates that network validation was successful
	reSuccess := regexp.MustCompile(`Success!`)
	// rePrepulledImage indicates that the network verifier is using a prepulled image
	rePrepulledImage := regexp.MustCompile(prepulledImageMessage)

	a.writeDebugLogs(ctx, "Scraping console output and waiting for user data script to complete...")

	// Periodically scrape console output and analyze the logs for any errors or a successful completion
	err := helpers.PollImmediate(30*time.Second, 4*time.Minute, func() (bool, error) {
		consoleOutput, err := a.AwsClient.GetConsoleOutput(ctx, &ec2.GetConsoleOutputInput{
			InstanceId: awsTools.String(instanceID),
			Latest:     awsTools.Bool(true),
		})
		if err != nil {
			return false, handledErrors.NewGenericError(err)
		}

		if consoleOutput.Output != nil {
			// In the early stages, an ec2 instance may be running but the console is not populated with any data
			if len(*consoleOutput.Output) == 0 {
				a.writeDebugLogs(ctx, "EC2 console consoleOutput not yet populated with data, continuing to wait...")
				return false, nil
			}

			// Store base64-encoded output for debug logs
			b64ConsoleLogs = *consoleOutput.Output

			// The console consoleOutput starts out base64 encoded
			scriptOutput, err := base64.StdEncoding.DecodeString(*consoleOutput.Output)
			if err != nil {
				a.writeDebugLogs(ctx, fmt.Sprintf("Error decoding console consoleOutput, will retry on next check interval: %s", err))
				return false, nil
			}

			consoleLogs = string(scriptOutput)

			// Check for the specific string we consoleOutput in the generated userdata file at the end to verify the userdata script has run
			// It is possible we get EC2 console consoleOutput, but the userdata script has not yet completed.
			userDataComplete := reUserDataComplete.FindString(consoleLogs)
			if len(userDataComplete) < 1 {
				a.writeDebugLogs(ctx, "EC2 console consoleOutput contains data, but end of userdata script not seen, continuing to wait...")
				return false, nil
			}

			// Check if the result is success
			success := reSuccess.FindAllStringSubmatch(consoleLogs, -1)
			if len(success) > 0 {
				return true, nil
			}

			// Add a message to debug logs if we're using the prepulled image
			prepulledImage := rePrepulledImage.FindAllString(consoleLogs, -1)
			if len(prepulledImage) > 0 {
				a.writeDebugLogs(ctx, prepulledImageMessage)
			}

			if a.isGenericErrorPresent(ctx, consoleLogs) {
				a.writeDebugLogs(ctx, "generic error found - please help us classify this by sharing it with us so that we can provide a more specific error message")
			}

			// If debug logging is enabled, consoleOutput the full console log that appears to include the full userdata run
			a.writeDebugLogs(ctx, fmt.Sprintf("base64-encoded console logs:\n---\n%s\n---", b64ConsoleLogs))

			if a.isEgressFailurePresent(string(scriptOutput)) {
				a.writeDebugLogs(ctx, "egress failures found")
			}

			return true, nil // finalize as there's `userdata end`
		}

		if len(b64ConsoleLogs) > 0 {
			a.writeDebugLogs(ctx, fmt.Sprintf("base64-encoded console logs:\n---\n%s\n---", b64ConsoleLogs))
		}

		return false, nil
	})

	return err
}

// isGenericErrorPresent checks consoleOutput for generic (unclassified) failures
func (a *AwsVerifier) isGenericErrorPresent(ctx context.Context, consoleOutput string) bool {
	// reGenericFailure is an attempt at a catch-all to help debug failures that we have not accounted for yet
	reGenericFailure := regexp.MustCompile(`(?m)^(.*Cannot.*)|(.*Could not.*)|(.*Failed.*)|(.*command not found.*)`)
	// reRetryAttempt will override reGenericFailure when matching against attempts to retry pulling a container image
	reRetryAttempt := regexp.MustCompile(`Failed, retrying in`)

	found := false

	genericFailures := reGenericFailure.FindAllString(consoleOutput, -1)
	if len(genericFailures) > 0 {
		for _, failure := range genericFailures {
			switch {
			// Ignore "Failed, retrying in" messages when retrying container image pulls as they are not terminal failures
			case reRetryAttempt.FindAllString(failure, -1) != nil:
				a.writeDebugLogs(ctx, fmt.Sprintf("ignoring failure that is retrying: %s", failure))
			// If we don't otherwise ignore a generic error, consider it one that needs attention
			default:
				a.Output.AddError(handledErrors.NewGenericError(errors.New(failure)))
				found = true
			}
		}
	}

	return found
}

// isEgressFailurePresent checks consoleOutput for network egress failures and stores them
// as NetworkVerifierErrors in a.Output.failures
func (a *AwsVerifier) isEgressFailurePresent(consoleOutput string) bool {
	// reEgressFailures will match a specific egress failure case
	reEgressFailures := regexp.MustCompile(`Unable to reach (\S+)`)
	found := false

	// egressFailures is a 2D slice of regex matches - egressFailures[0] represents a specific regex match
	// egressFailures[0][0] is the "Unable to reach" part of the match
	// egressFailures[0][1] is the "(\S+)" part of the match, i.e. the following string
	egressFailures := reEgressFailures.FindAllStringSubmatch(consoleOutput, -1)
	for _, e := range egressFailures {
		if len(e) == 2 {
			a.Output.SetEgressFailures([]string{e[1]})
			found = true
		}
	}

	return found
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

func generateUserData(variables map[string]string) (string, error) {
	const maxDataSize = 16 * 1024 // 16KB

	variableMapper := func(varName string) string {
		return variables[varName]
	}
	data := os.Expand(helpers.UserdataTemplate, variableMapper)

	// Convert data to a byte slice and check its length
	// User data is limited to 16 KB, in raw form, before it is base64-encoded
	dataBytes := []byte(data)
	if len(dataBytes) > maxDataSize {
		return "", fmt.Errorf("userData size exceeds the maximum limit of 16KB, if you used '--cacert', please check the cacert file size")
	}

	return base64.StdEncoding.EncodeToString([]byte(data)), nil
}

// setCloudImage returns a default AMI ID based on the region if one is not provided
func setCloudImage(cloudImageID *string, region string) error {
	if *cloudImageID == "" {
		*cloudImageID = defaultAmi[region]
		if *cloudImageID == "" {
			return fmt.Errorf("no default ami found for region %s ", region)
		}
	}

	return nil
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
		GroupId: output.GroupId,
		IpPermissions: []ec2Types.IpPermission{
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
		},
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
