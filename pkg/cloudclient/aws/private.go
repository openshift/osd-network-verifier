package aws

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	handledErrors "github.com/openshift/osd-network-verifier/pkg/errors"
	"github.com/openshift/osd-network-verifier/pkg/helpers"
	"github.com/openshift/osd-network-verifier/pkg/output"
	"github.com/openshift/osd-network-verifier/pkg/proxy"
)

type createEC2InstanceInput struct {
	amiID         string
	vpcSubnetID   string
	userdata      string
	ebsKmsKeyID   string
	instanceCount int32
}

const (
	instanceCount int32 = 1

	// TODO find a location for future docker images
	networkValidatorImage = "quay.io/app-sre/osd-network-verifier:v0.1.212-5f88b83"
	userdataEndVerifier   = "USERDATA END"
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

func newClient(ctx context.Context, logger ocmlog.Logger, accessID, accessSecret, sessiontoken, region,
	instanceType string, tags map[string]string, profile string) (*Client, error) {
	var cfg aws.Config
	var err error
	if profile != "" {
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithSharedConfigProfile(profile),
			config.WithRegion(region),
		)
	} else {
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
			config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
				Value: aws.Credentials{
					AccessKeyID: accessID, SecretAccessKey: accessSecret, SessionToken: sessiontoken,
				},
			}),
		)
	}
	if err != nil {
		return nil, err
	}

	c := &Client{
		ec2Client:    ec2.NewFromConfig(cfg),
		region:       region,
		instanceType: instanceType,
		tags:         tags,
		logger:       logger,
		output:       output.Output{},
	}

	// Validates the provided instance type will work with the verifier
	// NOTE a "nitro" EC2 instance type is required to be used
	if err := c.validateInstanceType(ctx); err != nil {
		return nil, err
	}

	return c, nil
}

func buildTags(tags map[string]string) []ec2Types.Tag {
	tagList := make([]ec2Types.Tag, 0, len(tags))
	for k, v := range tags {
		t := ec2Types.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		}
		tagList = append(tagList, t)
	}

	return tagList
}

func (c *Client) validateInstanceType(ctx context.Context) error {
	descInput := ec2.DescribeInstanceTypesInput{
		InstanceTypes: []ec2Types.InstanceType{ec2Types.InstanceType(c.instanceType)},
	}

	c.logger.Debug(ctx, "Gathering description of instance type %s from EC2", c.instanceType)
	descOut, err := c.ec2Client.DescribeInstanceTypes(ctx, &descInput)
	if err != nil {
		return handledErrors.NewGenericError(err)
	}

	// Effectively guaranteed to only have one match since we are casting c.instanceType into ec2Types.InstanceType
	// and placing it as the only InstanceType filter. Otherwise, ec2:DescribeInstanceTypes also accepts multiple as
	// an array of InstanceTypes which could return multiple matches.
	if len(descOut.InstanceTypes) != 1 {
		c.logger.Debug(ctx, "matched instance types", descOut.InstanceTypes)
		return fmt.Errorf("expected one instance type match for %s, got %d", c.instanceType, len(descOut.InstanceTypes))
	}

	if string(descOut.InstanceTypes[0].InstanceType) == c.instanceType {
		if descOut.InstanceTypes[0].Hypervisor != ec2Types.InstanceTypeHypervisorNitro {
			return fmt.Errorf("instance type %s must use hypervisor type 'nitro' to support reliable result collection, using %s", c.instanceType, descOut.InstanceTypes[0].Hypervisor)
		}
	}

	return nil
}

