package cpu

import (
	"testing"

	"github.com/openshift/osd-network-verifier/pkg/helpers"
)

// TestArchitecture_Comparable simply forces the compiler to confirm that the Architecture type
// is comparable. If not (e.g, because a non-comparable field was added to the struct type), this
// test will fail to compile
func TestArchitecture_Comparable(t *testing.T) {
	if ArchARM != ArchX86 {
		return
	}
}

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

func TestArchitecture_String(t *testing.T) {
	tests := []struct {
		name string
		arch Architecture
		want string
	}{
		{
			name: "x86",
			arch: ArchX86,
			want: "x86",
		},
		{
			name: "arm",
			arch: ArchARM,
			want: "arm",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			arch := tt.arch
			if got := arch.String(); got != tt.want {
				t.Errorf("Architecture.String() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestArchitecture_IsValid(t *testing.T) {
	type fields struct {
		names                  [3]string
		defaultAWSInstanceType string
		defaultGCPInstanceType string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "X86 happy path",
			fields: fields(ArchX86),
			want:   true,
		},
		{
			name:   "ARM happy path",
			fields: fields(ArchARM),
			want:   true,
		},
		{
			name: "fake arch",
			fields: fields{
				names:                  [3]string{"foo", "bar", "baz"},
				defaultAWSInstanceType: "foobar",
				defaultGCPInstanceType: "barfoo",
			},
			want: false,
		},
		{
			name:   "empty arch",
			fields: fields{},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			arch := Architecture{
				names:                  tt.fields.names,
				defaultAWSInstanceType: tt.fields.defaultAWSInstanceType,
				defaultGCPInstanceType: tt.fields.defaultGCPInstanceType,
			}
			if got := arch.IsValid(); got != tt.want {
				t.Errorf("Architecture.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestArchitectureByName(t *testing.T) {
	tests := []struct {
		name string
		want Architecture
	}{
		{
			name: "X86",
			want: ArchX86,
		},
		{
			name: "  x86_64   ",
			want: ArchX86,
		},
		{
			name: "AmD64",
			want: ArchX86,
		},
		{
			name: "aarch64",
			want: ArchARM,
		},
		{
			name: "ARM",
			want: ArchARM,
		},
		{
			name: "invalid name",
			want: Architecture{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ArchitectureByName(tt.name); got != tt.want {
				t.Errorf("ArchitectureByName() = %v, want %v", got, tt.want)
			}
		})
	}
}
