package cpu

import (
	"fmt"

	"github.com/openshift/osd-network-verifier/pkg/helpers"
)

// Architecture type represents specific CPU architectures and stores information on how they
// map to cloud instance types. The verifier currently supports two: cpu.ArchX86 and cpu.ArchARM
type Architecture struct {
	// names holds 3 unique lowercase names of the CPU architecture (e.g., "x86"). We use a fixed-
	// size array so that this struct remains comparable. Any of the 3 values can be used to refer
	// to this specific CPU architecture via cpu.ArchitectureByName(), but only the first (element
	// 0) element will be the "preferred name" returned by Architecture.String()
	names [3]string

	// defaultAWSInstanceType is the name of a low-cost AWS EC2 instance type that uses this CPU
	// architecture and can be considered a sane default for the verifier Probe instance
	defaultAWSInstanceType string

	// defaultGCPInstanceType is the name of a low-cost GCP Compute Engine machine type that uses
	// this CPU architecture and can be considered a sane default for the verifier Probe instance
	defaultGCPInstanceType string
}

// If adding a new Arch, be sure to add it to the switch case in Architecture.IsValid()
var (
	// ArchX86 represents a 64-bit CPU using an ARM-based instruction set
	ArchX86 = Architecture{
		names:                  [3]string{"x86", "x86_64", "amd64"},
		defaultAWSInstanceType: "t3.micro",
		defaultGCPInstanceType: "e2-micro",
	}

	// ArchARM represents a 64-bit CPU using an ARM-based instruction set
	ArchARM = Architecture{
		names:                  [3]string{"arm", "arm64", "aarch64"},
		defaultAWSInstanceType: "t4g.micro",
		defaultGCPInstanceType: "t2a-standard-1",
	}
)

// IsValid returns true if the Architecture is non-empty and supported by the network verifier
func (arch Architecture) IsValid() bool {
	switch arch {
	case ArchX86, ArchARM:
		return true
	default:
		return false
	}
}

// String returns the "preferred name" of the Architecture
func (arch Architecture) String() string {
	return arch.names[0]
}

// DefaultInstanceType returns a sane default instance/machine type for the given cloud platform
func (arch Architecture) DefaultInstanceType(platformType string) (string, error) {
	if !arch.IsValid() {
		return "", fmt.Errorf("invalid Architecture")
	}

	normalizedPlatformType, err := helpers.GetPlatformType(platformType)
	if err != nil {
		return "", err
	}

	switch normalizedPlatformType {
	case helpers.PlatformAWS, helpers.PlatformHostedCluster:
		return arch.defaultAWSInstanceType, nil
	case helpers.PlatformGCP:
		return arch.defaultGCPInstanceType, nil
	default:
		return "", fmt.Errorf("no default instance type for %s", normalizedPlatformType)
	}
}