// createEC2Instance attempts to create a single EC2 instance, tags it, and returns its id
func (c *Client) createEC2Instance(ctx context.Context, input *createEC2InstanceInput) (string, error) {
	ebsBlockDevice := &ec2Types.EbsBlockDevice{
		DeleteOnTermination: aws.Bool(true),
		Encrypted:           aws.Bool(true),
	}
	// Check if KMS key was specified for root volume encryption
	if input.ebsKmsKeyID != "" {
		ebsBlockDevice.KmsKeyId = aws.String(input.ebsKmsKeyID)
	}

	// Build our request, converting the go base types into the pointers required by the SDK
	instanceReq := ec2.RunInstancesInput{
		ImageId:      aws.String(input.amiID),
		MaxCount:     aws.Int32(input.instanceCount),
		MinCount:     aws.Int32(input.instanceCount),
		InstanceType: ec2Types.InstanceType(c.instanceType),
		// Because we're making this VPC aware, we also have to include a network interface specification
		NetworkInterfaces: []ec2Types.InstanceNetworkInterfaceSpecification{
			{
				AssociatePublicIpAddress: aws.Bool(true),
				DeviceIndex:              aws.Int32(0),
				SubnetId:                 aws.String(input.vpcSubnetID),
			},
		},
		// We specify block devices mainly to enable EBS encryption
		BlockDeviceMappings: []ec2Types.BlockDeviceMapping{
			{
				DeviceName: aws.String("/dev/xvda"),
				Ebs:        ebsBlockDevice,
			},
		},
		UserData: aws.String(input.userdata),
	}
	// Finally, we make our request
	instanceResp, err := c.ec2Client.RunInstances(ctx, &instanceReq)
	if err != nil {
		return "", handledErrors.NewGenericError(err)
	}

	for _, i := range instanceResp.Instances {
		c.logger.Info(ctx, "Created instance with ID: %s", *i.InstanceId)
	}

	if len(instanceResp.Instances) == 0 {
		// Shouldn't happen, but ensure safety of the following logic
		return "", handledErrors.NewGenericError(errors.New("unexpectedly found 0 instances after creation, please try again"))
	}

	instanceID := *instanceResp.Instances[0].InstanceId
	if err := c.createTags(ctx, instanceID); err != nil {
		// Unable to tag the instance
		return "", handledErrors.NewGenericError(err)
	}

	return instanceID, nil
}

func (c *Client) createTags(ctx context.Context, ids ...string) error {
	_, err := c.ec2Client.CreateTags(ctx, &ec2.CreateTagsInput{
		Resources: ids,
		Tags:      buildTags(c.tags),
	})

	return err
}

// describeEC2Instances returns the instance state name of an EC2 instance
// States and codes
// 0 : pending
// 16 : running
// 32 : shutting-down
// 48 : terminated
// 64 : stopping
// 80 : stopped
// 401 : failed
func (c *Client) describeEC2Instances(ctx context.Context, instanceID string) (*ec2Types.InstanceStateName, error) {
	c.logger.Debug(ctx, "Describing state of EC2 instance %s", instanceID)

	result, err := c.ec2Client.DescribeInstanceStatus(ctx, &ec2.DescribeInstanceStatusInput{
		InstanceIds: []string{instanceID},
	})

	if err != nil {
		return nil, handledErrors.NewGenericError(err)
	}

	if len(result.InstanceStatuses) > 1 {
		// Shouldn't happen, since we're describing using an instance ID
		return nil, errors.New("more than one EC2 instance found")
	}

	if len(result.InstanceStatuses) == 0 {
		// Don't return an error here as if the instance is still too new, it may not be
		// returned at all.
		c.logger.Debug(ctx, "Instance %s has no status yet", instanceID)
		return nil, nil
	}

	return &result.InstanceStatuses[0].InstanceState.Name, nil
}

// waitForEC2InstanceCompletion checks every 15s for up to 2 minutes for an instance to be in the running state
func (c *Client) waitForEC2InstanceCompletion(ctx context.Context, instanceID string) error {
	c.logger.Debug(ctx, "Waiting for EC2 instance %s to be running", instanceID)

	return helpers.PollImmediate(15*time.Second, 2*time.Minute, func() (bool, error) {
		instanceState, descError := c.describeEC2Instances(ctx, instanceID)
		if descError != nil {
			return false, descError
		}

		if instanceState == nil {
			// A state is not populated yet, check again later
			return false, nil
		}

		switch *instanceState {
		case ec2Types.InstanceStateNameRunning:
			// Instance is running, we're done waiting
			c.logger.Info(ctx, "EC2 Instance: %s Running", instanceID)
			return true, nil
		default:
			// Otherwise, check again later
			return false, nil
		}
	})
}

func generateUserData(variables map[string]string) (string, error) {
	variableMapper := func(varName string) string {
		return variables[varName]
	}
	data := os.Expand(helpers.UserdataTemplate, variableMapper)

	return base64.StdEncoding.EncodeToString([]byte(data)), nil
}

