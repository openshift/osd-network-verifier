package cpu

import (
	"fmt"

	"github.com/openshift/osd-network-verifier/pkg/helpers"
)

// Enumerated type representing CPU architectures
type Architecture string

const (
	// ArchX86 represents a 64-bit CPU using the x86_64 instruction set. Also commonly known as
	// "amd64 CPUs" or simply "Intel CPUs"
	ArchX86 Architecture = "x86"

	// ArchARM represents a 64-bit CPU using an ARM-based instruction set. Also commonly known as
	// "arm64" or "aarch64"
	ArchARM Architecture = "arm"
)

// DefaultInstanceType returns an instance type available on the given cloud platform that's small,
// low-cost, and uses the parent CPU Architecture. See instance_types.go for values
func (arch Architecture) DefaultInstanceType(platformType string) (string, error) {
	platformTypeStr, err := helpers.GetPlatformType(platformType)
	if err != nil {
		return "", fmt.Errorf("failed to fetch default %s instance type for platform %s: %w", arch, platformTypeStr, err)
	}
	if instanceType, ok := defaultInstanceTypes[platformTypeStr][arch]; ok {
		return instanceType, nil
	}

	return "", fmt.Errorf("no default %s instance type for platform %s", arch, platformTypeStr)
}
