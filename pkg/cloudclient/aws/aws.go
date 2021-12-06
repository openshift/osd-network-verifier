package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	configv1 "github.com/openshift/api/config/v1"
)

// ClientIdentifier is what kind of cloud this implement supports
const ClientIdentifier configv1.PlatformType = configv1.AWSPlatformType

// Client represents an AWS Client
type Client struct {
	ec2Client *ec2.Client
}

func (c *Client) ByoVPCValidator(context.Context) error {
	fmt.Println("interface executed: " + ClientIdentifier)
	return nil
}

// NewClient creates a new CloudClient for use with AWS.
func NewClient(creds credentials.StaticCredentialsProvider, region string) (*Client, error) {
	c, err := newClient(
		creds.Value.AccessKeyID,
		creds.Value.SecretAccessKey,
		creds.Value.SessionToken,
		region,
	)

	if err != nil {
		return nil, fmt.Errorf("couldn't create AWS client %w", err)
	}

	return c, nil
}

func (c *Client) ValidateEgress(ctx context.Context, vpcSubnetID, cloudImageID string) error {
	return c.validateEgress(ctx, vpcSubnetID, cloudImageID)
}
