package aws

import (
	"context"

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
