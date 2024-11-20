package cloud

import (
	"testing"
)

func TestPlatform_Comparable(t *testing.T) {
	if AWSClassic != AWSHCP {
		return
	}
}

func TestPlatform_String(t *testing.T) {
	tests := []struct {
		name     string
		platform Platform
		want     string
	}{
		{
			name:     "aws",
			platform: AWSClassic,
			want:     "aws-classic",
		},
		{
			name:     "aws-classic",
			platform: AWSClassic,
			want:     "aws-classic",
		},
		{
			name:     "hosted-cluster",
			platform: AWSHCP,
			want:     "aws-hcp",
		},
		{
			name:     "aws-hcp",
			platform: AWSHCP,
			want:     "aws-hcp",
		},
		{
			name:     "gcp",
			platform: GCPClassic,
			want:     "gcp-classic",
		},
		{
			name:     "gcp-classic",
			platform: GCPClassic,
			want:     "gcp-classic",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			platform := tt.platform
			if got := platform.String(); got != tt.want {
				t.Errorf("Platform.String() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestPlatform_IsValid(t *testing.T) {
	type fields struct {
		names [3]string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "AWS happy path",
			fields: fields(AWSClassic),
			want:   true,
		},
		{
			name:   "AWSHCP happy path",
			fields: fields(AWSHCP),
			want:   true,
		},
		{
			name:   "GCP happy path",
			fields: fields(GCPClassic),
			want:   true,
		},
		{
			name: "fake platform",
			fields: fields{
				names: [3]string{"foo", "bar"},
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
			platform := Platform{
				names: tt.fields.names,
			}
			if got := platform.IsValid(); got != tt.want {
				t.Errorf("Platform.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestByName(t *testing.T) {
	tests := []struct {
		name string
		want Platform
	}{
		{
			name: "aws",
			want: AWSClassic,
		},
		{
			name: "  aws-classic   ",
			want: AWSClassic,
		},
		{
			name: "aws-hcp",
			want: AWSHCP,
		},
		{
			name: "aws-hosted-cp",
			want: AWSHCP,
		},
		{
			name: "hostedcluster",
			want: AWSHCP,
		},
		{
			name: "gcp",
			want: GCPClassic,
		},
		{
			name: "gcp-classic",
			want: GCPClassic,
		},
		{
			name: "invalid name",
			want: Platform{},
		},
		{
			name: "",
			want: Platform{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := ByName(tt.name); got != tt.want {
				if err != nil {
					t.Errorf("Error, %s", err)
				}
				t.Errorf("ArchitectureByName() = %v, want %v", got, tt.want)
			}
		})
	}
}
