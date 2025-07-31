package cloud

import (
	"fmt"
	"slices"
	"strings"
)

// Platform type represents specific Platform types and how they map to their respective platforms.
type Platform struct {
	// names holds 3 unique lowercase names of the Platform (e.g., "aws"). We use a fixed-
	// size array so that this struct remains comparable. Any of the 3 values can be used to refer
	// to this specific Platform via Platform.ByName(), but only the first (element
	// 0) element will be the "preferred name" returned by Platform.String()
	names [3]string
}

var (
	AWSClassic = Platform{
		names: [3]string{"aws-classic", "aws"},
	}
	AWSHCP = Platform{
		names: [3]string{"aws-hcp", "aws-hosted-cp", "hostedcluster"},
	}
	AWSHCPZeroEgress = Platform{
		names: [3]string{"aws-hcp-zeroegress"},
	}
	GCPClassic = Platform{
		names: [3]string{"gcp-classic", "gcp"},
	}
)

// String returns the "preferred name" of the Platform
func (p Platform) String() string {
	return p.names[0]
}

// ByName returns a Platform supported by the verifier if the given name
// matches any known common names for a supported Platform. It returns an empty/invalid
// platform if the provided name isn't supported
func ByName(name string) (Platform, error) {
	normalizedName := strings.TrimSpace(strings.ToLower(name))

	if normalizedName == "" {
		return Platform{}, fmt.Errorf("attempted to lookup Platform with empty string")
	}

	if slices.Contains(AWSClassic.names[:], normalizedName) {
		return AWSClassic, nil
	}

	if slices.Contains(AWSHCP.names[:], normalizedName) {
		return AWSHCP, nil
	}

	if slices.Contains(GCPClassic.names[:], normalizedName) {
		return GCPClassic, nil
	}

	if slices.Contains(AWSHCPZeroEgress.names[:], normalizedName) {
		return AWSHCPZeroEgress, nil
	}

	return Platform{}, fmt.Errorf("no platform with name %s", name)
}

// IsValid returns true if the Platform is non-empty and supported by the network verifier
func (p Platform) IsValid() bool {
	switch p {
	case AWSClassic, AWSHCP, GCPClassic, AWSHCPZeroEgress:
		return true
	default:
		return false
	}
}

func (p Platform) IsAws() bool {
	switch p {
	case AWSClassic, AWSHCP, AWSHCPZeroEgress:
		return true
	default:
		return false
	}
}
