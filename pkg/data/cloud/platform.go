package platform

import (
	"fmt"
	"slices"
	"strings"
)

// Platform type represents specific Platform types and how they map to their respective platforms.
type Platform struct {
	// names holds 2 unique lowercase names of the Platform (e.g., "aws"). We use a fixed-
	// size array so that this struct remains comparable. Any of the 2 values can be used to refer
	// to this specific Platform via Platform.ByName(), but only the first (element
	// 0) element will be the "preferred name" returned by Platform.String()
	names [2]string
}

var (
	AWSClassic = Platform{
		names: [2]string{"aws-classic", "aws"},
	}
	AWSHCP = Platform{
		names: [2]string{"aws-hcp", "hostedcluster"},
	}
	GCPClassic = Platform{
		names: [2]string{"gcp-classic", "gcp"},
	}
)

// String returns the "preferred name" of the Platform
func (plat Platform) String() string {
	return plat.names[0]
}

// ByName returns a Platform supported by the verifier if the given name
// matches any known common names for a supported Platform. It returns an empty/invalid
// platform if the provided name isn't supported
func ByName(name string) (Platform, error) {
	normalizedName := strings.TrimSpace(strings.ToLower(name))
	if slices.Contains(AWSClassic.names[:], normalizedName) {
		return AWSClassic, nil
	}

	if slices.Contains(AWSHCP.names[:], normalizedName) {
		return AWSHCP, nil
	}

	if slices.Contains(GCPClassic.names[:], normalizedName) {
		return GCPClassic, nil
	}

	return Platform{}, fmt.Errorf("no platform with name %s", name)
}

// IsValid returns true if the Platform is non-empty and supported by the network verifier
func (plat Platform) isValid() bool {
	switch plat {
	case AWSClassic, AWSHCP, GCPClassic:
		return true
	default:
		return false
	}
}
