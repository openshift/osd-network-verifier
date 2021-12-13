package aws

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go/aws/awserr"
	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	instanceType  string = "t2.micro"
	instanceCount int    = 1
	defaultAmi           = map[string]string{
		// using Amazon Linux 2 AMI (HVM) - Kernel 5.10
		"us-east-1":      "ami-0ed9277fb7eb570c",
		"us-east-2":      "ami-002068ed284fb165b",
		"us-west-1":      "ami-03af6a70ccd8cb578",
		"us-west-2":      "ami-00f7e5c52c0f43726",
		"ca-central-1":   "ami-0bae7412735610274",
		"eu-north-1":     "ami-06bfd6343550d4a29",
		"eu-central-1":   "ami-05d34d340fb1d89e5",
		"eu-west-1":      "ami-04dd4500af104442f",
		"eu-west-2":      "ami-0d37e07bd4ff37148",
		"eu-west-3":      "ami-0d3c032f5934e1b41",
		"eu-south-1":     "",
		"ap-northeast-1": "",
		"ap-northeast-2": "",
		"ap-northeast-3": "",
		"ap-east-1":      "",
		"ap-south-1":     "",
		"ap-southeast-1": "",
		"ap-southeast-2": "",
		"sa-east-1":      "",
		"af-south-1":     "",
		"me-south-1":     "",
	}
)

func newClient(ctx context.Context, logger ocmlog.Logger, accessID, accessSecret, sessiontoken, region string, tags map[string]string) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID: accessID, SecretAccessKey: accessSecret, SessionToken: sessiontoken,
			},
		}),
	)
	if err != nil {
		return nil, err
	}

	return &Client{
		ec2Client: ec2.NewFromConfig(cfg),
		region:    region,
		tags:      tags,
		logger:    logger,
	}, nil
}

func buildTags(tags map[string]string) []ec2Types.TagSpecification {
	tagList := []ec2Types.Tag{}
	for k, v := range tags {
		t := ec2Types.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		}
		tagList = append(tagList, t)
	}

	tagSpec := ec2Types.TagSpecification{
		ResourceType: ec2Types.ResourceTypeInstance,
		Tags:         tagList,
	}

	return []ec2Types.TagSpecification{tagSpec}
}

func (c Client) createEC2Instance(ctx context.Context, amiID, instanceType string, instanceCount int, vpcSubnetID, userdata string, tags map[string]string) (ec2.RunInstancesOutput, error) {
	// Build our request, converting the go base types into the pointers required by the SDK
	instanceReq := ec2.RunInstancesInput{
		ImageId:      aws.String(amiID),
		MaxCount:     aws.Int32(int32(instanceCount)),
		MinCount:     aws.Int32(int32(instanceCount)),
		InstanceType: ec2Types.InstanceType(instanceType),
		// Because we're making this VPC aware, we also have to include a network interface specification
		NetworkInterfaces: []ec2Types.InstanceNetworkInterfaceSpecification{
			{
				AssociatePublicIpAddress: aws.Bool(true),
				DeviceIndex:              aws.Int32(0),
				SubnetId:                 aws.String(vpcSubnetID),
			},
		},
		UserData:          aws.String(userdata),
		TagSpecifications: buildTags(tags),
	}
	// Finally, we make our request
	instanceResp, err := c.ec2Client.RunInstances(ctx, &instanceReq)
	if err != nil {
		return ec2.RunInstancesOutput{}, err
	}

	for _, i := range instanceResp.Instances {
		c.logger.Info(ctx, "Created instance with ID: %s", *i.InstanceId)
	}

	return *instanceResp, nil
}

// Returns state code as int
func (c Client) describeEC2Instances(ctx context.Context, instanceID string) (int, error) {
	// States and codes
	// 0 : pending
	// 16 : running
	// 32 : shutting-down
	// 48 : terminated
	// 64 : stopping
	// 80 : stopped
	// 401 : failed
	result, err := c.ec2Client.DescribeInstanceStatus(ctx, &ec2.DescribeInstanceStatusInput{
		InstanceIds: []string{instanceID},
	})

	if err != nil {
		c.logger.Error(ctx, "Errors while describing the instance status: %s", err.Error())
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == "UnauthorizedOperation" {
				return 401, err
			}
		}
		return 0, err
	}

	if len(result.InstanceStatuses) > 1 {
		return 0, errors.New("more than one EC2 instance found")
	}

	if len(result.InstanceStatuses) == 0 {
		// Don't return an error here as if the instance is still too new, it may not be
		// returned at all.
		//return 0, errors.New("no EC2 instances found")
		c.logger.Debug(ctx, "Instance %s has no status yet", instanceID)
		return 0, nil
	}

	return int(*result.InstanceStatuses[0].InstanceState.Code), nil
}

