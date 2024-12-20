package egress_lists

import (
	"context"
	"fmt"
	"github.com/aws/smithy-go/ptr"
	"github.com/google/go-github/v63/github"
	"github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/data/cloud"
	"os"
	"strings"
	"testing"
)

func Test_GenerateEgressListsWithInput(t *testing.T) {
	generator := baseGenerator(nil)
	input := `
endpoints:
  - host: something.${AWS_REGION}.com
    ports:
      - 443
`

	tls, _, err := generator.GenerateEgressLists(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}

	expected := "https://something.us-east-1.com:443"
	if strings.TrimSpace(tls) != expected {
		t.Errorf("expected: %s, got: %s", expected, tls)
	}
}

func Test_GenerateEgressListsWithoutInput_FromGithub(t *testing.T) {
	input := `
endpoints:
  - host: github.${AWS_REGION}.com
    ports:
      - 443
`
	githubReposClient := &fakeGithubReposClient{
		err:     nil,
		content: input,
	}
	generator := baseGenerator(githubReposClient)

	tls, _, err := generator.GenerateEgressLists(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}

	expected := "https://github.us-east-1.com:443"
	if strings.TrimSpace(tls) != expected {
		t.Errorf("expected: %s, got: %s", expected, tls)
	}
}

func Test_GenerateEgressListsWithoutInput_WhenGitHubFails(t *testing.T) {
	githubReposClient := &fakeGithubReposClient{
		err:     fmt.Errorf("failed calling github"),
		content: "",
	}
	generator := baseGenerator(githubReposClient)

	tls, _, err := generator.GenerateEgressLists(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}

	// In this case we are falling back to the local file, so assert an arbitrary URL we know
	expected := "https://console.redhat.com:443"
	if !strings.Contains(tls, expected) {
		t.Errorf("expected string to contain %s, got: %s", expected, tls)
	}
}

func baseGenerator(github *fakeGithubReposClient) *Generator {
	logger, err := logging.NewStdLoggerBuilder().Streams(os.Stderr, os.Stderr).Build()
	if err != nil {
		panic(err)
	}

	return &Generator{
		PlatformType:      cloud.AWSClassic,
		Variables:         map[string]string{"AWS_REGION": "us-east-1"},
		logger:            logger,
		githubReposClient: github,
	}
}

type fakeGithubReposClient struct {
	err     error
	content string
}

func (f *fakeGithubReposClient) GetContents(_ context.Context, _, _, _ string, _ *github.RepositoryContentGetOptions) (
	fileContent *github.RepositoryContent, directoryContent []*github.RepositoryContent, resp *github.Response, err error,
) {
	return &github.RepositoryContent{
		Content: &f.content,
		URL:     ptr.String("github.com/test"),
		SHA:     ptr.String("abc123"),
	}, nil, nil, f.err
}
