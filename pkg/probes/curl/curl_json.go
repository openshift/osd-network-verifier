package curl

import (
	_ "embed"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/openshift/osd-network-verifier/pkg/data/cloud"
	"github.com/openshift/osd-network-verifier/pkg/data/curlgen"

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

//go:embed systemd-template.sh
var systemdTemplate string

const startingToken = "NV_CURLJSON_BEGIN" //nolint:gosec
const endingToken = "NV_CURLJSON_END"     //nolint:gosec
const outputLinePrefix = "@NV@"

var presetUserDataVariables = map[string]string{
	"USERDATA_BEGIN": startingToken,
	"USERDATA_END":   endingToken,
}

// GetStartingToken returns the string token used to signal the beginning of the probe's output
func (clp Probe) GetStartingToken() string { return startingToken }

// GetEndingToken returns the string token used to signal the end of the probe's output
func (clp Probe) GetEndingToken() string { return endingToken }

// GetMachineImageID returns the string ID of the VM image to be used for the probe instance
func (clp Probe) GetMachineImageID(platformType cloud.Platform, cpuArch cpu.Architecture, region string) (string, error) {
	//Validate platformType
	if !platformType.IsValid() {
		return "", handledErrors.NewGenericError(fmt.Errorf("invalid platform type specified %s", platformType))
	}

	if platformType == cloud.AWSHCP || platformType == cloud.AWSHCPZeroEgress {
		// HCP uses the same AMIs as Classic
		platformType = cloud.AWSClassic
	}

	// Normalize region key (GCP images are global/not region-scoped)
	normalizedRegion := region
	if platformType == cloud.GCPClassic {
		normalizedRegion = "*"
	}

	// Access lookup table
	imageID, keyExists := cloudMachineImageMap[platformType][cpuArch][normalizedRegion]
	if !keyExists {
		return "", fmt.Errorf(
			"no default curl probe machine image for arch %s in region %s of platform %s",
			cpuArch,
			normalizedRegion,
			platformType,
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
	// Use systemd to run curl (instead of cloud-init) if requested. Useful for
	// platforms that don't include cloud-init in their OS images (e.g., GCP)
	if userDataVariables["USE_SYSTEMD"] == "true" {
		userDataTemplate = systemdTemplate
	}

	// Extract required variables specified in template (if any)
	directivelessUserDataTemplate, requiredVariables := helpers.ExtractRequiredVariablesDirective(userDataTemplate)

	// TIMEOUT might be a duration string (e.g., "3s"), but curl only accepts a naked
	// positive decimal number of seconds

	var err error
	userDataVariables["TIMEOUT"], err = normalizeSaneNonzeroDuration(userDataVariables["TIMEOUT"], "%.2f")
	if err != nil {
		return "", fmt.Errorf("invalid userdata variable TIMEOUT: %w", err)
	}
	// Same goes for DELAY, except cloud-init only accepts a positive integer number of seconds
	userDataVariables["DELAY"], err = normalizeSaneNonzeroDuration(userDataVariables["DELAY"], "%.f")
	if err != nil {
		return "", fmt.Errorf("invalid userdata variable DELAY: %w", err)
	}

	curlOptions := curlgen.Options{
		CaPath:          "/etc/pki/tls/certs/",
		ProxyCaPath:     "/etc/pki/tls/certs/",
		Retry:           3,
		MaxTime:         userDataVariables["TIMEOUT"],
		NoTls:           userDataVariables["NOTLS"],
		Urls:            userDataVariables["URLS"],
		TlsDisabledUrls: userDataVariables["TLSDISABLED_URLS"],
	}

	userDataVariables["CURL_COMMAND"], err = curlgen.GenerateString(&curlOptions, outputLinePrefix)
	if err != nil {
		return "", err
	}

	// Ensure userDataVariables complies with requiredVariables and presetUserDataVariables. See
	// docstring for helpers.ValidateProvidedVariables() for more details
	err = helpers.ValidateProvidedVariables(userDataVariables, presetUserDataVariables, requiredVariables)
	if err != nil {
		return "", err
	}

	// We expect CACERT to be either empty or a base64 encoded PEM-formatted CA certificate string.
	// When a CA certificate is provided, we add it to the system's CA store via cloud-init.
	// Docs: https://cloudinit.readthedocs.io/en/latest/reference/modules.html#ca-certificates
	if cacert := userDataVariables["CACERT"]; cacert != "" {
		type CaCert struct {
			Trusted []string `yaml:"trusted"`
		}
		type CloudConfig struct {
			CaCerts CaCert `yaml:"ca_certs"`
		}

		decodedCert, err := base64.StdEncoding.DecodeString(cacert)
		if err != nil {
			return "", fmt.Errorf("failed to base64 decode provided CA certificate: %w", err)
		}

		cloudInit := CloudConfig{
			CaCerts: CaCert{
				Trusted: []string{
					strings.TrimSpace(string(decodedCert)),
				},
			},
		}

		cloudInitYamlBytes, cloudInitMarshalErr := yaml.Marshal(&cloudInit)
		if cloudInitMarshalErr != nil {
			return "", fmt.Errorf("unable to create cloud init config: %w", cloudInitMarshalErr)
		}

		userDataVariables["CACERT_RENDERED"] = strings.TrimSpace(string(cloudInitYamlBytes))
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
// When ensurePrivate is set to true, will not only check the endpoint is accessible, but also ensure the endpoint is private
func (clp Probe) ParseProbeOutput(ensurePrivate bool, probeOutput string, outputDestination *output.Output) {
	// probeOutput first needs to be "repaired" due to curl and AWS bugs
	repairedProbeOutput := helpers.FixLeadingZerosInJSON(helpers.RemoveTimestamps(probeOutput))
	probeResults, errMap := bulkDeserializeCurlJSONProbeResult(repairedProbeOutput)
	for _, probeResult := range probeResults {
		outputDestination.AddDebugLogs(fmt.Sprintf("%+v\n", probeResult))
		if !probeResult.IsSuccessfulConnection() {
			// Replace "telnet" with "tcp" in output to prevent confusion over a probe
			// implementation detail
			url := strings.Replace(probeResult.URL, "telnet", "tcp", 1)
			outputDestination.SetEgressFailures(
				[]string{fmt.Sprintf("%s (%s)", url, probeResult.ErrorMsg)},
			)
		}
		// when ensurePrivate is set to true, we need to make sure the returned IP address is private
		if ensurePrivate {
			remoteIP := net.ParseIP(probeResult.RemoteIP)
			if !remoteIP.IsPrivate() {
				probeResult.ErrorMsg = "The endpoint is non private"
				url := strings.Replace(probeResult.URL, "telnet", "tcp", 1)
				outputDestination.SetEgressFailures(
					[]string{fmt.Sprintf("%s (%s)", url, probeResult.ErrorMsg)})
			}
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
