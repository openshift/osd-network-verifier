// Experimental curl-based probe shim
// Allows the verifier client to use the experimental probe interface
// This is just a shim to allow for testing until we deprecate the legacy probe code
package awsverifier

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"time"

	awsTools "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	handledErrors "github.com/openshift/osd-network-verifier/pkg/errors"
	"github.com/openshift/osd-network-verifier/pkg/helpers"
	"github.com/openshift/osd-network-verifier/pkg/probes"
)

func (a *AwsVerifier) findUnreachableEndpointsExperimental(ctx context.Context, instanceID string, probe probes.Probe) error {
	var (
		b64ConsoleLogs string
		consoleLogs    string
	)

	// reUserDataComplete indicates that the network validation completed
	reUserDataComplete := regexp.MustCompile(probe.GetEndingToken())
	// // reSuccess indicates that network validation was successful
	// reSuccess := regexp.MustCompile(`Success!`)
	// // rePrepulledImage indicates that the network verifier is using a prepulled image
	// rePrepulledImage := regexp.MustCompile(prepulledImageMessage)

	a.writeDebugLogs(ctx, "Scraping console output and waiting for user data script to complete...")

	// Periodically scrape console output and analyze the logs for any errors or a successful completion
	err := helpers.PollImmediate(30*time.Second, 4*time.Minute, func() (bool, error) {
		consoleOutput, err := a.AwsClient.GetConsoleOutput(ctx, &ec2.GetConsoleOutputInput{
			InstanceId: awsTools.String(instanceID),
			Latest:     awsTools.Bool(true),
		})
		if err != nil {
			return false, handledErrors.NewGenericError(err)
		}

		if consoleOutput.Output != nil {
			// In the early stages, an ec2 instance may be running but the console is not populated with any data
			if len(*consoleOutput.Output) == 0 {
				a.writeDebugLogs(ctx, "EC2 console consoleOutput not yet populated with data, continuing to wait...")
				return false, nil
			}

			// Store base64-encoded output for debug logs
			b64ConsoleLogs = *consoleOutput.Output

			// The console consoleOutput starts out base64 encoded
			scriptOutput, err := base64.StdEncoding.DecodeString(*consoleOutput.Output)
			if err != nil {
				a.writeDebugLogs(ctx, fmt.Sprintf("Error decoding console consoleOutput, will retry on next check interval: %s", err))
				return false, nil
			}

			consoleLogs = string(scriptOutput)

			// Check for the specific string we consoleOutput in the generated userdata file at the end to verify the userdata script has run
			// It is possible we get EC2 console consoleOutput, but the userdata script has not yet completed.
			userDataComplete := reUserDataComplete.FindString(consoleLogs)
			if len(userDataComplete) < 1 {
				a.writeDebugLogs(ctx, "EC2 console consoleOutput contains data, but end of userdata script not seen, continuing to wait...")
				return false, nil
			}

			// Check if the result is success
			// success := reSuccess.FindAllStringSubmatch(consoleLogs, -1)
			// if len(success) > 0 {
			// 	return true, nil
			// }

			// // Add a message to debug logs if we're using the prepulled image
			// prepulledImage := rePrepulledImage.FindAllString(consoleLogs, -1)
			// if len(prepulledImage) > 0 {
			// 	a.writeDebugLogs(ctx, prepulledImageMessage)
			// }

			// if a.isGenericErrorPresent(ctx, consoleLogs) {
			// 	a.writeDebugLogs(ctx, "generic error found - please help us classify this by sharing it with us so that we can provide a more specific error message")
			// }

			// If debug logging is enabled, consoleOutput the full console log that appears to include the full userdata run
			a.writeDebugLogs(ctx, fmt.Sprintf("base64-encoded console logs:\n---\n%s\n---", b64ConsoleLogs))

			// if a.isEgressFailurePresent(string(scriptOutput)) {
			// 	a.writeDebugLogs(ctx, "egress failures found")
			// }
			probe.ParseProbeOutput(string(scriptOutput), &a.Output)

			return true, nil // finalize as there's `userdata end`
		}

		if len(b64ConsoleLogs) > 0 {
			a.writeDebugLogs(ctx, fmt.Sprintf("base64-encoded console logs:\n---\n%s\n---", b64ConsoleLogs))
		}

		return false, nil
	})

	return err
}
