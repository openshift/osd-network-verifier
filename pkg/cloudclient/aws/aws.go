package aws

import (
	"context"
	"fmt"

	awscredsv2 "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	awscredsv1 "github.com/aws/aws-sdk-go/aws/credentials"
	configv1 "github.com/openshift/api/config/v1"
)

// ClientIdentifier is what kind of cloud this implement supports
const ClientIdentifier configv1.PlatformType = configv1.AWSPlatformType

// Client represents an AWS Client
type Client struct {
	ec2Client *ec2.Client
	region    string
}

func (c *Client) ByoVPCValidator(context.Context) error {
	fmt.Println("interface executed: " + ClientIdentifier)
	return nil
}

// NewClient creates a new CloudClient for use with AWS.
func NewClient(creds interface{}, region string) (client *Client, err error) {

	switch c := creds.(type) {
	case awscredsv1.Credentials:
		if value, err := c.Get(); err == nil {
			client, err = newClient(
				value.AccessKeyID,
				value.SecretAccessKey,
				value.SessionToken,
				region,
			)
		}
	case awscredsv2.StaticCredentialsProvider:
		client, err = newClient(
			c.Value.AccessKeyID,
			c.Value.SecretAccessKey,
			c.Value.SessionToken,
			region,
		)
	default:
		err = fmt.Errorf("unsupported credentials type %T", c)
	}

	if err != nil {
		return nil, fmt.Errorf("couldn't create AWS client %w", err)
	}

	return
}

func (c *Client) ValidateEgress(ctx context.Context, vpcSubnetID, cloudImageID string) error {
	return c.validateEgress(ctx, vpcSubnetID, cloudImageID)
}
