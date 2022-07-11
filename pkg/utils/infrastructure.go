// utils package should be used by clients and should not import any client packages to avoid import cycles
package utils

import (
	"os"
	"strings"
	// following has predefined provider identifier types, but removing dependency as it requires go 1.17
	//configv1 "github.com/openshift/api/config/v1"
)

type Infrastructure struct {
}

const (
	TYPE_AWS     = "AWS"
	TYPE_GCP     = "GCP"
	TYPE_INVALID = "invalid"
)

// PlatformType returns AWS if CLI input AWS profile is set/ or CLI input cloudType=AWS/ or env var AWS_ACCESS_KEY_ID or AWS_PROFILE are set
// returns GCP if CLI input cloudType=GCP
// returns "invalid" platformtype otherwise
func PlatformType(cliPlatformType string, cliAwsProfile string) string {
	if strings.EqualFold(cliPlatformType, "aws") ||
		os.Getenv("AWS_ACCESS_KEY_ID") != "" ||
		os.Getenv("AWS_PROFILE") != "" ||
		cliAwsProfile != "" {
		return TYPE_AWS
	}
	if strings.EqualFold(cliPlatformType, "gcp") {
		return TYPE_GCP
	}
	return TYPE_INVALID
}
