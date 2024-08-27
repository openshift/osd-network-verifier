package egress_lists

// TRANSITIONAL IMPLEMENTATION (UNSTABLE API)
// This module currently provides very basic fetching of egress lists stored within the binary
// in legacy probe/golden-AMI format. Most of its current logic & structs were borrowed from
// osd-network-verifier-golden-ami/build/bin/network-validator.go with very little validation.
// OSD-22628 will significantly change this module. Consider this internal API unstable until
// that card is complete.

import (
	"context"
	_ "embed"
	"fmt"
	"os"

	"github.com/google/go-github/v63/github"
	platform "github.com/openshift/osd-network-verifier/pkg/data/cloud"
	"gopkg.in/yaml.v3"
)

//go:embed aws-classic.yaml
var templateAWSClassic string

//go:embed aws-hcp.yaml
var templateAWSHCP string

//go:embed gcp-classic.yaml
var templateGCPClassic string

func GetLocalEgressList(platformType string) (string, error) {
	platformTypeName, err := platform.PlatformByName(platformType)
	if err != nil {
		return "", err
	}
	platformType = platformTypeName.String()

	switch platformType {
	case platform.GCPClassic.String():
		return templateGCPClassic, nil
	case platform.AWSHCP.String():
		return templateAWSHCP, nil
	case platform.AWSClassic.String():
		return templateAWSClassic, nil
	default:
		return "", fmt.Errorf("no egress list registered for platform '%s'", platformType)
	}
}

func GetGithubEgressList(platformType string) (*github.RepositoryContent, error) {
	ghClient := github.NewClient(nil)
	path := "/pkg/data/egress_lists/"
	platformTypeName, err := platform.PlatformByName(platformType)
	if err != nil {
		return nil, err
	}
	platformType = platformTypeName.String()

	switch platformType {
	case platform.GCPClassic.String():
		path += platform.GCPClassic.String()
	case platform.AWSHCP.String():
		path += platform.AWSHCP.String()
	case platform.AWSClassic.String():
		path += platform.AWSClassic.String()
	default:
		return nil, fmt.Errorf("no egress list registered for platform '%s'", platformType)
	}
	fileContentResponse, _, _, err := ghClient.Repositories.GetContents(context.TODO(), "openshift", "osd-network-verifier", fmt.Sprintf("%s.yaml", path), nil)
	return fileContentResponse, err
}

// EgressListToString returns two strings, the sum of which contains all the URLs
// within a given platformType's egress list.
// The first string returned contains all the URLs with tlsDisabled=false,
// while the second string contains all URLs with tlsDisabled=true
func EgressListToString(egressListYamlStr string, variables map[string]string) (string, string, error) {
	variableMapper := func(varName string) string {
		return variables[varName]
	}
	buf := []byte(os.Expand(egressListYamlStr, variableMapper))

	endpoints := reachabilityConfig{}
	err := yaml.Unmarshal(buf, &endpoints)
	if err != nil {
		return "", "", err
	}
	// Build curl-compatible string of URLs
	var urlListStr string
	var tlsDisabledURLListStr string
	for _, endpoint := range endpoints.Endpoints {
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
			urlStr := fmt.Sprintf("%s://%s:%d ", protocol, endpoint.Host, port)

			if endpoint.TLSDisabled {
				tlsDisabledURLListStr += urlStr
				continue
			}
			urlListStr += urlStr
		}
	}
	return urlListStr, tlsDisabledURLListStr, nil
}

// endpoint type (as it appears in the current YAML schema)
// Borrowed from osd-network-verifier-golden-ami/build/bin/network-validator.go
type endpoint struct {
	Host        string `yaml:"host"`
	Ports       []int  `yaml:"ports"`
	TLSDisabled bool   `yaml:"tlsDisabled"`
}

// reachabilityConfig list type (as it appears in the current YAML schema)
// Borrowed from osd-network-verifier-golden-ami/build/bin/network-validator.go
type reachabilityConfig struct {
	Endpoints []endpoint `yaml:"endpoints"`
}
