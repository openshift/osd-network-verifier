package egress_lists

// TRANSITIONAL IMPLEMENTATION (UNSTABLE API)
// This module currently provides very basic fetching of egress lists stored within the binary
// in legacy probe/golden-AMI format. Most of its current logic & structs were borrowed from
// osd-network-verifier-golden-ami/build/bin/network-validator.go with very little validation.
// OSD-22628 will significantly change this module. Consider this internal API unstable until
// that card is complete.

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/openshift/osd-network-verifier/pkg/helpers"
	"gopkg.in/yaml.v3"
)

//go:embed aws-classic.yaml
var templateAWSClassic string

//go:embed aws-hcp.yaml
var templateAWSHCP string

//go:embed gcp-classic.yaml
var templateGCPClassic string

// GetEgressListAsCurlString returns a string of curl parameters representing all of the URLs
// contained within a given platformType's egress list
func GetEgressListAsCurlString(platformType string, region string) (string, error) {
	normalizedPlatformType, err := helpers.GetPlatformType(platformType)
	if err != nil {
		return "", err
	}

	var egressListYamlStr string
	switch normalizedPlatformType {
	case helpers.PlatformGCP:
		egressListYamlStr = templateGCPClassic
	case helpers.PlatformHostedCluster:
		egressListYamlStr = templateAWSHCP
	case helpers.PlatformAWS:
		egressListYamlStr = templateAWSClassic
	default:
		return "", fmt.Errorf("no egress list registered for platform '%s' (normalized to '%s')", platformType, normalizedPlatformType)
	}

	curlStr, err := curlStringFromYAML(egressListYamlStr, map[string]string{"AWS_REGION": region})
	if err != nil {
		return "", fmt.Errorf("unable to parse YAML in egress list for platform '%s' (normalized to '%s')", platformType, normalizedPlatformType)
	}

	return curlStr, nil
}

// endpoint type (as it appears in the current YAML schema)
// Borrowed from osd-network-verifier-golden-ami/build/bin/network-validator.go
type endpoint struct {
	Host        string `yaml:"host"`
	Ports       []int  `yaml:"ports"`
	TLSDisabled bool   `yaml:"tlsDisabled"`
}

// endpoint list type (as it appears in the current YAML schema)
// Borrowed from osd-network-verifier-golden-ami/build/bin/network-validator.go
type reachabilityConfig struct {
	Endpoints []endpoint `yaml:"endpoints"`
}

// crude YAML to curl-formatted string converter
// Adapted from osd-network-verifier-golden-ami/build/bin/network-validator.go
func curlStringFromYAML(yamlStr string, variables map[string]string) (string, error) {
	// Expand variables and parse into endpoints
	endpoints := reachabilityConfig{}
	variableMapper := func(varName string) string {
		return variables[varName]
	}
	buf := []byte(os.Expand(yamlStr, variableMapper))
	err := yaml.Unmarshal(buf, &endpoints)
	if err != nil {
		return "", err
	}
	// Build curl-compatible string of URLs
	var urlListStr string
	for _, endpoint := range endpoints.Endpoints {
		var insecureStart string
		var insecureEnd string
		if endpoint.TLSDisabled {
			// See https://curl.se/docs/manpage.html#-k
			insecureStart = "-k "
			insecureEnd = " --no-insecure"
		}
		// Infer protocol
		for _, port := range endpoint.Ports {
			var protocol string
			switch port {
			case 80:
				protocol = "http"
			case 443:
				protocol = "https"
			default:
				protocol = "telnet"
			}
			urlListStr += fmt.Sprintf("%s%s://%s:%d%s ", insecureStart, protocol, endpoint.Host, port, insecureEnd)
		}
	}
	return urlListStr, nil
}