func (c *Client) findUnreachableEndpoints(ctx context.Context, instanceID string) error {
	// Compile the regular expressions once
	reUserDataComplete := regexp.MustCompile(userdataEndVerifier)
	reUnreachableErrors := regexp.MustCompile(`Unable to reach (\S+)`)
	reGenericFailure := regexp.MustCompile(`(?m)^(.*Cannot.*)|(.*Could not.*)|(.*Failed.*)|(.*command not found.*)`)
	reDockerFailure := regexp.MustCompile(`(?m)(docker)`)

	input := &ec2.GetConsoleOutputInput{
		InstanceId: aws.String(instanceID),
		Latest:     aws.Bool(true),
	}

	c.logger.Debug(ctx, "Scraping console output and waiting for user data script to complete...")

	// Periodically scrape console output and analyze the logs for any errors or a successful completion
	err := helpers.PollImmediate(30*time.Second, 4*time.Minute, func() (bool, error) {
		output, err := c.ec2Client.GetConsoleOutput(ctx, input)
		if err != nil {
			return false, handledErrors.NewGenericError(err)
		}

		if output.Output != nil {
			// In the early stages, an ec2 instance may be running but the console is not populated with any data
			if len(*output.Output) == 0 {
				c.logger.Debug(ctx, "EC2 console output not yet populated with data, continuing to wait...")
				return false, nil
			}

			// The console output starts out base64 encoded
			scriptOutput, err := base64.StdEncoding.DecodeString(*output.Output)
			if err != nil {
				c.logger.Debug(ctx, "Error decoding console output, will retry on next check interval: %s", err)
				return false, nil
			}

			// Check for the specific string we output in the generated userdata file at the end to verify the userdata script has run
			// It is possible we get EC2 console output, but the userdata script has not yet completed.
			userDataComplete := reUserDataComplete.FindString(string(scriptOutput))
			if len(userDataComplete) < 1 {
				c.logger.Debug(ctx, "EC2 console output contains data, but end of userdata script not seen, continuing to wait...")
				return false, nil
			}

			// Check output for failures, report as exceptions if they occurred
			genericFailures := reGenericFailure.FindAllStringSubmatch(string(scriptOutput), -1)
			if len(genericFailures) > 0 {
				c.logger.Debug(ctx, fmt.Sprint(genericFailures))

				dockerFailures := reDockerFailure.FindAllString(string(scriptOutput), -1)
				if len(dockerFailures) > 0 {
					// Should be resolved by OSD-13003 and OSD-13007
					c.output.AddException(handledErrors.NewGenericError(errors.New("docker was unable to install or run. Further investigation needed")))
					c.output.AddError(handledErrors.NewGenericError(fmt.Errorf("%v", dockerFailures)))
				} else {
					// TODO: Flesh out generic issues, for now we only know about Docker
					c.output.AddException(handledErrors.NewGenericError(errors.New("egress tests were not run due to an uncaught error in setup or execution. Further investigation needed")))
					c.output.AddError(handledErrors.NewGenericError(fmt.Errorf("%v", genericFailures)))
				}
			}

			// If debug logging is enabled, output the full console log that appears to include the full userdata run
			c.logger.Debug(ctx, "Full EC2 console output:\n---\n%s\n---", scriptOutput)

			c.output.SetEgressFailures(reUnreachableErrors.FindAllString(string(scriptOutput), -1))
			return true, nil
		}

		return false, nil
	})

	return err
}

// terminateEC2Instance terminates target ec2 instance
// uses c.output to store result of the execution
func (c *Client) terminateEC2Instance(ctx context.Context, instanceID string) error {
	c.logger.Info(ctx, "Terminating ec2 instance with id %s", instanceID)
	input := ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	}
	if _, err := c.ec2Client.TerminateInstances(ctx, &input); err != nil {
		return handledErrors.NewGenericError(err)
	}

	return nil
}

// setCloudImage returns a default AMI ID based on the region if one is not provided
func (c *Client) setCloudImage(cloudImageID string) (string, error) {
	if cloudImageID == "" {
		cloudImageID = defaultAmi[c.region]
		if cloudImageID == "" {
			return "", fmt.Errorf("no default ami found for region %s, please specify one with `--image-id`", c.region)
		}
	}

	return cloudImageID, nil
}

