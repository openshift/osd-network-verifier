package cloudclient

import (
	"context"
	"fmt"

	awscredsv2 "github.com/aws/aws-sdk-go-v2/credentials"
	awscredsv1 "github.com/aws/aws-sdk-go/aws/credentials"
	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	awsCloudClient "github.com/openshift/osd-network-verifier/pkg/cloudclient/aws"
	gcpCloudClient "github.com/openshift/osd-network-verifier/pkg/cloudclient/gcp"

	"golang.org/x/oauth2/google"
)

// CloudClient defines the interface for a cloud agnostic implementation
// For mocking: mockgen -source=pkg/cloudclient/cloudclient.go -package mocks -destination=pkg/cloudclient/mocks/mock_cloudclient.go
type CloudClient interface {

	// ByoVPCValidator validates the configuration given by the customer
	ByoVPCValidator(ctx context.Context) error

	// ValidateEgress validates that all required targets are reachable from the vpcsubnet
	// required target are defined in https://docs.openshift.com/rosa/rosa_getting_started/rosa-aws-prereqs.html#osd-aws-privatelink-firewall-prerequisites
	ValidateEgress(ctx context.Context, vpcSubnetID, cloudImageID string) error
}

func NewClient(ctx context.Context, logger ocmlog.Logger, creds interface{}, region, instanceType string, tags map[string]string) (CloudClient, error) {
	switch c := creds.(type) {
	case awscredsv1.Credentials, awscredsv2.StaticCredentialsProvider:
		return awsCloudClient.NewClient(ctx, logger, c, region, instanceType, tags)
	case *google.Credentials:
		return gcpCloudClient.NewClient(ctx, logger, c, region, instanceType, tags)
	default:
		return nil, fmt.Errorf("unsupported credentials type %T", c)
	}

}
