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
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	instanceType  string = "t2.micro"
	instanceCount int    = 1
)

func newClient(accessID, accessSecret, sessiontoken, region string) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
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
	}, nil
}

func createEC2Instance(ec2Client *ec2.Client, amiID, instanceType string, instanceCount int, vpcSubnetID, userdata string) (ec2.RunInstancesOutput, error) {
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
		UserData: aws.String(userdata),
	}
	// Finally, we make our request
	instanceResp, err := ec2Client.RunInstances(context.TODO(), &instanceReq)
	if err != nil {
		return ec2.RunInstancesOutput{}, err
	}

	for _, i := range instanceResp.Instances {
		fmt.Println("Created instance with ID:", *i.InstanceId)
	}

	return *instanceResp, nil
}

// Returns state code as int
func describeEC2Instances(client *ec2.Client, instanceID string) (int, error) {
	// States and codes
	// 0 : pending
	// 16 : running
	// 32 : shutting-down
	// 48 : terminated
	// 64 : stopping
	// 80 : stopped
	// 401 : failed
	result, err := client.DescribeInstanceStatus(context.TODO(), &ec2.DescribeInstanceStatusInput{
		InstanceIds: []string{instanceID},
	})

	if err != nil {
		fmt.Printf("Errors while describing the instance status: %s\n", err.Error())
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
		// retured at all.
		//return 0, errors.New("no EC2 instances found")
		fmt.Printf("Instance %s has no status yet\n", instanceID)
		return 0, nil
	}

	return int(*result.InstanceStatuses[0].InstanceState.Code), nil
}

func waitForEC2InstanceCompletion(ec2Client *ec2.Client, instanceID string) error {
	//wait for the instance to run
	var descError error
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
		var code int
		code, descError = describeEC2Instances(ec2Client, instanceID)
		if code == 16 { // 16 represents a successful region initialization
			// Instance is running, break
			break
		} else if code == 401 { // 401 represents an UnauthorizedOperation error
			// Missing permission to perform operations, account needs to fail
			return fmt.Errorf("Missing required permissions for account: %s", descError)
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

	fmt.Printf("EC2 Instance: %s Running\n", instanceID)
	return nil
}

func generateUserData() (string, error) {
	var data strings.Builder
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

func findUnreachableEndpoints(ec2Client *ec2.Client, instanceID string) ([]string, error) {
	var match []string

	err := wait.PollImmediate(30*time.Second, 10*time.Minute, func() (bool, error) {
		output, err := ec2Client.GetConsoleOutput(context.TODO(), &ec2.GetConsoleOutputInput{InstanceId: &instanceID})
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
		fmt.Print("waiting for UserData script to complete\n")
		return false, nil
	})
	return match, err
}

func terminateEC2Instance(ec2Client *ec2.Client, instanceID string) error {
	input := ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	}
	_, err := ec2Client.TerminateInstances(context.TODO(), &input)
	if err != nil {
		//log message saying there's been an error while Terminating ec2 instance
		return err
	}

	return nil
}

func (c *Client) validateEgress(ctx context.Context, vpcSubnetID, cloudImageID string) error {
	// Generate the userData file
	userData, err := generateUserData()
	if err != nil {
		panic(fmt.Sprintf("Unable to generate UserData file: %s\n", err.Error()))
	}

	// Create an ec2 instance
	instance, err := createEC2Instance(c.ec2Client, cloudImageID, instanceType, instanceCount, vpcSubnetID, userData)
	if err != nil {
		panic(fmt.Sprintf("Unable to create EC2 Instance: %s\n", err.Error()))
	}
	instanceID := *instance.Instances[0].InstanceId

	// Wait for the ec2 instance to be running
	fmt.Printf("Waiting for EC2 instance %s to be running\n", instanceID)
	err = waitForEC2InstanceCompletion(c.ec2Client, instanceID)
	if err != nil {
		panic(err)
	}
	fmt.Println("Gather and parse console log output")
	unreachableEndpoints, err := findUnreachableEndpoints(c.ec2Client, instanceID)
	if err != nil {
		panic(err)
	}

	fmt.Println("Terminating instance")
	if err := terminateEC2Instance(c.ec2Client, instanceID); err != nil {
		panic(err)
	}
	if len(unreachableEndpoints) > 0 {
		return errors.New(fmt.Sprintf("multiple target unreachable %q", unreachableEndpoints))
	}
	return nil
}
