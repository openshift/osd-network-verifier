package legacy

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"regexp"

	handledErrors "github.com/openshift/osd-network-verifier/pkg/errors"
	"github.com/openshift/osd-network-verifier/pkg/helpers"
	"github.com/openshift/osd-network-verifier/pkg/output"
)

type LegacyProbe struct{}

//go:embed userdata-template.yaml
var userDataTemplate string

const (
	startingToken = "USERDATA BEGIN"
	endingToken   = "USERDATA END"
)

var presetUserDataVariables = map[string]string{
	// The legacy probe had tokens marking the starts/ends of both the probe's docker image
	// and the userdata script as a whole. The parsing logic expects the userdata's entire output,
	// so we map startingToken and endingToken to userdata-marking tokens accordingly
	"USERDATA_BEGIN":           startingToken,
	"USERDATA_END":             endingToken,
	"VALIDATOR_START_VERIFIER": "VALIDATOR START",
	"VALIDATOR_END_VERIFIER":   "VALIDATOR END",
	// IMAGE="$IMAGE" is a legacy hack allowing us to sneak normal shell variables past os.Expand
	"IMAGE": "$IMAGE",
}

// GetStartingToken returns the string token used to signal the beginning of the probe's output
func (prb LegacyProbe) GetStartingToken() string { return startingToken }

// GetEndingToken returns the string token used to signal the end of the probe's output
func (prb LegacyProbe) GetEndingToken() string { return endingToken }

// GetMachineImageID returns the string ID of the VM image to be used for the probe instance
func (prb LegacyProbe) GetMachineImageID(platformType string, cpuArch string, region string) (string, error) {
	// Validate/normalize platformType
	normalizedPlatformType, err := helpers.GetPlatformType(platformType)
	if err != nil {
		return "", err
	}

	// Access lookup table
	imageID, keyExists := cloudMachineImageMap[normalizedPlatformType][cpuArch][region]
	if !keyExists {
		return "", fmt.Errorf(
			"no default LegacyProbe machine image for arch %s in region %s of platform %s",
			cpuArch,
			region,
			normalizedPlatformType,
		)
	}

	return imageID, nil
}

// GetExpandedUserData returns a YAML-formatted userdata string filled-in ("expanded") with
// the values provided in userDataVariables according to os.Expand(). E.g., if the userdata
// template contains "name: $FOO" and userDataVariables = {"FOO": "bar"}, the returned string
// will contain "name: bar". Errors will be returned if values aren't provided for required
// variables listed in the template's "network-verifier-required-variables" directive, or if
// values *are* provided for variables that must be set to a certain value for the probe to
// function correctly (presetUserDataVariables) -- this function will fill-in those values for you.
func (prb LegacyProbe) GetExpandedUserData(userDataVariables map[string]string) (string, error) {
	// Extract required variables specified in template (if any)
	directivelessUserDataTemplate, requiredVariables := helpers.ExtractRequiredVariablesDirective(userDataTemplate)

	// Ensure userDataVariables complies with requiredVariables and presetUserDataVariables. See
	// docstring for helpers.ValidateProvidedVariables() for more details
	err := helpers.ValidateProvidedVariables(userDataVariables, presetUserDataVariables, requiredVariables)
	if err != nil {
		return "", err
	}

	// Expand template
	return os.Expand(directivelessUserDataTemplate, func(userDataVar string) string {
		if presetVal, isPreset := presetUserDataVariables[userDataVar]; isPreset {
			return presetVal
		}
		return userDataVariables[userDataVar]
	}), nil
}

// ParseProbeOutput accepts a string containing all probe output that appeared between
// the startingToken and the endingToken and a pointer to an Output object. outputDestination
// will be filled with the results from the egress check
func (prb LegacyProbe) ParseProbeOutput(probeOutput string, outputDestination *output.Output) {
	// reSuccess indicates that network validation was successful
	reSuccess := regexp.MustCompile(`Success!`)

	// Check if the result is success
	success := reSuccess.FindAllStringSubmatch(probeOutput, -1)
	if len(success) > 0 {
		return
	}

	if isGenericErrorPresent(probeOutput, outputDestination) {
		// isGenericErrorPresent will add any generic errors to outputDestination
		outputDestination.AddDebugLogs("generic error found - please help us classify this by sharing it with us so that we can provide a more specific error message")
	}

	if isEgressFailurePresent(probeOutput, outputDestination) {
		// isGenericErrorPresent will add any egress errors to outputDestination
		outputDestination.AddDebugLogs("egress failures found")
	}
}

// isGenericErrorPresent checks consoleOutput for generic (unclassified) failures
func isGenericErrorPresent(consoleOutput string, outputDestination *output.Output) bool {
	// reGenericFailure is an attempt at a catch-all to help debug failures that we have not accounted for yet
	reGenericFailure := regexp.MustCompile(`(?m)^(.*Cannot.*)|(.*Could not.*)|(.*Failed.*)|(.*command not found.*)`)
	// reRetryAttempt will override reGenericFailure when matching against attempts to retry pulling a container image
	reRetryAttempt := regexp.MustCompile(`Failed, retrying in`)

	found := false

	genericFailures := reGenericFailure.FindAllString(consoleOutput, -1)
	if len(genericFailures) > 0 {
		for _, failure := range genericFailures {
			switch {
			// Ignore "Failed, retrying in" messages when retrying container image pulls as they are not terminal failures
			case reRetryAttempt.FindAllString(failure, -1) != nil:
				outputDestination.AddDebugLogs(fmt.Sprintf("ignoring failure that is retrying: %s", failure))
			// If we don't otherwise ignore a generic error, consider it one that needs attention
			default:
				outputDestination.AddError(handledErrors.NewGenericError(errors.New(failure)))
				found = true
			}
		}
	}

	return found
}

// isEgressFailurePresent checks consoleOutput for network egress failures and stores them
// as NetworkVerifierErrors in a.Output.failures
func isEgressFailurePresent(probeOutput string, outputDestination *output.Output) bool {
	// reEgressFailures will match a specific egress failure case
	reEgressFailures := regexp.MustCompile(`Unable to reach (\S+)`)
	found := false

	// egressFailures is a 2D slice of regex matches - egressFailures[0] represents a specific regex match
	// egressFailures[0][0] is the "Unable to reach" part of the match
	// egressFailures[0][1] is the "(\S+)" part of the match, i.e. the following string
	egressFailures := reEgressFailures.FindAllStringSubmatch(probeOutput, -1)
	for _, e := range egressFailures {
		if len(e) == 2 {
			outputDestination.SetEgressFailures([]string{e[1]})
			found = true
		}
	}

	return found
}
