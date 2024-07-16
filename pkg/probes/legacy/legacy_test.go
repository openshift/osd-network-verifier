package legacy

import (
	_ "embed"
	"regexp"
	"strings"
	"testing"

	"github.com/openshift/osd-network-verifier/pkg/data/cpu"
	"github.com/openshift/osd-network-verifier/pkg/helpers"
	"github.com/openshift/osd-network-verifier/pkg/output"
	"github.com/openshift/osd-network-verifier/pkg/probes"
	"gopkg.in/yaml.v3"
)

// TestLegacyProbe_ImplementsProbeInterface simply forces the compiler
// to confirm that the CurlJSONProbe type and its Probe alias properly
// implement the Probe interface. If not (e.g, because a required method
// is missing), this test will fail to compile
func TestLegacyProbe_ImplementsProbeInterface(t *testing.T) {
	var _ probes.Probe = (*Probe)(nil)
}

// TestLegacyProbe_GetMachineImageID tests this probe's cloud VM image lookup table
func TestLegacyProbe_GetMachineImageID(t *testing.T) {
	type args struct {
		platformType string
		cpuArch      cpu.Architecture
		region       string
	}
	tests := []struct {
		name string
		args args
		// wantRegex should contain a valid regular expression to be matched
		// against the image ID output.
		wantRegex string
		wantErr   bool
	}{
		{
			name: "AWS happy path",
			args: args{
				platformType: helpers.PlatformAWS,
				cpuArch:      cpu.ArchX86,
				region:       "us-east-1",
			},
			wantRegex: `ami-\w+`,
			wantErr:   false,
		},
		{
			name: "AWS alt platform name",
			args: args{
				platformType: helpers.PlatformAWSClassic,
				cpuArch:      cpu.ArchX86,
				region:       "us-east-1",
			},
			wantRegex: `ami-\w+`,
			wantErr:   false,
		},
		{
			name: "AWS HCP",
			args: args{
				platformType: helpers.PlatformAWSHCP,
				cpuArch:      cpu.ArchX86,
				region:       "us-east-1",
			},
			wantRegex: `ami-\w+`,
			wantErr:   false,
		},
		{
			name: "GCP must error",
			args: args{
				platformType: helpers.PlatformGCP,
				cpuArch:      cpu.ArchX86,
				region:       "europe-west1-c",
			},
			wantErr: true,
		},
		{
			name: "alt-GCP must error",
			args: args{
				platformType: helpers.PlatformGCPClassic,
				cpuArch:      cpu.ArchX86,
				region:       "europe-west1-c",
			},
			wantErr: true,
		},
		{
			name: "ARM must error",
			args: args{
				platformType: helpers.PlatformAWSClassic,
				cpuArch:      cpu.ArchARM,
				region:       "us-east-1",
			},
			wantErr: true,
		},
		{
			name: "bad AWS region",
			args: args{
				platformType: helpers.PlatformAWS,
				cpuArch:      cpu.ArchX86,
				region:       "foobar",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prb := Probe{}
			got, err := prb.GetMachineImageID(tt.args.platformType, tt.args.cpuArch, tt.args.region)
			if (err != nil) != tt.wantErr {
				t.Errorf("legacy.Probe.GetMachineImageID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Check if function's output contains a regex match
			if len(tt.wantRegex) > 0 {
				reWant := regexp.MustCompile(tt.wantRegex)
				if len(reWant.FindString(got)) < 1 {
					t.Errorf("legacy.Probe.GetMachineImageID() output does not match regex `%s`, content=%v", tt.wantRegex, got)
				}
			}
		})
	}
}

// TestLegacyProbe_GetExpandedUserData tests the correctness of the user-
// data produced by the probe. This test is different from most other unit
// tests in that it uses regexes to validate the output string (so that we
// don't have to update this test with each little change to
// userdata-template.yaml) and performs basic YAML syntax validation by
// attempting to yaml.Unmarshal() the output string
func TestLegacyProbe_GetExpandedUserData(t *testing.T) {
	tests := []struct {
		name              string
		userDataVariables map[string]string
		// wantRegex should contain a valid regular expression to be matched
		// against the userdata output. Recommend starting each regex with
		// `#cloud-config[\s\S]*`
		wantRegex string
		wantErr   bool
		// Any test with skipIfNoRequiredVariables==true will be skipped
		// if userdata-template.yaml lacks a "# network-verifier-required-variables"
		// directive
		skipIfNoRequiredVariables bool
	}{
		{
			name: "happy path",
			userDataVariables: map[string]string{
				"AWS_REGION":      "us-east-1",
				"CONFIG_PATH":     "/app/build/config/aws.yaml",
				"NOTLS":           "false",
				"TIMEOUT":         "3s",
				"DELAY":           "5",
				"VALIDATOR_IMAGE": "quay.io/app-sre/osd-network-verifier:v0.1.90-f2e86a9",
				"VALIDATOR_REPO":  "quay.io/app-sre/osd-network-verifier",
			},
			wantRegex: `#cloud-config[\s\S]*docker pull quay.io/app-sre/osd-network-verifier:v0.1.90-f2e86a9[\s\S]*docker run -e "AWS_REGION=us-east-1"[\s\S]*delay: 5`,
		},
		{
			name: "CA cert provided",
			userDataVariables: map[string]string{
				"AWS_REGION":      "ap-southeast-1",
				"CONFIG_PATH":     "/app/build/config/aws.yaml",
				"NOTLS":           "false",
				"TIMEOUT":         "3s",
				"DELAY":           "5",
				"VALIDATOR_IMAGE": "quay.io/app-sre/osd-network-verifier:v0.1.90-f2e86a9",
				"VALIDATOR_REPO":  "quay.io/app-sre/osd-network-verifier",
				"CACERT":          "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNiakNDQWZPZ0F3SUJBZ0lRWXZZeWJPWEU0MmhjRzJMZG5DNmRsVEFLQmdncWhrak9QUVFEQXpCNE1Rc3cKQ1FZRFZRUUdFd0pGVXpFUk1BOEdBMVVFQ2d3SVJrNU5WQzFTUTAweERqQU1CZ05WQkFzTUJVTmxjbVZ6TVJndwpGZ1lEVlFSaERBOVdRVlJGVXkxUk1qZ3lOakF3TkVveExEQXFCZ05WQkFNTUkwRkRJRkpCU1ZvZ1JrNU5WQzFTClEwMGdVMFZTVmtsRVQxSkZVeUJUUlVkVlVrOVRNQjRYRFRFNE1USXlNREE1TXpjek0xb1hEVFF6TVRJeU1EQTUKTXpjek0xb3dlREVMTUFrR0ExVUVCaE1DUlZNeEVUQVBCZ05WQkFvTUNFWk9UVlF0VWtOTk1RNHdEQVlEVlFRTApEQVZEWlhKbGN6RVlNQllHQTFVRVlRd1BWa0ZVUlZNdFVUSTRNall3TURSS01Td3dLZ1lEVlFRRERDTkJReUJTClFVbGFJRVpPVFZRdFVrTk5JRk5GVWxaSlJFOVNSVk1nVTBWSFZWSlBVekIyTUJBR0J5cUdTTTQ5QWdFR0JTdUIKQkFBaUEySUFCUGE2VjFQSXlxdmZOa3BTSWVTWDBvTm5udkJsVWRCZWg4ZEhzVm55VjBlYkFBS1RSQmRwMjBMSApzYkk2R0E2MFhZeXpabDJoTlBrMkxFbmI4MGI4czBScFJCTm0vZGZGL2E4MlRjNERUUWR4ejY5cUJkS2lRMW9LClVtOEJBMDZPaTZOQ01FQXdEd1lEVlIwVEFRSC9CQVV3QXdFQi96QU9CZ05WSFE4QkFmOEVCQU1DQVFZd0hRWUQKVlIwT0JCWUVGQUc1TCsrL0VZWmc4ay9RUVc2cmN4L24wbTVKTUFvR0NDcUdTTTQ5QkFNREEya0FNR1lDTVFDdQpTdU1yUU1OMEVmS1ZyUllqM2s0TUd1WmRwU1JlYTBSNy9EamlUOHVjUlJjUlRCUW5KbFU1ZFVvRHpCT1FuNUlDCk1RRDZTbXhnaUhQejdyaVlZcW5PSzhMWmlxWndNUjJ2c0pSTTYwL0c0OUh6WXFjOC81TXVCMXhKQVdkcEVnSnkKditjPQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==",
			},
			wantRegex: `#cloud-config[\s\S]*echo "LS0tLS1CRUd\w*Cg=="`,
		},
		{
			name: "proxy provided",
			userDataVariables: map[string]string{
				"AWS_REGION":      "eu-west-1",
				"CONFIG_PATH":     "/app/build/config/aws.yaml",
				"NOTLS":           "true",
				"TIMEOUT":         "3s",
				"DELAY":           "5",
				"VALIDATOR_IMAGE": "quay.io/app-sre/osd-network-verifier:v0.1.90-f2e86a9",
				"VALIDATOR_REPO":  "quay.io/app-sre/osd-network-verifier",
				"HTTP_PROXY":      "http://1.2.3.4:567",
				"HTTPS_PROXY":     "https://proxy.example.com:443",
			},
			wantRegex: `#cloud-config[\s\S]*-e "HTTP_PROXY=http://1.2.3.4:567"[\s\S]*HTTPS_PROXY=https://proxy.example.com:443 /run-container.sh`,
		},
		{
			name:                      "missing variables required by directive",
			userDataVariables:         map[string]string{},
			wantErr:                   true,
			skipIfNoRequiredVariables: true,
		},
		{
			name: "input variable conflicts with preset",
			userDataVariables: map[string]string{
				"AWS_REGION":      "us-east-1",
				"CONFIG_PATH":     "/app/build/config/aws.yaml",
				"NOTLS":           "false",
				"TIMEOUT":         "3s",
				"DELAY":           "5",
				"VALIDATOR_IMAGE": "quay.io/app-sre/osd-network-verifier:v0.1.90-f2e86a9",
				"VALIDATOR_REPO":  "quay.io/app-sre/osd-network-verifier",
				"IMAGE":           "plz overwrite preset variable",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipIfNoRequiredVariables && !strings.Contains(userDataTemplate, "network-verifier-required-variables") {
				t.SkipNow()
			}

			prb := Probe{}
			// First check if function is returning an error
			got, err := prb.GetExpandedUserData(tt.userDataVariables)
			if (err != nil) != tt.wantErr {
				t.Errorf("legacy.Probe.GetExpandedUserData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Then check if function's output contains a regex match
			if len(tt.wantRegex) > 0 {
				reWant := regexp.MustCompile(tt.wantRegex)
				if len(reWant.FindString(got)) < 1 {
					t.Errorf("legacy.Probe.GetExpandedUserData() output does not match regex `%s`, content=%v", tt.wantRegex, got)
				}
			}

			// Finally, ensure output is valid YAML
			gotByteSlice := []byte(got)
			var unmarshalled interface{}
			err = yaml.Unmarshal(gotByteSlice, &unmarshalled)
			if err != nil {
				t.Errorf("legacy.Probe.GetExpandedUserData() produced invalid YAML (err: %v), content=%v", err, got)
				return
			}
		})
	}
}

// Test_isGenericErrorPresent checks that this probe's text-parsing helper function can
// accurately detect generic failure messages
func Test_isGenericErrorPresent(t *testing.T) {
	tests := []struct {
		name                string
		consoleOutput       string
		expectGenericErrors bool
	}{
		{
			name: "Retry error",
			consoleOutput: `USERDATA BEGIN
Failed, retrying in 2s to do stuff
Success!
USERDATA END`,
			expectGenericErrors: false,
		},
		{
			name: "Generic error",
			consoleOutput: `USERDATA BEGIN
Failed, retrying in 2s to do stuff
Could not do stuff
USERDATA END`,
			expectGenericErrors: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var outputDestination output.Output

			actual := isGenericErrorPresent(test.consoleOutput, &outputDestination)
			if test.expectGenericErrors != actual {
				t.Errorf("expected %v, got %v", test.expectGenericErrors, actual)
			}

			if test.expectGenericErrors {
				if outputDestination.IsSuccessful() {
					t.Errorf("expected errors, but output still marked as successful")
				}
			}
		})
	}
}

// Test_isEgressFailurePresent checks that this probe's text-parsing helper function can
// accurately detect egress failure messages
func Test_isEgressFailurePresent(t *testing.T) {
	tests := []struct {
		name                   string
		consoleOutput          string
		expectedEgressFailures bool
		expectedCount          int
	}{
		{
			name: "no egress failures",
			consoleOutput: `USERDATA BEGIN
Success!
USERDATA END`,
			expectedEgressFailures: false,
		},
		{
			name: "egress failures present",
			consoleOutput: `USERDATA BEGIN
Unable to reach www.example.com:443 within specified timeout after 3 retries: Get "https://www.example.com": context deadline exceeded (Client.Timeout exceeded while awaiting headers)
Unable to reach www.example.com:80 within specified timeout after 3 retries: Get "http://www.example.com": context deadline exceeded (Client.Timeout exceeded while awaiting headers)
USERDATA END`,
			expectedEgressFailures: true,
			expectedCount:          2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var outputDestination output.Output

			actual := isEgressFailurePresent(test.consoleOutput, &outputDestination)
			if test.expectedEgressFailures != actual {
				t.Errorf("expected %v, got %v", test.expectedEgressFailures, actual)
			}
			failures := outputDestination.GetEgressURLFailures()
			for _, f := range failures {
				t.Log(f.EgressURL())
			}
			if test.expectedCount != len(failures) {
				t.Errorf("expected %v egress failures, got %v", test.expectedCount, len(failures))
			}
		})
	}
}