// validateEgress performs validation process for egress
// Basic workflow is:
// - prepare for ec2 instance creation
// - create instance and wait till it gets ready, wait for userdata script execution
// - find unreachable endpoints & parse output, then terminate instance
// - return `c.output` which stores the execution results
func (c *Client) validateEgress(ctx context.Context, vpcSubnetID, cloudImageID string, kmsKeyID string, timeout time.Duration, p proxy.ProxyConfig) *output.Output {
	c.logger.Debug(ctx, "Using configured timeout of %s for each egress request", timeout.String())
	// Generate the userData file
	userDataVariables := map[string]string{
		"AWS_REGION":               c.region,
		"USERDATA_BEGIN":           "USERDATA BEGIN",
		"USERDATA_END":             userdataEndVerifier,
		"VALIDATOR_START_VERIFIER": "VALIDATOR START",
		"VALIDATOR_END_VERIFIER":   "VALIDATOR END",
		"VALIDATOR_IMAGE":          networkValidatorImage,
		"TIMEOUT":                  timeout.String(),
		"HTTP_PROXY":               p.HttpProxy,
		"HTTPS_PROXY":              p.HttpsProxy,
		"CACERT":                   base64.StdEncoding.EncodeToString([]byte(p.Cacert)),
		"NOTLS":                    strconv.FormatBool(p.NoTls),
	}
	userData, err := generateUserData(userDataVariables)
	if err != nil {
		return c.output.AddError(err)
	}
	c.logger.Debug(ctx, "Base64-encoded generated userdata script:\n---\n%s\n---", userData)

	cloudImageID, err = c.setCloudImage(cloudImageID)
	if err != nil {
		return c.output.AddError(err) // fatal
	}
	c.logger.Debug(ctx, "Using AMI: %s", cloudImageID)

	instanceID, err := c.createEC2Instance(ctx, &createEC2InstanceInput{
		amiID:         cloudImageID,
		vpcSubnetID:   vpcSubnetID,
		userdata:      userData,
		ebsKmsKeyID:   kmsKeyID,
		instanceCount: instanceCount,
	})
	if err != nil {
		return c.output.AddError(err) // fatal
	}

	if instanceReadyErr := c.waitForEC2InstanceCompletion(ctx, instanceID); instanceReadyErr != nil {
		// try to terminate the created instance
		if err := c.terminateEC2Instance(ctx, instanceID); err != nil {
			c.output.AddError(err)
		}
		return c.output.AddError(instanceReadyErr) // fatal
	}

	if err := c.findUnreachableEndpoints(ctx, instanceID); err != nil {
		c.output.AddError(err)
	}

	if err := c.terminateEC2Instance(ctx, instanceID); err != nil {
		c.output.AddError(err)
	}

	return &c.output
}

// verifyDns performs verification process for VPC's DNS
// Basic workflow is:
// - ask AWS API for VPC attributes
// - ensure they're set correctly
func (c *Client) verifyDns(ctx context.Context, vpcID string) *output.Output {
	c.logger.Info(ctx, "Verifying DNS config for VPC %s", vpcID)
	// Request boolean values from AWS API
	dnsSprtResult, err := c.ec2Client.DescribeVpcAttribute(ctx, &ec2.DescribeVpcAttributeInput{
		Attribute: ec2Types.VpcAttributeNameEnableDnsSupport,
		VpcId:     aws.String(vpcID),
	})
	if err != nil {
		c.output.AddError(handledErrors.NewGenericError(err))
		c.output.AddException(handledErrors.NewGenericError(
			fmt.Errorf("failed to validate the %s attribute on VPC: %s is true", ec2Types.VpcAttributeNameEnableDnsSupport, vpcID)),
		)
		return &c.output
	}

	dnsHostResult, err := c.ec2Client.DescribeVpcAttribute(ctx, &ec2.DescribeVpcAttributeInput{
		Attribute: ec2Types.VpcAttributeNameEnableDnsHostnames,
		VpcId:     aws.String(vpcID),
	})
	if err != nil {
		c.output.AddError(handledErrors.NewGenericError(err))
		c.output.AddException(handledErrors.NewGenericError(
			fmt.Errorf("failed to validate the %s attribute on VPC: %s is true", ec2Types.VpcAttributeNameEnableDnsHostnames, vpcID),
		))
		return &c.output
	}

	// Verify results
	c.logger.Info(ctx, "DNS Support for VPC %s: %t", vpcID, *dnsSprtResult.EnableDnsSupport.Value)
	c.logger.Info(ctx, "DNS Hostnames for VPC %s: %t", vpcID, *dnsHostResult.EnableDnsHostnames.Value)
	if !(*dnsSprtResult.EnableDnsSupport.Value) {
		c.output.AddException(handledErrors.NewGenericError(
			fmt.Errorf("the %s attribute on VPC: %s is %t, must be true", ec2Types.VpcAttributeNameEnableDnsSupport, vpcID, *dnsSprtResult.EnableDnsSupport.Value),
		))
	}

	if !(*dnsHostResult.EnableDnsHostnames.Value) {
		c.output.AddException(handledErrors.NewGenericError(
			fmt.Errorf("the %s attribute on VPC: %s is %t, must be true", ec2Types.VpcAttributeNameEnableDnsHostnames, vpcID, *dnsHostResult.EnableDnsHostnames.Value),
		))
	}

	return &c.output
}
