// Experimental curl-based probe shim
// Allows the verifier client to parse YAML endpoint lists while testing the experimental probe
// This is just a shim to allow for testing until OSD-21609 proposes a new endpoint list format
package egress

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

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
func curlStringFromYAML(filePath string, variables map[string]string) (string, error) {
	// Read YAML file
	buf, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	// Expand variables and parse into endpoints
	endpoints := reachabilityConfig{}
	variableMapper := func(varName string) string {
		return variables[varName]
	}
	buf = []byte(os.Expand(string(buf), variableMapper))
	err = yaml.Unmarshal(buf, &endpoints)
	if err != nil {
		return "", err
	}
	// Build curl-compatible string of URLs
	var urlListStr string
	for _, endpoint := range endpoints.Endpoints {
		if endpoint.TLSDisabled {
			// NotYetImplemented
			fmt.Printf("WARN: endpoint %s sets TLSDisabled=true, which is not yet "+
				"supported by the experimental curl probe. Endpoint will be probed "+
				"as if TLSDisabled=false, likely causing failed egress check unless "+
				"--no-tls is passed",
				endpoint.Host)
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
			urlListStr += fmt.Sprintf("%s://%s:%d ", protocol, endpoint.Host, port)
		}
	}
	return urlListStr, nil
}
