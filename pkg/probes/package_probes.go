package probes

import (
	"github.com/openshift/osd-network-verifier/pkg/output"
)

type Probe interface {
	GetStartingToken() string
	GetEndingToken() string
	GetExpandedUserData(map[string]string) (string, error)
	ParseProbeOutput(string, *output.Output)
}
