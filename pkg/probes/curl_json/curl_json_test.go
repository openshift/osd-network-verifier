package curl_json

import (
	_ "embed"
	"regexp"
	"strings"
	"testing"

	"github.com/openshift/osd-network-verifier/pkg/helpers"
	"github.com/openshift/osd-network-verifier/pkg/probes"
	"gopkg.in/yaml.v3"
)

// TestCurlJSONProbe_ImplementsProbeInterface simply forces the compiler
// to confirm that the CurlJSONProbe type properly implements the Probe
// interface. If not (e.g, because a required method is missing), this
// test will fail to compile
func TestCurlJSONProbe_ImplementsProbeInterface(t *testing.T) {
	var _ probes.Probe = (*CurlJSONProbe)(nil)
}

// TestCurlJSONProbe_GetExpandedUserData tests the correctness of the user-
// data produced by the probe. This test is different from most other unit
// tests in that it uses regexes to validate the output string (so that we
// don't have to update this test with each little change to
// userdata-template.yaml) and performs basic YAML syntax validation by
// attempting to yaml.Unmarshal() the output string
func TestCurlJSONProbe_GetExpandedUserData(t *testing.T) {
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
				"TIMEOUT": "1",
				"DELAY":   "2",
				"URLS":    "http://example.com:80 https://example.org:443",
			},
			wantRegex: `#cloud-config[\s\S]*http:\/\/example.com:80 https:\/\/example.org:443`,
		},
		{
			name: "CA cert provided",
			userDataVariables: map[string]string{
				"TIMEOUT": "1",
				"DELAY":   "2",
				"URLS":    "http://example.com:80 https://example.org:443",
				"CACERT":  "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNiakNDQWZPZ0F3SUJBZ0lRWXZZeWJPWEU0MmhjRzJMZG5DNmRsVEFLQmdncWhrak9QUVFEQXpCNE1Rc3cKQ1FZRFZRUUdFd0pGVXpFUk1BOEdBMVVFQ2d3SVJrNU5WQzFTUTAweERqQU1CZ05WQkFzTUJVTmxjbVZ6TVJndwpGZ1lEVlFSaERBOVdRVlJGVXkxUk1qZ3lOakF3TkVveExEQXFCZ05WQkFNTUkwRkRJRkpCU1ZvZ1JrNU5WQzFTClEwMGdVMFZTVmtsRVQxSkZVeUJUUlVkVlVrOVRNQjRYRFRFNE1USXlNREE1TXpjek0xb1hEVFF6TVRJeU1EQTUKTXpjek0xb3dlREVMTUFrR0ExVUVCaE1DUlZNeEVUQVBCZ05WQkFvTUNFWk9UVlF0VWtOTk1RNHdEQVlEVlFRTApEQVZEWlhKbGN6RVlNQllHQTFVRVlRd1BWa0ZVUlZNdFVUSTRNall3TURSS01Td3dLZ1lEVlFRRERDTkJReUJTClFVbGFJRVpPVFZRdFVrTk5JRk5GVWxaSlJFOVNSVk1nVTBWSFZWSlBVekIyTUJBR0J5cUdTTTQ5QWdFR0JTdUIKQkFBaUEySUFCUGE2VjFQSXlxdmZOa3BTSWVTWDBvTm5udkJsVWRCZWg4ZEhzVm55VjBlYkFBS1RSQmRwMjBMSApzYkk2R0E2MFhZeXpabDJoTlBrMkxFbmI4MGI4czBScFJCTm0vZGZGL2E4MlRjNERUUWR4ejY5cUJkS2lRMW9LClVtOEJBMDZPaTZOQ01FQXdEd1lEVlIwVEFRSC9CQVV3QXdFQi96QU9CZ05WSFE4QkFmOEVCQU1DQVFZd0hRWUQKVlIwT0JCWUVGQUc1TCsrL0VZWmc4ay9RUVc2cmN4L24wbTVKTUFvR0NDcUdTTTQ5QkFNREEya0FNR1lDTVFDdQpTdU1yUU1OMEVmS1ZyUllqM2s0TUd1WmRwU1JlYTBSNy9EamlUOHVjUlJjUlRCUW5KbFU1ZFVvRHpCT1FuNUlDCk1RRDZTbXhnaUhQejdyaVlZcW5PSzhMWmlxWndNUjJ2c0pSTTYwL0c0OUh6WXFjOC81TXVCMXhKQVdkcEVnSnkKditjPQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==",
			},
			wantRegex: `#cloud-config[\s\S]*proxy.pem[\s\S]*encoding: b64[\s\S]*LS0tLS1CRUd\w*Cg==\n[\s\S]*https://example.org:443`,
		},
		{
			name: "set NOTLS",
			userDataVariables: map[string]string{
				"TIMEOUT": "1",
				"DELAY":   "2",
				"URLS":    "http://example.com:80 https://example.org:443",
				"NOTLS":   "True",
			},
			wantRegex: `#cloud-config[\s\S]* -k [\s\S]*http:\/\/example.com:80 https:\/\/example.org:443`,
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
				"TIMEOUT":        "1",
				"DELAY":          "2",
				"URLS":           "http://example.com:80 https://example.org:443",
				"USERDATA_BEGIN": "foobar",
			},
			wantErr: true,
		},
		{
			name: "invalid DELAY",
			userDataVariables: map[string]string{
				"TIMEOUT": "1",
				"DELAY":   "-2",
				"URLS":    "http://example.com:80 https://example.org:443",
			},
			wantErr: true,
		},
		{
			name: "invalid TIMEOUT",
			userDataVariables: map[string]string{
				"TIMEOUT": "-1",
				"DELAY":   "2",
				"URLS":    "http://example.com:80 https://example.org:443",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipIfNoRequiredVariables && !strings.Contains(userDataTemplate, "network-verifier-required-variables") {
				t.SkipNow()
			}

			prb := CurlJSONProbe{}
			// First check if function is returning an error
			got, err := prb.GetExpandedUserData(tt.userDataVariables)
			if (err != nil) != tt.wantErr {
				t.Errorf("CurlJSONProbe.GetExpandedUserData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Then check if function's output contains a regex match
			if len(tt.wantRegex) > 0 {
				reWant := regexp.MustCompile(tt.wantRegex)
				if len(reWant.FindString(got)) < 1 {
					t.Errorf("CurlJSONProbe.GetExpandedUserData() output does not match regex `%s`, content=%v", tt.wantRegex, got)
				}
			}

			// Finally, ensure output is valid YAML
			gotByteSlice := []byte(got)
			var unmarshalled interface{}
			err = yaml.Unmarshal(gotByteSlice, &unmarshalled)
			if err != nil {
				t.Errorf("CurlJSONProbe.GetExpandedUserData() produced invalid YAML (err: %v), content=%v", err, got)
				return
			}
		})
	}
}

