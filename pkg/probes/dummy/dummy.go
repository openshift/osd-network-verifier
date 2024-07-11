package dummy

import (
	"github.com/openshift/osd-network-verifier/pkg/output"
)

type DummyProbe struct{}

const (
	startingToken = "DUMMY_START"
	endingToken   = "DUMMY_END"
)

// GetStartingToken returns the string token used to signal the beginning of the probe's output
func (prb DummyProbe) GetStartingToken() string { return startingToken }

// GetEndingToken returns the string token used to signal the end of the probe's output
func (prb DummyProbe) GetEndingToken() string { return endingToken }

// GetMachineImageID returns the string ID of the VM image to be used for the probe instance
func (prb DummyProbe) GetMachineImageID(platformType string, cpuArch string, region string) (string, error) {
	return "rhel-9", nil
}

// GetExpandedUserData returns a bash-formatted userdata string
func (prb DummyProbe) GetExpandedUserData(userDataVariables map[string]string) (string, error) {
	return `#!/bin/sh
	systemctl mask --now serial-getty@ttyS0.service
	systemctl disable --now syslog.socket rsyslog.service
	sysctl -w kernel.printk="0 4 0 7"
	echo DUMMY_START > /dev/ttyS0
	echo "hello world" > /dev/ttyS0
	echo DUMMY_END > /dev/ttyS0`, nil
}

// ParseProbeOutput is not implemented for this dummy probe
func (prb DummyProbe) ParseProbeOutput(probeOutput string, outputDestination *output.Output) {
	return
}
