package legacy

import (
	_ "embed"
	"regexp"
	"testing"

	"github.com/openshift/osd-network-verifier/pkg/helpers"
	"github.com/openshift/osd-network-verifier/pkg/probes"
)

// TestLegacyProbe_ImplementsProbeInterface simply forces the compiler
// to confirm that the LegacyProbe type properly implements the Probe
// interface. If not (e.g, because a required method is missing), this
// test will fail to compile
func TestLegacyProbe_ImplementsProbeInterface(t *testing.T) {
	var _ probes.Probe = (*LegacyProbe)(nil)
}

// TestLegacyProbe_GetMachineImageID tests this probe's cloud VM image lookup table
func TestLegacyProbe_GetMachineImageID(t *testing.T) {
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
			name: "GCP must error",
			args: args{
				platformType: helpers.PlatformGCP,
				cpuArch:      helpers.ArchX86,
				region:       "europe-west1-c",
			},
			wantErr: true,
		},
		{
			name: "alt-GCP must error",
			args: args{
				platformType: helpers.PlatformGCPClassic,
				cpuArch:      helpers.ArchX86,
				region:       "europe-west1-c",
			},
			wantErr: true,
		},
		{
			name: "ARM must error",
			args: args{
				platformType: helpers.PlatformAWSClassic,
				cpuArch:      helpers.ArchARM,
				region:       "us-east-1",
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prb := LegacyProbe{}
			got, err := prb.GetMachineImageID(tt.args.platformType, tt.args.cpuArch, tt.args.region)
			if (err != nil) != tt.wantErr {
				t.Errorf("LegacyProbe.GetMachineImageID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Check if function's output contains a regex match
			if len(tt.wantRegex) > 0 {
				reWant := regexp.MustCompile(tt.wantRegex)
				if len(reWant.FindString(got)) < 1 {
					t.Errorf("LegacyProbe.GetMachineImageID() output does not match regex `%s`, content=%v", tt.wantRegex, got)
				}
			}
		})
	}
}
