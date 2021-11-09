package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	configv1 "github.com/openshift/api/config/v1"
)

// ClientIdentifier is what kind of cloud this implement supports
const ClientIdentifier configv1.PlatformType = configv1.AWSPlatformType

// Client represents an AWS Client
type Client struct {
	ec2Client ec2iface.EC2API
}

func (c *Client) ByoVPCValidator(context.Context) error {
	return nil
}

func newClient(accessID, accessSecret, region string) (*Client, error) {
	awsConfig := &aws.Config{Region: aws.String(region), Credentials: credentials.NewStaticCredentials(accessID, accessSecret, "")}
	s, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, err
	}
	return &Client{
		ec2Client: ec2.New(s),
	}, nil
}

// NewClient creates a new CloudClient for use with AWS.
func NewClient() (*Client, error) {
	c, err := newClient(
		string("INVALID_accessKeyID"),
		string("INVALID_secretAccessKey"),
		"eu-west-1")

	if err != nil {
		return nil, fmt.Errorf("couldn't create AWS client %w", err)
	}

	return c, nil
}
