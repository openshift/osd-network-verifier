package verifier

import (
	"context"
	"time"

	"github.com/openshift/osd-network-verifier/pkg/data/cpu"
	"github.com/openshift/osd-network-verifier/pkg/output"
	"github.com/openshift/osd-network-verifier/pkg/probes"
	"github.com/openshift/osd-network-verifier/pkg/proxy"
)

// VerifierService defines the behaviors necessary to run verifier completely. Any clients that use that fullfills this interface
// will be able to run all verifier test
type verifierService interface {

	// ValidateEgress validates that all required targets are reachable from the vpcsubnet
	// target URLs: https://docs.openshift.com/rosa/rosa_getting_started/rosa-aws-prereqs.html#osd-aws-privatelink-firewall-prerequisites
	// Expected return value is *output.Output that's storing failures, exceptions and errors
	ValidateEgress(vei ValidateEgressInput) *output.Output

	// VerifyDns verifies that a given VPC meets the DNS requirements specified in:
	// https://docs.openshift.com/container-platform/4.10/installing/installing_aws/installing-aws-vpc.html
	// Expected return value is *output.Output that's storing failures, exceptions and errors
	VerifyDns(vdi VerifyDnsInput) *output.Output
}

type ValidateEgressInput struct {
	// Timeout sets the maximum duration an egress endpoint request can take before it aborts and
	// is retried or marked as blocked
	Timeout                              time.Duration
	Ctx                                  context.Context
	SubnetID, CloudImageID, PlatformType string
	Proxy                                proxy.ProxyConfig
	Tags                                 map[string]string
	AWS                                  AwsEgressConfig
	GCP                                  GcpEgressConfig
	SkipInstanceTermination              bool
	TerminateDebugInstance               string
	ImportKeyPair                        string
	ForceTempSecurityGroup               bool

	// InstanceType sets the type or size of the instance (VM) launched into the target subnet. Only
	// instance types using 64-bit X86 or ARM CPUs are supported. For AWS, only instance types using
	// the "Nitro" hypervisor are supported, as other hypervisors don't allow the verifier to gather
	// probe results from the instance's serial console. If no valid InstanceType is provided, the
	// verifier falls back to a supported default using the same CPU architecture as the requested
	// instance type (if applicable) or as specified in the CPUArchitecture field
	InstanceType string

	// Probe controls the behavior of the instance that the verifier launches into the target
	// subnet. Defaults to a curl-based probe (curl.Probe) if unset. legacy.Probe is also available
	// if you'd like the verifier to emulate its pre-1.0 behavior, or you may provide your own
	// implementation of the probes.Probe interface
	Probe probes.Probe

	// CPUArchitecture controls the CPU architecture of the default/fallback cloud instance type.
	// Has no effect if a supported value of InstanceType is provided.
	CPUArchitecture cpu.Architecture
}
type AwsEgressConfig struct {
	KmsKeyID          string
	SecurityGroupIDs  []string
	TempSecurityGroup string
}
type GcpEgressConfig struct {
	Region, Zone, ProjectID, VpcName string
}

// ValidateEgress pass in a GCP or AWS client that know how to fufill above interface
func ValidateEgress(vs verifierService, vei ValidateEgressInput) *output.Output {
	return vs.ValidateEgress(vei)
}

type VerifyDnsInput struct {
	Ctx   context.Context
	VpcID string
}

// VerifyDns pass in a GCP or AWS client that know how to fufill above interface
func VerifyDns(vs verifierService, vdi VerifyDnsInput) *output.Output {
	return vs.VerifyDns(vdi)
}
