package probes

import (
	"github.com/openshift/osd-network-verifier/pkg/output"
)

type Probe interface {
	GetStartingToken() string
	GetEndingToken() string
	GetUserDataTemplate() string
	ParseProbeOutput(string) *output.Output
}
