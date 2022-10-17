package aws

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	handledErrors "github.com/openshift/osd-network-verifier/pkg/errors"
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
type EC2Client interface {
	CreateTags(ctx context.Context, params *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error)
	RunInstances(ctx context.Context, params *ec2.RunInstancesInput, optFns ...func(*ec2.Options)) (*ec2.RunInstancesOutput, error)
	DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
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

func (c *Client) DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	return c.ec2Client.DescribeInstances(ctx, params, optFns...)
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

// TerminateEC2Instance terminates target ec2 instance
func (c *Client) TerminateEC2Instance(ctx context.Context, instanceID string) error {
	input := ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	}
	if _, err := c.ec2Client.TerminateInstances(ctx, &input); err != nil {
		return handledErrors.NewGenericError(err)
	}

	return nil
}
