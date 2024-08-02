package gcpverifier

import (
	"fmt"
	"strings"
	"time"

	"github.com/openshift/osd-network-verifier/pkg/helpers"
	"github.com/openshift/osd-network-verifier/pkg/probes"
)

func get_tokens(consoleOutput string, probe probes.Probe) bool {
	// Check for startingToken and endingToken
	startingTokenSeen := strings.Contains(consoleOutput, probe.GetStartingToken())
	endingTokenSeen := strings.Contains(consoleOutput, probe.GetEndingToken())
	if !startingTokenSeen {
		if endingTokenSeen {
			fmt.Printf("raw console logs:\n---\n%s\n---", consoleOutput)
			fmt.Printf("probe output corrupted: endingToken encountered before startingToken")
			return false
		}
		fmt.Printf("consoleOutput contains data, but probe has not yet printed startingToken, continuing to wait...")
		return false
	}
	if !endingTokenSeen {
		fmt.Printf("consoleOutput contains data, but probe has not yet printed endingToken, continuing to wait...")
		return false
	}
	// If we make it this far, we know that both startingTokenSeen and endingTokenSeen are true

	// Separate the probe's output from the rest of the console output (using startingToken and endingToken)
	rawProbeOutput := strings.TrimSpace(helpers.CutBetween(consoleOutput, probe.GetStartingToken(), probe.GetEndingToken()))
	if len(rawProbeOutput) < 1 {
		fmt.Printf("raw console logs:\n---\n%s\n---", consoleOutput)
		fmt.Printf("probe output corrupted: no data between startingToken and endingToken")
		return false
	}
	// Send probe's output off to the Probe interface for parsing
	fmt.Printf("probe output:\n---\n%s\n---", rawProbeOutput)

	return true
}

// test parsing with poll immediate
func get_unreachable_endpoints(consoleOutput string, probe probes.Probe) error {
	fmt.Printf("Scraping console output and waiting for user data script to complete...")
	// Scrapes console at specified interval up to specified timeout
	err := helpers.PollImmediate(30*time.Second, 4*time.Minute, func() (bool, error) {
		// Get the console output from the ComputeService instance
		get_tokens(consoleOutput, probe)
		return true, nil
	})
	return err
}
