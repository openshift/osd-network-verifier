package egress_lists

import (
	_ "embed"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/data/cloud"
)

//go:embed aws-classic.yaml
var templateAWSClassic string

//go:embed aws-hcp.yaml
var templateAWSHCP string

//go:embed gcp-classic.yaml
var templateGCPClassic string

//go:embed aws-hcp-zeroegress.yaml
var templateAWSHCPZeroEgress string

//go:embed aws-govcloud-classic.yaml
var templateAWSGovCloudClassic string

// Generator provides a mechanism to generate egress lists for a given platform and set of variables
type Generator struct {
	// PlatformType represents the cloud and type of platform we are generating egress lists for
	PlatformType cloud.Platform

	// Variables is a map of string:string used to replace templated values in canned egress lists
	Variables map[string]string

	logger logging.Logger
}

func NewGenerator(platformType cloud.Platform, variables map[string]string, logger logging.Logger) *Generator {
	return &Generator{
		PlatformType: platformType,
		Variables:    variables,
		logger:       logger,
	}
}

// GenerateEgressLists takes an optional egressListYaml as input, and then attempts to return generated EgressLists
// in the following order:
// - If a populated egressListYaml is passed, use that
// - Otherwise, use the local yaml embedded in this package
func (g *Generator) GenerateEgressLists(egressListYaml string) (string, string, error) {
	if egressListYaml != "" {
		return g.EgressListToString(egressListYaml, g.Variables)
	}

	egress, err := g.GetLocalEgressList()
	if err != nil {
		return "", "", err
	}

	return g.EgressListToString(egress, g.Variables)
}

func (g *Generator) GetLocalEgressList() (string, error) {
	switch g.PlatformType {
	case cloud.GCPClassic:
		return templateGCPClassic, nil
	case cloud.AWSHCP:
		return templateAWSHCP, nil
	case cloud.AWSClassic:
		return templateAWSClassic, nil
	case cloud.AWSGovCloudClassic:
		return templateAWSGovCloudClassic, nil
	case cloud.AWSHCPZeroEgress:
		return templateAWSHCPZeroEgress, nil
	default:
		return "", fmt.Errorf("no egress list registered for platform '%s'", g.PlatformType)
	}
}

// EgressListToString returns two strings, the sum of which contains all the URLs
// within a given platformType's egress list.
// The first string returned contains all the URLs with tlsDisabled=false,
// while the second string contains all URLs with tlsDisabled=true
func (g *Generator) EgressListToString(egressListYamlStr string, variables map[string]string) (string, string, error) {
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

type endpoint struct {
	Host        string `yaml:"host"`
	Ports       []int  `yaml:"ports"`
	TLSDisabled bool   `yaml:"tlsDisabled"`
}

type reachabilityConfig struct {
	Endpoints []endpoint `yaml:"endpoints"`
}
