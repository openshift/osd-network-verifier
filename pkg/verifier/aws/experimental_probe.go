// Experimental curl-based probe shim
// Allows the verifier client to use the experimental probe interface
// This is just a shim to allow for testing until we deprecate the legacy probe code
package awsverifier

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	awsTools "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	handledErrors "github.com/openshift/osd-network-verifier/pkg/errors"
	"github.com/openshift/osd-network-verifier/pkg/helpers"
	"github.com/openshift/osd-network-verifier/pkg/probes"
)

func (a *AwsVerifier) findUnreachableEndpointsExperimental(ctx context.Context, instanceID string, probe probes.Probe) error {
	var consoleOutput string

	a.writeDebugLogs(ctx, "Scraping console output and waiting for user data script to complete...")

	// Periodically scrape console output and analyze the logs for any errors or a successful completion
	err := helpers.PollImmediate(30*time.Second, 4*time.Minute, func() (bool, error) {
		b64EncodedConsoleOutput, err := a.AwsClient.GetConsoleOutput(ctx, &ec2.GetConsoleOutputInput{
			InstanceId: awsTools.String(instanceID),
			Latest:     awsTools.Bool(true),
		})
		if err != nil {
			return false, handledErrors.NewGenericError(err)
		}

		if b64EncodedConsoleOutput.Output == nil {
			return false, nil
		}
		// In the early stages, an ec2 instance may be running but the console is not populated with any data
		if len(*b64EncodedConsoleOutput.Output) == 0 {
			a.writeDebugLogs(ctx, "EC2 console consoleOutput not yet populated with data, continuing to wait...")
			return false, nil
		}

		// Decode base64-encoded console output
		consoleOutputBytes, err := base64.StdEncoding.DecodeString(*b64EncodedConsoleOutput.Output)
		if err != nil {
			a.writeDebugLogs(ctx, fmt.Sprintf("Error decoding console consoleOutput, will retry on next check interval: %s", err))
			return false, nil
		}
		consoleOutput = string(consoleOutputBytes)

		// Check for startingToken and endingToken
		startingTokenSeen := strings.Contains(consoleOutput, probe.GetStartingToken())
		endingTokenSeen := strings.Contains(consoleOutput, probe.GetEndingToken())
		if !startingTokenSeen {
			if endingTokenSeen {
				a.writeDebugLogs(ctx, fmt.Sprintf("raw console logs:\n---\n%s\n---", consoleOutput))
				return false, handledErrors.NewGenericError(fmt.Errorf("probe output corrupted: endingToken encountered before startingToken"))
			}
			a.writeDebugLogs(ctx, "consoleOutput contains data, but probe has not yet printed startingToken, continuing to wait...")
			return false, nil
		}
		if !endingTokenSeen {
			a.writeDebugLogs(ctx, "consoleOutput contains startingToken, but probe has not yet printed endingToken, continuing to wait...")
			return false, nil
		}

		// If we make it this far, we know that both startingTokenSeen and endingTokenSeen are true

		// Separate the probe's output from the rest of the console output (using startingToken and endingToken)
		rawProbeOutput := strings.TrimSpace(helpers.CutBetween(consoleOutput, probe.GetStartingToken(), probe.GetEndingToken()))
		if len(rawProbeOutput) < 1 {
			a.writeDebugLogs(ctx, fmt.Sprintf("raw console logs:\n---\n%s\n---", consoleOutput))
			return false, handledErrors.NewGenericError(fmt.Errorf("probe output corrupted: no data between startingToken and endingToken"))
		}

		a.writeDebugLogs(ctx, fmt.Sprintf("probe output:\n---\n%s\n---", rawProbeOutput))
		probe.ParseProbeOutput(rawProbeOutput, &a.Output)

		return true, nil
	})

	return err
}
