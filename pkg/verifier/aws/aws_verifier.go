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
		// using AMI from
		"af-south-1":     "ami-0305ce24a63f7cd96",
		"ap-east-1":      "ami-04b0c3f978c805497",
		"ap-northeast-1": "ami-0f36dc8565e1204ac",
		"ap-northeast-2": "ami-00e55c924048d51cd",
		"ap-northeast-3": "ami-092632c2d4888ee15",
		"ap-south-1":     "ami-027ee3c5ed1f1fbfc",
		"ap-southeast-1": "ami-09f43282cd35a5b53",
		"ap-southeast-2": "ami-0eb1973086a7b8a1a",
		"ca-central-1":   "ami-08dc2cc48baa4a493",
		"eu-central-1":   "ami-0a520b55e97ca808c",
		"eu-north-1":     "ami-0d6c03859f2d5ba76",
		"eu-south-1":     "ami-0af4bdc3e6f25374f",
		"eu-west-1":      "ami-0949e6f98fdcc8a48",
		"eu-west-2":      "ami-05af13545b8dcf09d",
		"eu-west-3":      "ami-099c6b480ddecfa28",
		"me-south-1":     "ami-08348a910dc888949",
		"sa-east-1":      "ami-0e1e7df70438a9e28",
		"us-east-1":      "ami-091db60579967890f",
		"us-east-2":      "ami-09d6a8053437e16bf",
		"us-west-1":      "ami-0cacfe7d77039ede2",
		"us-west-2":      "ami-03ab344882b539e44",
	}
)

const (
	instanceCount int32 = 1

	// TODO find a location for future docker images
	networkValidatorImage = "quay.io/app-sre/osd-network-verifier:v0.1.212-5f88b83"
	userdataEndVerifier   = "USERDATA END"
)

// AwsVerifier holds an aws client and knows how to fuifill the VerifierSerice which contains all functions needed for verifier
type AwsVerifier struct {
	AwsClient aws.Client
	Logger    ocmlog.Logger
	Output    output.Output
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

	return &AwsVerifier{*awsClient, logger, output.Output{}}, nil
}

func (a *AwsVerifier) validateInstanceType(ctx context.Context, instanceType string) error {
	descInput := ec2.DescribeInstanceTypesInput{
		InstanceTypes: []ec2Types.InstanceType{ec2Types.InstanceType(instanceType)},
	}

	a.writeDebugLogs(fmt.Sprintf("Gathering description of instance type %s from EC2", instanceType))
	descOut, err := a.AwsClient.DescribeInstanceTypes(ctx, &descInput)
	if err != nil {
		return handledErrors.NewGenericError(err)
	}

	// Effectively guaranteed to only have one match since we are casting c.instanceType into ec2Types.InstanceType
	// and placing it as the only InstanceType filter. Otherwise, ec2:DescribeInstanceTypes also accepts multiple as
	// an array of InstanceTypes which could return multiple matches.
	if len(descOut.InstanceTypes) != 1 {
		a.writeDebugLogs(fmt.Sprintf("matched instance types: %v", descOut.InstanceTypes))
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
		// Because we're making this VPC aware, we also have to include a network interface specification
		NetworkInterfaces: []ec2Types.InstanceNetworkInterfaceSpecification{eniSpecification},
		// We specify block devices mainly to enable EBS encryption
		BlockDeviceMappings: []ec2Types.BlockDeviceMapping{
			{
				DeviceName: awsTools.String("/dev/xvda"),
				Ebs:        ebsBlockDevice,
			},
		},
		UserData: awsTools.String(input.userdata),
	}
	// Finally, we make our request
	instanceResp, err := a.AwsClient.RunInstances(input.ctx, &instanceReq)
	if err != nil {
		return "", handledErrors.NewGenericError(err)
	}

	for _, i := range instanceResp.Instances {
		a.Logger.Info(context.TODO(), "Created instance with ID: %s", *i.InstanceId)
	}

	if len(instanceResp.Instances) == 0 {
		// Shouldn't happen, but ensure safety of the following logic
		return "", handledErrors.NewGenericError(errors.New("unexpectedly found 0 instances after creation, please try again"))
	}

	instanceID := *instanceResp.Instances[0].InstanceId
	if err := a.createTags(input.tags, instanceID); err != nil {
		// Unable to tag the instance
		return "", handledErrors.NewGenericError(err)
	}

	return instanceID, nil
}

