package egress_lists

import (
	"os"
	"strings"
	"testing"

	"github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/data/cloud"
)

func Test_GenerateEgressListsWithInput(t *testing.T) {
	generator := baseGenerator()
	input := `
endpoints:
  - host: something.${AWS_REGION}.com
    ports:
      - 443
`

	tls, _, err := generator.GenerateEgressLists(input)
	if err != nil {
		t.Fatal(err)
	}

	expected := "https://something.us-east-1.com:443"
	if strings.TrimSpace(tls) != expected {
		t.Errorf("expected: %s, got: %s", expected, tls)
	}
}

func Test_GenerateEgressListsWithoutInput_UsesLocalList(t *testing.T) {
	generator := baseGenerator()

	tls, _, err := generator.GenerateEgressLists("")
	if err != nil {
		t.Fatal(err)
	}

	// Assert an arbitrary URL we know is in the local embedded egress list
	expected := "https://console.redhat.com:443"
	if !strings.Contains(tls, expected) {
		t.Errorf("expected string to contain %s, got: %s", expected, tls)
	}
}

func baseGenerator() *Generator {
	logger, err := logging.NewStdLoggerBuilder().Streams(os.Stderr, os.Stderr).Build()
	if err != nil {
		panic(err)
	}

	return &Generator{
		PlatformType: cloud.AWSClassic,
		Variables:    map[string]string{"AWS_REGION": "us-east-1"},
		logger:       logger,
	}
}
