package cloudclient

import (
	"context"
	"fmt"
	"time"

	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/output"
	"github.com/openshift/osd-network-verifier/pkg/utils"
)

// Config struct for cloudclient creation.
// This struct separates provider specific configs and makes coudclient provider agnostic.
// Downstream function utilizes "GetClientFor" factory with
// their desired set of ClientConfig and ExecConfig.
// This factory returns their required client intelligently based on the input.
// For most tests such as egress, all params in this struct are optional and
// client can be created solely using on credentials.
type ClientConfig struct {
	CloudType string
	AWSConfig *utils.AWSClientConfig
	GCPConfig *utils.GCPClientConfig
}

// Execution options.
// These are only related to execution options (such as debug, timeout) and do not include configs for cloud provider.
type ExecConfig struct {
	// Operation options
	Debug   bool
	Timeout time.Duration

	// Following are passed to client to mitigate "cannot create context from nil parent" error
	Ctx    context.Context
	Logger ocmlog.Logger
}

var (
	DefaultTime = 2 * time.Second
	Debug       = false
)

// CloudClient defines the interface for a cloud agnostic implementation
// For mocking: mockgen -source=pkg/cloudclient/cloudclient.go -package mocks -destination=pkg/cloudclient/mocks/mock_cloudclient.go
type CloudClient interface {

	// ByoVPCValidator validates the configuration given by the customer
	ByoVPCValidator(params ValidateByoVpc) error

	// ValidateEgress validates that all required targets are reachable from the vpcsubnet
	// target URLs: https://docs.openshift.com/rosa/rosa_getting_started/rosa-aws-prereqs.html#osd-aws-privatelink-firewall-prerequisites
	// Expected return value is *output.Output that's storing failures, exceptions and errors
	ValidateEgress(params ValidateEgress) *output.Output

	// VerifyDns verifies that a given VPC meets the DNS requirements specified in:
	// https://docs.openshift.com/container-platform/4.10/installing/installing_aws/installing-aws-vpc.html
	// Expected return value is *output.Output that's storing failures, exceptions and errors
	VerifyDns(params ValidateDns) *output.Output
}

var controllerMapping = map[string]Factory{}

type Factory func(clientConfig *ClientConfig, execConfig *ExecConfig) (CloudClient, error)

func Register(providerType string, factoryFunc Factory) {
	controllerMapping[providerType] = factoryFunc
}

// GetClientFor returns the CloudClient for any cloud provider based on clientConfig.
// ExecConfig is used to format output, timeouts etc.
func GetClientFor(clientConfig *ClientConfig, execConfig *ExecConfig) (CloudClient, error) {
	platformType := utils.PlatformType(clientConfig.CloudType)
	cli, err := controllerMapping[platformType](clientConfig, execConfig)
	if err != nil {
		return nil, (fmt.Errorf("Couldn't create cloud client for %s: %s", platformType, err))

	}
	return cli, nil
}

// Test parameter structs. Define struct for each new test.

type ValidateEgress struct {
	VpcSubnetID string
}

type ValidateDns struct {
	VpcId string
}

type ValidateByoVpc struct {
	//	todo
}
