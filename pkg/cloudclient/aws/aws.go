package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/output"
)

// ClientIdentifier is what kind of cloud this implement supports
const ClientIdentifier string = "AWS"

// Client represents an AWS Client
type Client struct {
	CloudTags    map[string]string
	AwsProfile   string
	VpcSubnetID  string
	CloudImageID string
	Timeout      time.Duration
	KmsKeyID     string
	ec2Client    EC2Client
	region       string
	instanceType string
	tags         map[string]string
	logger       ocmlog.Logger
	output       output.Output
}

type ClientInput struct {
	VpcSubnetID     string
	CloudImageID    string
	Timeout         time.Duration
	KmsKeyID        string
	Ctx             context.Context
	Logger          ocmlog.Logger
	Region          string
	InstanceType    string
	Tags            map[string]string
	Profile         string
	AccessKeyId     string
	SessionToken    string
	SecretAccessKey string
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

func (c *Client) ValidateEgress(ctx context.Context) *output.Output {
	return c.validateEgress(ctx)
}

func (c *Client) VerifyDns(ctx context.Context, vpcID string) *output.Output {
	return c.verifyDns(ctx, vpcID)
}

// NewClient creates a new CloudClient for use with AWS.
func NewClient(input *ClientInput) (client *Client, err error) {
	client, err = newClient(input)
	if err != nil {
		return nil, fmt.Errorf("Unable to create AWS client: %w", err)
	}
	return
}

func GetEc2ClientFromInput(input *ClientInput) (ec2.Client, error) {
	ec2Client, err := getEc2ClientFromInput(*input)
	if err != nil {
		return ec2Client, fmt.Errorf("unable to create EC2 Client: %w", err)
	}
	return ec2Client, nil
}
