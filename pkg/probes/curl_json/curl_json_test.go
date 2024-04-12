package curl_json

import (
	_ "embed"
	"testing"

	"github.com/openshift/osd-network-verifier/pkg/probes"
)

// TestCurlJSONProbe_ImplementsProbeInterface simply forces the compiler
// to confirm that the TestCurlJSONProbe type properly implements the Probe
// interface. If not (e.g, because a required method is missing), this
// test will fail to compile
func TestCurlJSONProbe_ImplementsProbeInterface(t *testing.T) {
	var _ probes.Probe = (*CurlJSONProbe)(nil)
}
