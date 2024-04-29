package curl_json

import (
	_ "embed"
	"fmt"
	"os"

	handledErrors "github.com/openshift/osd-network-verifier/pkg/errors"
	"github.com/openshift/osd-network-verifier/pkg/helpers"
	"github.com/openshift/osd-network-verifier/pkg/output"
)

type CurlJSONProbe struct{}

//go:embed userdata-template.yaml
var userDataTemplate string

const startingToken = "NV_CURLJSON_BEGIN"
const endingToken = "NV_CURLJSON_END"
const outputLinePrefix = "@NV@"

var presetUserDataVariables = map[string]string{
	"USERDATA_BEGIN": startingToken,
	"USERDATA_END":   endingToken,
	"LINE_PREFIX":    outputLinePrefix,
}

// GetStartingToken returns the string token used to signal the beginning of the probe's output
func (prb CurlJSONProbe) GetStartingToken() string { return startingToken }

// GetStartingToken returns the string token used to signal the end of the probe's output
func (prb CurlJSONProbe) GetEndingToken() string { return endingToken }

// GetExpandedUserData returns a YAML-formatted userdata string filled-in ("expanded") with
// the values provided in userDataVariables according to os.Expand(). E.g., if the userdata
// template contains "name: $FOO" and userDataVariables = {"FOO": "bar"}, the returned string
// will contain "name: bar". Errors will be returned if values aren't provided for required
// variables listed in the template's "network-verifier-required-variables" directive, or if
// values *are* provided for variables that must be set to a certain value for the probe to
// function correctly (presetUserDataVariables) -- this function will fill-in those values for you.
func (prb CurlJSONProbe) GetExpandedUserData(userDataVariables map[string]string) (string, error) {
	// Extract required variables specified in template and check for presence
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
func (prb CurlJSONProbe) ParseProbeOutput(probeOutput string, outputDestination *output.Output) {
	probeResults, errMap := bulkDeserializeCurlJSONProbeResult(helpers.FixLeadingZerosInJSON(probeOutput))
	for _, probeResult := range probeResults {
		outputDestination.AddDebugLogs(fmt.Sprintf("%+v\n", probeResult))
		if !probeResult.isSuccessfulConnection() {
			outputDestination.SetEgressFailures(
				[]string{fmt.Sprintf("%s (%s)", probeResult.URL, probeResult.ErrorMsg)},
			)
		}
	}
	for lineNum, err := range errMap {
		outputDestination.AddError(
			handledErrors.NewGenericError(
				fmt.Errorf("error processing line %d: %w", lineNum, err),
			),
		)
	}
}