func (a *AwsVerifier) findUnreachableEndpoints(ctx context.Context, instanceID string) error {
	var (
		b64ConsoleLogs string
		consoleLogs    string
	)
	// Compile the regular expressions once
	reUserDataComplete := regexp.MustCompile(userdataEndVerifier)
	reUnreachableErrors := regexp.MustCompile(`Unable to reach (\S+)`)
	reGenericFailure := regexp.MustCompile(`(?m)^(.*Cannot.*)|(.*Could not.*)|(.*Failed.*)|(.*command not found.*)`)
	reDockerFailure := regexp.MustCompile(`(?m)(docker)`)

	input := &ec2.GetConsoleOutputInput{
		InstanceId: awsTools.String(instanceID),
		Latest:     awsTools.Bool(true),
	}

	a.writeDebugLogs("Scraping console output and waiting for user data script to complete...")

	// Periodically scrape console output and analyze the logs for any errors or a successful completion
	err := helpers.PollImmediate(30*time.Second, 4*time.Minute, func() (bool, error) {
		consoleOutput, err := a.AwsClient.GetConsoleOutput(ctx, input)
		if err != nil {
			return false, handledErrors.NewGenericError(err)
		}

		if consoleOutput.Output != nil {
			// In the early stages, an ec2 instance may be running but the console is not populated with any data
			if len(*consoleOutput.Output) == 0 {
				a.writeDebugLogs("EC2 console consoleOutput not yet populated with data, continuing to wait...")
				return false, nil
			}

			// Store base64-encoded output for debug logs
			b64ConsoleLogs = *consoleOutput.Output

			// The console consoleOutput starts out base64 encoded
			scriptOutput, err := base64.StdEncoding.DecodeString(*consoleOutput.Output)
			if err != nil {
				a.writeDebugLogs(fmt.Sprintf("Error decoding console consoleOutput, will retry on next check interval: %s", err))
				return false, nil
			}

			consoleLogs = string(scriptOutput)

			// Check for the specific string we consoleOutput in the generated userdata file at the end to verify the userdata script has run
			// It is possible we get EC2 console consoleOutput, but the userdata script has not yet completed.
			userDataComplete := reUserDataComplete.FindString(consoleLogs)
			if len(userDataComplete) < 1 {
				a.writeDebugLogs("EC2 console consoleOutput contains data, but end of userdata script not seen, continuing to wait...")
				return false, nil
			}

			// Check consoleOutput for failures, report as exceptions if they occurred
			genericFailures := reGenericFailure.FindAllStringSubmatch(consoleLogs, -1)
			if len(genericFailures) > 0 {
				a.writeDebugLogs(fmt.Sprint(genericFailures))

				dockerFailures := reDockerFailure.FindAllString(consoleLogs, -1)
				if len(dockerFailures) > 0 {
					// Should be resolved by OSD-13003 and OSD-13007
					a.Output.AddException(handledErrors.NewGenericError(errors.New("docker was unable to install or run. Further investigation needed")))
					a.Output.AddError(handledErrors.NewGenericError(fmt.Errorf("%v", dockerFailures)))
				} else {
					// TODO: Flesh out generic issues, for now we only know about Docker
					a.Output.AddException(handledErrors.NewGenericError(errors.New("egress tests were not run due to an uncaught error in setup or execution. Further investigation needed")))
					a.Output.AddError(handledErrors.NewGenericError(fmt.Errorf("%v", genericFailures)))
				}
			}

			// If debug logging is enabled, consoleOutput the full console log that appears to include the full userdata run
			a.writeDebugLogs(fmt.Sprintf("base64-encoded console logs:\n---\n%s\n---", b64ConsoleLogs))

			a.Output.SetEgressFailures(reUnreachableErrors.FindAllString(string(scriptOutput), -1))
			return true, nil
		}

		if len(b64ConsoleLogs) > 0 {
			a.writeDebugLogs(fmt.Sprintf("base64-encoded console logs:\n---\n%s\n---", b64ConsoleLogs))
		}

		return false, nil
	})

	return err
}

func (a *AwsVerifier) createTags(tags map[string]string, ids ...string) error {
	if len(tags) <= 0 {
		return nil
	}
	_, err := a.AwsClient.CreateTags(context.TODO(), &ec2.CreateTagsInput{
		Resources: ids,
		Tags:      buildTags(tags),
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

func generateUserData(variables map[string]string) (string, error) {
	variableMapper := func(varName string) string {
		return variables[varName]
	}
	data := os.Expand(helpers.UserdataTemplate, variableMapper)

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

func (a *AwsVerifier) writeDebugLogs(log string) {
	a.Output.AddDebugLogs(log)
	a.Logger.Debug(context.TODO(), log)
}