// TestCurlJSONProbe_UserDataTemplateContainsDeclaredVariables ensures
// that this probe's userdata-template.yaml contains all of the variables
// required by the template itself (using #network-verifier-required-variables)
// and by TestCurlJSONProbe's presetUserDataVariables (defined in curl_json.go)
func TestCurlJSONProbe_UserDataTemplateContainsDeclaredVariables(t *testing.T) {
	// Check preset variables
	for presetVariableName := range presetUserDataVariables {
		if !strings.Contains(userDataTemplate, "${"+presetVariableName+"}") {
			t.Errorf("CurlJSONProbe.presetUserDataVariables has key %[1]s, but could not find required '${%[1]s}' in probe's userdata-template.yaml", presetVariableName)
			return
		}
	}

	// Check required variables
	directivelessUserDataTemplate, requiredVariables := helpers.ExtractRequiredVariablesDirective(userDataTemplate)
	for _, requiredVariableName := range requiredVariables {
		if !strings.Contains(directivelessUserDataTemplate, "${"+requiredVariableName+"}") {
			t.Errorf("CurlJSONProbe's userdata-template.yaml declares %[1]s as required, but could not find '${%[1]s}' in file", requiredVariableName)
			return
		}
	}

}

// TestCurlJSONProbe_GetMachineImageID tests this probe's cloud VM image lookup table
func TestCurlJSONProbe_GetMachineImageID(t *testing.T) {
	type args struct {
		platformType string
		cpuArch      string
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
				cpuArch:      helpers.ArchX86,
				region:       "us-east-1",
			},
			wantRegex: `ami-\w+`,
			wantErr:   false,
		},
		{
			name: "GCP happy path",
			args: args{
				platformType: helpers.PlatformGCP,
				cpuArch:      helpers.ArchX86,
				region:       "europe-west1-c",
			},
			wantRegex: `rhel-\d`,
			wantErr:   false,
		},
		{
			name: "AWS alt platform name",
			args: args{
				platformType: helpers.PlatformAWSClassic,
				cpuArch:      helpers.ArchX86,
				region:       "us-east-1",
			},
			wantRegex: `ami-\w+`,
			wantErr:   false,
		},
		{
			name: "GCP alt platform name",
			args: args{
				platformType: helpers.PlatformGCPClassic,
				cpuArch:      helpers.ArchX86,
				region:       "europe-west1-c",
			},
			wantRegex: `rhel-\d`,
			wantErr:   false,
		},
		{
			name: "AWS ARM",
			args: args{
				platformType: helpers.PlatformAWSClassic,
				cpuArch:      helpers.ArchARM,
				region:       "us-east-1",
			},
			wantRegex: `ami-\w+`,
			wantErr:   false,
		},
		{
			name: "GCP ARM",
			args: args{
				platformType: helpers.PlatformGCP,
				cpuArch:      helpers.ArchARM,
				region:       "europe-west1-c",
			},
			wantRegex: `rhel-\d-arm64`,
			wantErr:   false,
		},
		{
			name: "bad plaform",
			args: args{
				platformType: "foobar",
				cpuArch:      helpers.ArchX86,
				region:       "europe-west1-c",
			},
			wantErr: true,
		},
		{
			name: "bad arch",
			args: args{
				platformType: helpers.PlatformGCP,
				cpuArch:      "foobar",
				region:       "europe-west1-c",
			},
			wantErr: true,
		},
		{
			name: "bad AWS region",
			args: args{
				platformType: helpers.PlatformAWS,
				cpuArch:      helpers.ArchX86,
				region:       "foobar",
			},
			wantErr: true,
		},
		{
			name: "ignore bad GCP region",
			args: args{
				platformType: helpers.PlatformGCP,
				cpuArch:      helpers.ArchX86,
				region:       "foobar",
			},
			wantRegex: `rhel-\d`,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prb := CurlJSONProbe{}
			got, err := prb.GetMachineImageID(tt.args.platformType, tt.args.cpuArch, tt.args.region)
			if (err != nil) != tt.wantErr {
				t.Errorf("CurlJSONProbe.GetMachineImageID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Check if function's output contains a regex match
			if len(tt.wantRegex) > 0 {
				reWant := regexp.MustCompile(tt.wantRegex)
				if len(reWant.FindString(got)) < 1 {
					t.Errorf("CurlJSONProbe.GetMachineImageID() output does not match regex `%s`, content=%v", tt.wantRegex, got)
				}
			}
		})
	}
}