func (c Client) waitForEC2InstanceCompletion(ctx context.Context, instanceID string) error {
	//wait for the instance to run
	totalWait := 25 * 60
	currentWait := 1
	// Double the wait time until we reach totalWait seconds
	for totalWait > 0 {
		currentWait = currentWait * 2
		if currentWait > totalWait {
			currentWait = totalWait
		}
		totalWait -= currentWait
		time.Sleep(time.Duration(currentWait) * time.Second)
		code, descError := c.describeEC2Instances(ctx, instanceID)
		if code == 16 { // 16 represents a successful region initialization
			// Instance is running, break
			break
		} else if code == 401 { // 401 represents an UnauthorizedOperation error
			// Missing permission to perform operations, account needs to fail
			return fmt.Errorf("missing required permissions for account: %s", descError)
		}

		if descError != nil {
			// Log an error and make sure that instance is terminated
			descErrorMsg := fmt.Sprintf("Could not get EC2 instance state, terminating instance %s", instanceID)

			if descError, ok := descError.(awserr.Error); ok {
				descErrorMsg = fmt.Sprintf("Could not get EC2 instance state: %s, terminating instance %s", descError.Code(), instanceID)
			}

			return fmt.Errorf("%s: %s", descError, descErrorMsg)
		}
	}

	c.logger.Info(ctx, "EC2 Instance: %s Running", instanceID)
	return nil
}

func generateUserData() (string, error) {
	var data strings.Builder
	data.Grow(351)
	data.WriteString("#!/bin/bash -xe\n")
	data.WriteString("exec > >(tee /var/log/user-data.log|logger -t user-data -s 2>/dev/console) 2>&1\n")

	data.WriteString(`echo "USERDATA BEGIN"` + "\n")
	data.WriteString("sudo yum update -y\n")
	data.WriteString("sudo amazon-linux-extras install docker\n")
	data.WriteString("sudo service docker start\n")
	// TODO find a location for future docker images
	data.WriteString("sudo docker pull docker.io/tiwillia/network-validator-test:v0.1\n")
	data.WriteString("sudo docker run docker.io/tiwillia/network-validator-test:v0.1\n")
	data.WriteString(`echo "USERDATA END"` + "\n")

	userData := base64.StdEncoding.EncodeToString([]byte(data.String()))

	return userData, nil
}

func (c Client) findUnreachableEndpoints(ctx context.Context, instanceID string) ([]string, error) {
	var match []string

	err := wait.PollImmediate(30*time.Second, 10*time.Minute, func() (bool, error) {
		output, err := c.ec2Client.GetConsoleOutput(ctx, &ec2.GetConsoleOutputInput{InstanceId: &instanceID})
		if err == nil && output.Output != nil {
			// Find unreachable targets from output
			scriptOutput, err := base64.StdEncoding.DecodeString(*output.Output)
			if err != nil {
				// unable to decode output. we will try again
				return false, nil
			}
			re := regexp.MustCompile(`Unable to reach (\S+)`)
			match = re.FindAllString(string(scriptOutput), -1)

			return true, nil
		}
		c.logger.Debug(ctx, "waiting for UserData script to complete")
		return false, nil
	})
	return match, err
}

func (c Client) terminateEC2Instance(ctx context.Context, instanceID string) error {
	input := ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	}
	_, err := c.ec2Client.TerminateInstances(ctx, &input)
	if err != nil {
		c.logger.Error(ctx, "Unable to terminate EC2 instance: %s", err.Error())
		return err
	}

	return nil
}

func (c *Client) validateEgress(ctx context.Context, vpcSubnetID, cloudImageID string) error {
	// Generate the userData file
	userData, err := generateUserData()
	if err != nil {
		err = fmt.Errorf("Unable to generate UserData file: %s", err.Error())
		return err
	}
	// If a cloud image wasn't provided by the caller,
	if cloudImageID == "" {
		// use defaultAmi for the region instead
		cloudImageID = defaultAmi[c.region]

		if cloudImageID == "" {
			return fmt.Errorf("No default AMI found for region %s ", c.region)
		}
	}

	// Create an ec2 instance
	instance, err := c.createEC2Instance(ctx, cloudImageID, instanceType, instanceCount, vpcSubnetID, userData, c.tags)
	if err != nil {
		err = fmt.Errorf("Unable to create EC2 Instance: %s", err.Error())
		return err
	}
	instanceID := *instance.Instances[0].InstanceId

	// Wait for the ec2 instance to be running
	c.logger.Debug(ctx, "Waiting for EC2 instance %s to be running", instanceID)
	err = c.waitForEC2InstanceCompletion(ctx, instanceID)
	if err != nil {
		err = fmt.Errorf("Error while waiting for EC2 instance to start: %s", err.Error())
		return err
	}
	c.logger.Info(ctx, "Gathering and parsing console log output...")
	unreachableEndpoints, err := c.findUnreachableEndpoints(ctx, instanceID)
	if err != nil {
		c.logger.Error(ctx, "Error parsing output from console log: %s", err.Error())
		return err
	}

	c.logger.Info(ctx, "Terminating ec2 instance with id %s", instanceID)
	if err := c.terminateEC2Instance(ctx, instanceID); err != nil {
		err = fmt.Errorf("Error terminating instances: %s", err.Error())
		return err
	}
	if len(unreachableEndpoints) > 0 {
		return fmt.Errorf("multiple targets unreachable %q", unreachableEndpoints)
	}
	return nil
}
