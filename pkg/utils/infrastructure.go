// utils package should be used by clients and should not import any client packages to avoid import cycles
package utils

import (
	"os"
	"strings"

	configv1 "github.com/openshift/api/config/v1"
)

// PlatformType returns AWS if CLI input AWS profile is set/ or CLI input cloudType=AWS/ or env var AWS_ACCESS_KEY_ID or AWS_PROFILE are set
// returns GCP if CLI input cloudType=GCP
// returns "invalid" platformtype otherwise
func PlatformType(cliPlatformType string, cliAwsProfile string) configv1.PlatformType {
	if strings.EqualFold(cliPlatformType, "aws") ||
		os.Getenv("AWS_ACCESS_KEY_ID") != "" ||
		os.Getenv("AWS_PROFILE") != "" ||
		cliAwsProfile != "" {
		return configv1.AWSPlatformType
	}
	if strings.EqualFold(cliPlatformType, "gcp") {
		return configv1.GCPPlatformType
	}
	return "invalid"
}
