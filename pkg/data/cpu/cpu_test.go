package cpu

import (
	"testing"

	"github.com/openshift/osd-network-verifier/pkg/helpers"
)

func TestArchitecture_DefaultInstanceType(t *testing.T) {
	tests := []struct {
		name         string
		arch         Architecture
		platformType string
		want         string
		wantErr      bool
	}{
		{
			name:         "happy path",
			arch:         ArchX86,
			platformType: helpers.PlatformAWS,
			want:         "t3.micro",
			wantErr:      false,
		},
		{
			name:         "alt platform name",
			arch:         ArchARM,
			platformType: helpers.PlatformAWSClassic,
			want:         "t4g.micro",
			wantErr:      false,
		},
		{
			name:         "HCP",
			arch:         ArchX86,
			platformType: helpers.PlatformAWSHCP,
			want:         "t3.micro",
			wantErr:      false,
		},
		{
			name:         "GCP",
			arch:         ArchARM,
			platformType: helpers.PlatformGCPClassic,
			want:         "t2a-standard-1",
			wantErr:      false,
		},
		{
			name:         "invalid platform",
			arch:         ArchARM,
			platformType: "foobar",
			wantErr:      true,
		},
		{
			name:         "fake Architecture",
			arch:         "foobar",
			platformType: helpers.PlatformAWS,
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.arch.DefaultInstanceType(tt.platformType)
			if (err != nil) != tt.wantErr {
				t.Errorf("Architecture.DefaultInstanceType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Architecture.DefaultInstanceType() = %v, want %v", got, tt.want)
			}
		})
	}
}
