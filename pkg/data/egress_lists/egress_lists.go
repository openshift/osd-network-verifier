package egress_lists

import (
	"context"
	_ "embed"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"

	"github.com/google/go-github/v63/github"
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

type githubReposClient interface {
	GetContents(ctx context.Context, owner, repo, path string, opts *github.RepositoryContentGetOptions) (fileContent *github.RepositoryContent, directoryContent []*github.RepositoryContent, resp *github.Response, err error)
}

// Generator provides a mechanism to generate egress lists for a given platform and set of variables
type Generator struct {
	// PlatformType represents the cloud and type of platform we are generating egress lists for
	PlatformType cloud.Platform

	// Variables is a map of string:string used to replace templated values in canned egress lists
	Variables map[string]string

	logger            logging.Logger
	githubReposClient githubReposClient
}

func NewGenerator(platformType cloud.Platform, variables map[string]string, logger logging.Logger) *Generator {
	return &Generator{
		PlatformType:      platformType,
		Variables:         variables,
		logger:            logger,
		githubReposClient: github.NewClient(nil).Repositories,
	}
}

// GenerateEgressLists takes an optional egressListYaml as input, and then attempts to return generated EgressLists
// in the following order:
// - If a populated egressListYaml is passed, use that
// - Otherwise, try to get the values from GitHub, and if that fails
// - Fallback to the local yaml embedded in this package
func (g *Generator) GenerateEgressLists(ctx context.Context, egressListYaml string) (string, string, error) {
	if egressListYaml != "" {
		return g.EgressListToString(egressListYaml, g.Variables)
	}

	egressResponse, err := g.GetGithubEgressList(ctx)
	if err != nil {
		g.logger.Error(ctx, "Failed to get egress list from GitHub, falling back to local list: %v", err)

		egress, err := g.GetLocalEgressList()
		if err != nil {
			return "", "", err
		}

		return g.EgressListToString(egress, g.Variables)
	}

	egress, err := egressResponse.GetContent()
	if err != nil {
		return "", "", err
	}

	g.logger.Info(ctx, "Using egress URL list from %s at SHA %s", egressResponse.GetURL(), egressResponse.GetSHA())

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
	case cloud.AWSHCPZeroEgress:
		return templateAWSHCPZeroEgress, nil
	default:
		return "", fmt.Errorf("no egress list registered for platform '%s'", g.PlatformType)
	}
}

func (g *Generator) GetGithubEgressList(ctx context.Context) (*github.RepositoryContent, error) {
	path := "/pkg/data/egress_lists/"

	switch g.PlatformType {
	case cloud.GCPClassic:
		path += cloud.GCPClassic.String()
	case cloud.AWSHCP:
		path += cloud.AWSHCP.String()
	case cloud.AWSClassic:
		path += cloud.AWSClassic.String()
	case cloud.AWSHCPZeroEgress:
		path += cloud.AWSHCPZeroEgress.String()
	default:
		return nil, fmt.Errorf("no egress list registered for platform '%s'", g.PlatformType)
	}
	fileContentResponse, _, _, err := g.githubReposClient.GetContents(ctx, "openshift", "osd-network-verifier", fmt.Sprintf("%s.yaml", path), nil)
	return fileContentResponse, err
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
