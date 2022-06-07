package aws

import (
	"context"
	"fmt"
	"time"

	awscredsv2 "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	awscredsv1 "github.com/aws/aws-sdk-go/aws/credentials"
	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/output"
)

// ClientIdentifier is what kind of cloud this implement supports
const ClientIdentifier string = "AWS"

// Client represents an AWS Client
type Client struct {
	ec2Client    EC2Client
	region       string
	instanceType string
	tags         map[string]string
	logger       ocmlog.Logger
	output       output.Output
}

// Extend EC2Client so that we can mock them all for testing
// to re-generate mockfile once another interface is added for testing:
// mockgen -source=pkg/cloudclient/aws/aws.go -package mocks -destination=pkg/cloudclient/mocks/mock_aws.go
type EC2Client interface {
	RunInstances(ctx context.Context, params *ec2.RunInstancesInput, optFns ...func(*ec2.Options)) (*ec2.RunInstancesOutput, error)
	DescribeInstanceStatus(ctx context.Context, input *ec2.DescribeInstanceStatusInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceStatusOutput, error)
	DescribeInstanceTypes(ctx context.Context, input *ec2.DescribeInstanceTypesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceTypesOutput, error)
	GetConsoleOutput(ctx context.Context, input *ec2.GetConsoleOutputInput, optFns ...func(*ec2.Options)) (*ec2.GetConsoleOutputOutput, error)
	TerminateInstances(ctx context.Context, input *ec2.TerminateInstancesInput, optFns ...func(*ec2.Options)) (*ec2.TerminateInstancesOutput, error)
	DescribeVpcAttribute(ctx context.Context, input *ec2.DescribeVpcAttributeInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcAttributeOutput, error)
}

func (c *Client) ByoVPCValidator(ctx context.Context) error {
	c.logger.Info(ctx, "interface executed: %s", ClientIdentifier)
	return nil
}

func (c *Client) ValidateEgress(ctx context.Context, vpcSubnetID, cloudImageID string, kmsKeyID string, timeout time.Duration) *output.Output {
	return c.validateEgress(ctx, vpcSubnetID, cloudImageID, kmsKeyID, timeout)
}

func (c *Client) VerifyDns(ctx context.Context, vpcID string) *output.Output {
	return c.verifyDns(ctx, vpcID)
}

// NewClient creates a new CloudClient for use with AWS.
func NewClient(ctx context.Context, logger ocmlog.Logger, creds interface{}, region, instanceType string, tags map[string]string) (client *Client, err error) {
	switch c := creds.(type) {
	case string:
		client, err = newClient(
			ctx,
			logger,
			"",
			"",
			"",
			region,
			instanceType,
			tags,
			fmt.Sprintf("%v", creds),
		)
	case awscredsv1.Credentials:
		var value awscredsv1.Value
		if value, err = c.Get(); err == nil {
			client, err = newClient(
				ctx,
				logger,
				value.AccessKeyID,
				value.SecretAccessKey,
				value.SessionToken,
				region,
				instanceType,
				tags,
				"",
			)
		}
	case awscredsv2.StaticCredentialsProvider:
		client, err = newClient(
			ctx,
			logger,
			c.Value.AccessKeyID,
			c.Value.SecretAccessKey,
			c.Value.SessionToken,
			region,
			instanceType,
			tags,
			"",
		)
	case string:
		client, err = newClientFromProfile(
			ctx,
			logger,
			c,
			region,
			instanceType,
			tags,
		)
	default:
		err = fmt.Errorf("unsupported credentials type %T", c)
	}

	if err != nil {
		return nil, fmt.Errorf("Unable to create AWS client: %w", err)
	}

	return
}
