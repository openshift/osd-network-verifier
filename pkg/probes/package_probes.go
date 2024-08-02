package probes

import (
	"github.com/openshift/osd-network-verifier/pkg/data/cpu"
	"github.com/openshift/osd-network-verifier/pkg/output"
)

type Probe interface {
	GetMachineImageID(platformType string, cpuArch cpu.Architecture, region string) (string, error)
	GetStartingToken() string
	GetEndingToken() string
	GetExpandedUserData(userDataVariables map[string]string, userDataTemplate string) (string, error)
	ParseProbeOutput(string, *output.Output)
}
