package cpu

import (
	"fmt"
	"slices"
	"strings"

	"github.com/openshift/osd-network-verifier/pkg/data/cloud"
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

// If adding a new Arch, be sure to add it to Architecture.IsValid() and cpu.ArchitectureByName()
var (
	// ArchX86 represents a 64-bit CPU using an x86-based instruction set
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

// String returns the "preferred name" of the Architecture
func (arch Architecture) String() string {
	return arch.names[0]
}

// DefaultInstanceType returns a sane default instance/machine type for the given cloud platform
func (arch Architecture) DefaultInstanceType(platformType cloud.Platform) (string, error) {
	if !arch.IsValid() {
		return "", fmt.Errorf("invalid Architecture")
	}

	if !platformType.IsValid() {
		return "", fmt.Errorf("invalid Platform")
	}

	switch platformType {
	case cloud.AWSClassic, cloud.AWSHCP, cloud.AWSHCPZeroEgress:
		return arch.defaultAWSInstanceType, nil
	case cloud.GCPClassic:
		return arch.defaultGCPInstanceType, nil
	default:
		return "", fmt.Errorf("no default instance type for %s", platformType)
	}
}

// IsValid returns true if the Architecture is non-empty and supported by the network verifier
func (arch Architecture) IsValid() bool {
	switch arch {
	case ArchX86, ArchARM:
		return true
	default:
		return false
	}
}

// ArchitectureByName returns an Architecture supported by the verifier if the given name
// matches any known common names for a supported Architecture. It returns an empty/invalid
// architecture if the provided name isn't supported
func ArchitectureByName(name string) Architecture {
	normalizedName := strings.TrimSpace(strings.ToLower(name))

	if normalizedName == "" {
		return Architecture{}
	}

	if slices.Contains(ArchX86.names[:], normalizedName) {
		return ArchX86
	}

	if slices.Contains(ArchARM.names[:], normalizedName) {
		return ArchARM
	}

	return Architecture{}
}
