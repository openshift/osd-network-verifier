package curl

import (
	_ "embed"
	"fmt"
	"os"
	"strconv"

	"github.com/openshift/osd-network-verifier/pkg/data/cpu"
	handledErrors "github.com/openshift/osd-network-verifier/pkg/errors"
	"github.com/openshift/osd-network-verifier/pkg/helpers"
	"github.com/openshift/osd-network-verifier/pkg/output"
)

// curl.Probe is an implementation of the probes.Probe interface that uses the venerable curl tool to
// check for blocked egresses in a target network. It launches an unmodified RHEL9 instance (although
// any OS with curl v7.76.1 or compatible will work) and uses userdata to transmit at runtime a list
// of egress URLs to which curl will attempt to connect. Curl will return the results as JSON via
// serial console, which this probe can then parse into a standard output format. Any reported
// egressURL errors will contain curl's detailed error messages. Additional command line options can
// be provided to curl via the CURLOPT userdataVariable. This probe has been confirmed to support X86
// instances on AWS. In theory, it should also support GCP and any CPU architecture supported by RHEL.
type Probe struct{}

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
func (clp Probe) GetStartingToken() string { return startingToken }

// GetEndingToken returns the string token used to signal the end of the probe's output
func (clp Probe) GetEndingToken() string { return endingToken }

// GetMachineImageID returns the string ID of the VM image to be used for the probe instance
func (clp Probe) GetMachineImageID(platformType string, cpuArch cpu.Architecture, region string) (string, error) {
	// Validate/normalize platformType
	normalizedPlatformType, err := helpers.GetPlatformType(platformType)
	if err != nil {
		return "", err
	}
	if normalizedPlatformType == helpers.PlatformHostedCluster {
		// HCP uses the same AMIs as Classic
		normalizedPlatformType = helpers.PlatformAWS
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
			"no default curl probe machine image for arch %s in region %s of platform %s",
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
func (clp Probe) GetExpandedUserData(userDataVariables map[string]string) (string, error) {
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
			// In addition to adding the curl flag, we can merge the list of "tlsDisabled" URLs
			// with the list of "normal" URLs now (since all URLs will be "tlsDisabled")
			userDataVariables["URLS"] += " " + userDataVariables["TLSDISABLED_URLS"]
			userDataVariables["TLSDISABLED_URLS"] = ""
		}
	}

	// Assuming NOTLS=false, "tlsDisabled" URLs must have curl's "--insecure" flag applied *only* to them.
	// We use curl's "parser reset" flag ("--next" or "-:") to do this, but this has the unfortunate side
	// effect of forcing us to re-pass most curl flags (except global flags and those irrelevant to HTTPS)
	if userDataVariables["TLSDISABLED_URLS"] != "" {
		userDataVariables["TLSDISABLED_URLS_RENDERED"] = fmt.Sprintf(
			// TODO consider a better way of keeping this in sync with what's in userdata-template.yaml?
			`--next -k --retry 3 --retry-connrefused -s -I -m %s -w "%%{stderr}%s%%{json}\n" %s %s --proto =https`,
			userDataVariables["TIMEOUT"],
			outputLinePrefix,
			userDataVariables["CURLOPT"],
			userDataVariables["TLSDISABLED_URLS"],
		)
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
func (clp Probe) ParseProbeOutput(probeOutput string, outputDestination *output.Output) {
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
