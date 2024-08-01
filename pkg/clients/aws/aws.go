package aws

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	handledErrors "github.com/openshift/osd-network-verifier/pkg/errors"
)

// Client represents an AWS Client
// For mocking the whole aws client, use the following:
// mockgen -source=pkg/clients/aws/aws.go -package mocks -destination=pkg/mocks/mock_aws.go
type Client struct {
	ec2Client EC2Client
	Region    string
}

func (c *Client) SetClient(e EC2Client) {
	c.ec2Client = e
}

// NewClientFromConfig creates an osd-network-verifier AWS Client from an aws-sdk-go-v2 Config
func NewClientFromConfig(cfg aws.Config) (*Client, error) {
	return &Client{
		ec2Client: ec2.NewFromConfig(cfg),
		Region:    cfg.Region,
	}, nil
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
	RunInstances(ctx context.Context, params *ec2.RunInstancesInput, optFns ...func(*ec2.Options)) (*ec2.RunInstancesOutput, error)
	DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
	DescribeInstanceTypes(ctx context.Context, input *ec2.DescribeInstanceTypesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceTypesOutput, error)
	GetConsoleOutput(ctx context.Context, input *ec2.GetConsoleOutputInput, optFns ...func(*ec2.Options)) (*ec2.GetConsoleOutputOutput, error)
	TerminateInstances(ctx context.Context, input *ec2.TerminateInstancesInput, optFns ...func(*ec2.Options)) (*ec2.TerminateInstancesOutput, error)
	DescribeVpcAttribute(ctx context.Context, input *ec2.DescribeVpcAttributeInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcAttributeOutput, error)
	CreateSecurityGroup(ctx context.Context, params *ec2.CreateSecurityGroupInput, optFns ...func(*ec2.Options)) (*ec2.CreateSecurityGroupOutput, error)
	DeleteSecurityGroup(ctx context.Context, params *ec2.DeleteSecurityGroupInput, optFns ...func(*ec2.Options)) (*ec2.DeleteSecurityGroupOutput, error)
	DescribeSecurityGroups(ctx context.Context, params *ec2.DescribeSecurityGroupsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error)
	AuthorizeSecurityGroupEgress(ctx context.Context, params *ec2.AuthorizeSecurityGroupEgressInput, optFns ...func(*ec2.Options)) (*ec2.AuthorizeSecurityGroupEgressOutput, error)
	RevokeSecurityGroupEgress(ctx context.Context, params *ec2.RevokeSecurityGroupEgressInput, optFns ...func(*ec2.Options)) (*ec2.RevokeSecurityGroupEgressOutput, error)
	DescribeSubnets(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error)
	ImportKeyPair(ctx context.Context, params *ec2.ImportKeyPairInput, optFns ...func(*ec2.Options)) (*ec2.ImportKeyPairOutput, error)
	DeleteKeyPair(ctx context.Context, params *ec2.DeleteKeyPairInput, optFns ...func(*ec2.Options)) (*ec2.DeleteKeyPairOutput, error)
	DescribeKeyPairs(ctx context.Context, params *ec2.DescribeKeyPairsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeKeyPairsOutput, error)
	ModifyInstanceAttribute(ctx context.Context, params *ec2.ModifyInstanceAttributeInput, optFns ...func(*ec2.Options)) (*ec2.ModifyInstanceAttributeOutput, error)
}

func (c *Client) DescribeKeyPairs(ctx context.Context, params *ec2.DescribeKeyPairsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeKeyPairsOutput, error) {
	return c.ec2Client.DescribeKeyPairs(ctx, params, optFns...)
}

func (c *Client) ModifyInstanceAttribute(ctx context.Context, params *ec2.ModifyInstanceAttributeInput, optFns ...func(*ec2.Options)) (*ec2.ModifyInstanceAttributeOutput, error) {
	return c.ec2Client.ModifyInstanceAttribute(ctx, params, optFns...)
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

func (c *Client) CreateSecurityGroup(ctx context.Context, params *ec2.CreateSecurityGroupInput, optFns ...func(*ec2.Options)) (*ec2.CreateSecurityGroupOutput, error) {
	return c.ec2Client.CreateSecurityGroup(ctx, params, optFns...)
}

func (c *Client) DescribeSubnets(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
	return c.ec2Client.DescribeSubnets(ctx, params, optFns...)
}

func (c *Client) DeleteSecurityGroup(ctx context.Context, params *ec2.DeleteSecurityGroupInput, optFns ...func(*ec2.Options)) (*ec2.DeleteSecurityGroupOutput, error) {
	return c.ec2Client.DeleteSecurityGroup(ctx, params, optFns...)
}

func (c *Client) DescribeSecurityGroups(ctx context.Context, params *ec2.DescribeSecurityGroupsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error) {
	return c.ec2Client.DescribeSecurityGroups(ctx, params, optFns...)
}

func (c *Client) AuthorizeSecurityGroupEgress(ctx context.Context, params *ec2.AuthorizeSecurityGroupEgressInput, optFns ...func(*ec2.Options)) (*ec2.AuthorizeSecurityGroupEgressOutput, error) {
	return c.ec2Client.AuthorizeSecurityGroupEgress(ctx, params, optFns...)
}

func (c *Client) RevokeSecurityGroupEgress(ctx context.Context, params *ec2.RevokeSecurityGroupEgressInput, optFns ...func(*ec2.Options)) (*ec2.RevokeSecurityGroupEgressOutput, error) {
	return c.ec2Client.RevokeSecurityGroupEgress(ctx, params, optFns...)
}

func (c *Client) ImportKeyPair(ctx context.Context, params *ec2.ImportKeyPairInput, optFns ...func(*ec2.Options)) (*ec2.ImportKeyPairOutput, error) {
	return c.ec2Client.ImportKeyPair(ctx, params, optFns...)
}

func (c *Client) DeleteKeyPair(ctx context.Context, params *ec2.DeleteKeyPairInput, optFns ...func(*ec2.Options)) (*ec2.DeleteKeyPairOutput, error) {
	return c.ec2Client.DeleteKeyPair(ctx, params, optFns...)
}

// TerminateEC2Instance terminates target ec2 instance
func (c *Client) TerminateEC2Instance(ctx context.Context, instanceID string) error {
	input := ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	}
	if _, err := c.ec2Client.TerminateInstances(ctx, &input); err != nil {
		return handledErrors.NewGenericError(err)
	}

	// Wait up to 5 minutes for the instance to be terminated, using a lower
	// MinDelay than the default 15s so that we don't wait unnecessarily
	reduceMinDelay := func(i *ec2.InstanceTerminatedWaiterOptions) {
		i.MinDelay = 3 * time.Second
	}
	waiter := ec2.NewInstanceTerminatedWaiter(c)
	if err := waiter.Wait(ctx, &ec2.DescribeInstancesInput{InstanceIds: []string{instanceID}}, 5*time.Minute, reduceMinDelay); err != nil {
		return handledErrors.NewGenericError(err)
	}

	return nil
}