// Test_normalizeSaneNonzeroDuration tests this probe's duration string normalization
// and sanity checking function. This test assumes fmtStr = "%.2f"
func Test_normalizeSaneNonzeroDuration(t *testing.T) {
	tests := []struct {
		name                string
		possibleDurationStr string
		want                string
		wantErr             bool
	}{
		{
			name:                "integer with unit",
			possibleDurationStr: "3s",
			want:                "3.00",
			wantErr:             false,
		},
		{
			name:                "float with unit",
			possibleDurationStr: "1.2s",
			want:                "1.20",
			wantErr:             false,
		},
		{
			name:                "bare integer",
			possibleDurationStr: "7",
			want:                "7.00",
			wantErr:             false,
		},
		{
			name:                "bare float",
			possibleDurationStr: "6.5",
			want:                "6.50",
			wantErr:             false,
		},
		{
			name:                "too large",
			possibleDurationStr: "4h",
			wantErr:             true,
		},
		{
			name:                "negative integer",
			possibleDurationStr: "-3",
			wantErr:             true,
		},
		{
			name:                "negative with unit",
			possibleDurationStr: "-1m",
			wantErr:             true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeSaneNonzeroDuration(tt.possibleDurationStr, "%.2f")
			if (err != nil) != tt.wantErr {
				t.Errorf("normalizeSaneNonzeroDuration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("normalizeSaneNonzeroDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}
