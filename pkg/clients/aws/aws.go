package aws

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	handledErrors "github.com/openshift/osd-network-verifier/pkg/errors"
	"github.com/openshift/osd-network-verifier/pkg/helpers"
)

// Client represents an AWS Client
type Client struct {
	ec2Client EC2Client
	Region    string
}

// NewClient creates AWS Client either pass in secret data or profile to work .
func NewClient(ctx context.Context, accessID, accessSecret, sessiontoken, region, profile string) (*Client, error) {
	c := &Client{
		Region: region,
	}

	if profile != "" {
		cfg, err := config.LoadDefaultConfig(ctx,
			config.WithSharedConfigProfile(profile),
			config.WithRegion(region),
		)
		if err != nil {
			return &Client{}, err
		}
		c.ec2Client = ec2.NewFromConfig(cfg)
		return c, nil
	}

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID: accessID, SecretAccessKey: accessSecret, SessionToken: sessiontoken,
			},
		}),
	)
	if err != nil {
		return &Client{}, err
	}

	c.ec2Client = ec2.NewFromConfig(cfg)
	return c, nil
}

// Extend EC2Client so that we can mock them all for testing
// to re-generate mockfile once another interface is added for testing:
// mockgen -source=pkg/cloudclient/aws/aws.go -package mocks -destination=pkg/cloudclient/mocks/mock_aws.go
type EC2Client interface {
	CreateTags(ctx context.Context, params *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error)
	RunInstances(ctx context.Context, params *ec2.RunInstancesInput, optFns ...func(*ec2.Options)) (*ec2.RunInstancesOutput, error)
	DescribeInstanceStatus(ctx context.Context, input *ec2.DescribeInstanceStatusInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceStatusOutput, error)
	DescribeInstanceTypes(ctx context.Context, input *ec2.DescribeInstanceTypesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceTypesOutput, error)
	GetConsoleOutput(ctx context.Context, input *ec2.GetConsoleOutputInput, optFns ...func(*ec2.Options)) (*ec2.GetConsoleOutputOutput, error)
	TerminateInstances(ctx context.Context, input *ec2.TerminateInstancesInput, optFns ...func(*ec2.Options)) (*ec2.TerminateInstancesOutput, error)
	DescribeVpcAttribute(ctx context.Context, input *ec2.DescribeVpcAttributeInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcAttributeOutput, error)
}

func (c *Client) CreateTags(ctx context.Context, params *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error) {
	return c.ec2Client.CreateTags(ctx, params, optFns...)
}

func (c *Client) DescribeVpcAttribute(ctx context.Context, input *ec2.DescribeVpcAttributeInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcAttributeOutput, error) {
	return c.ec2Client.DescribeVpcAttribute(ctx, input, optFns...)
}

func (c *Client) RunInstances(ctx context.Context, params *ec2.RunInstancesInput, optFns ...func(*ec2.Options)) (*ec2.RunInstancesOutput, error) {
	return c.ec2Client.RunInstances(ctx, params, optFns...)
}
func (c *Client) DescribeInstanceTypes(ctx context.Context, input *ec2.DescribeInstanceTypesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceTypesOutput, error) {
	return c.ec2Client.DescribeInstanceTypes(ctx, input, optFns...)
}

func (c *Client) GetConsoleOutput(ctx context.Context, input *ec2.GetConsoleOutputInput, optFns ...func(*ec2.Options)) (*ec2.GetConsoleOutputOutput, error) {
	return c.ec2Client.GetConsoleOutput(ctx, input, optFns...)
}

// DescribeEC2Instances returns the instance state name of an EC2 instance
// States and codes
// 0 : pending
// 16 : running
// 32 : shutting-down
// 48 : terminated
// 64 : stopping
// 80 : stopped
// 401 : failed
func (c *Client) describeEC2Instances(ctx context.Context, instanceID string) (*ec2Types.InstanceStateName, error) {
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
		return nil, nil
	}

	return &result.InstanceStatuses[0].InstanceState.Name, nil
}

// waitForEC2InstanceCompletion checks every 15s for up to 2 minutes for an instance to be in the running state
func (c *Client) WaitForEC2InstanceCompletion(ctx context.Context, instanceID string) error {
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
			return true, nil
		default:
			// Otherwise, check again later
			return false, nil
		}
	})
}

// terminateEC2Instance terminates target ec2 instance
func (c *Client) TerminateEC2Instance(ctx context.Context, instanceID string) error {
	input := ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	}
	if _, err := c.ec2Client.TerminateInstances(ctx, &input); err != nil {
		return handledErrors.NewGenericError(err)
	}

	return nil
}
