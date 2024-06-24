package curl_json

import (
	_ "embed"
	"fmt"
	"os"
	"strconv"

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

// GetEndingToken returns the string token used to signal the end of the probe's output
func (prb CurlJSONProbe) GetEndingToken() string { return endingToken }

// GetMachineImageID returns the string ID of the VM image to be used for the probe instance
func (prb CurlJSONProbe) GetMachineImageID(platformType string, cpuArch string, region string) (string, error) {
	// Validate/normalize platformType
	normalizedPlatformType, err := helpers.GetPlatformType(platformType)
	if err != nil {
		return "", err
	}

	// Normalize region key (GCP images are global/not region-scoped)
	normalizedRegion := region
	if normalizedPlatformType == helpers.PlatformGCP {
		normalizedRegion = "*"
	}

	// Access lookup table
	imageID, keyExists := cloudMachineImageMap[normalizedPlatformType][cpuArch][normalizedRegion]
	if !keyExists {
		return "", fmt.Errorf(
			"no default CurlJSONProbe machine image for arch %s in region %s of platform %s",
			cpuArch,
			normalizedRegion,
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
func (prb CurlJSONProbe) GetExpandedUserData(userDataVariables map[string]string) (string, error) {
	// Extract required variables specified in template (if any)
	directivelessUserDataTemplate, requiredVariables := helpers.ExtractRequiredVariablesDirective(userDataTemplate)

	// Ensure userDataVariables complies with requiredVariables and presetUserDataVariables. See
	// docstring for helpers.ValidateProvidedVariables() for more details
	err := helpers.ValidateProvidedVariables(userDataVariables, presetUserDataVariables, requiredVariables)
	if err != nil {
		return "", err
	}

	// TIMEOUT might be a duration string (e.g., "3s"), but curl only accepts a naked
	// positive decimal number of seconds
	userDataVariables["TIMEOUT"], err = normalizeSaneNonzeroDuration(userDataVariables["TIMEOUT"], "%.2f")
	if err != nil {
		return "", fmt.Errorf("invalid userdata variable TIMEOUT: %w", err)
	}
	// Same goes for DELAY, except cloud-init only accepts a positive integer number of seconds
	userDataVariables["DELAY"], err = normalizeSaneNonzeroDuration(userDataVariables["DELAY"], "%.f")
	if err != nil {
		return "", fmt.Errorf("invalid userdata variable DELAY: %w", err)
	}

	// For compatibility reasons, we expect CACERT to be either empty or a base64-encoded
	// PEM-formatted CA certicate. When one is provided, we "render" it with a cloud-init
	// preamble that writes the file to disk, and we tell curl about the cert file via flag
	if userDataVariables["CACERT"] != "" {
		cloudInitPreamble := `write_files:
- path: /proxy.pem
  permissions: '0755'
  encoding: b64
  content: `
		userDataVariables["CACERT_RENDERED"] = cloudInitPreamble + userDataVariables["CACERT"]
		userDataVariables["CURLOPT"] += " --cacert /proxy.pem "
	}

	// Also for compatibility reasons, we map the NOTLS variable to curl's "insecure" flag
	if userDataVariables["NOTLS"] != "" {
		noTLS, err := strconv.ParseBool(userDataVariables["NOTLS"])
		if err != nil {
			return "", fmt.Errorf("invalid userdata variable NOTLS: %w", err)
		}
		if noTLS {
			userDataVariables["CURLOPT"] += " -k "
		}
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
	// probeOutput first needs to be "repaired" due to curl and AWS bugs
	repairedProbeOutput := helpers.FixLeadingZerosInJSON(helpers.RemoveTimestamps(probeOutput))
	probeResults, errMap := bulkDeserializeCurlJSONProbeResult(repairedProbeOutput)
	for _, probeResult := range probeResults {
		outputDestination.AddDebugLogs(fmt.Sprintf("%+v\n", probeResult))
		if !probeResult.IsSuccessfulConnection() {
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

// normalizeSaneNonzeroDuration first converts a given string expected to hold a duration
// (e.g., "3s" or "2") to a float64 using helpers.DurationToBareSeconds(). It then ensures the
// float duration is "sane," i.e., greater than 0 seconds but less than 3 hours*. If sane, the
// duration in seconds is Sprintf'd using the provided fmtStr and returned. If not sane, an error
// is returned.
// * We max at 3 hours under the assumption that the verifier isn't doing anything for >3hrs
func normalizeSaneNonzeroDuration(possibleDurationStr string, fmtStr string) (string, error) {
	durationSeconds := helpers.DurationToBareSeconds(possibleDurationStr)
	if durationSeconds <= 0 {
		return "", fmt.Errorf("invalid %s value (parsed as %.2f sec)", possibleDurationStr, durationSeconds)
	}

	if durationSeconds > 10800 {
		return "", fmt.Errorf("value %s (parsed as %.2f sec) is too large", possibleDurationStr, durationSeconds)
	}

	return fmt.Sprintf(fmtStr, durationSeconds), nil
}
