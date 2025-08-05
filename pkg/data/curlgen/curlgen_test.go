package curlgen

import (
	"testing"
)

func TestGenerateString(t *testing.T) {

	tests := []struct {
		name    string
		args    *Options
		want    string
		wantErr bool
	}{
		{
			name: "Happy Path",
			args: &Options{
				CaPath:          "/some/config/path/",
				ProxyCaPath:     "/some/config/path/",
				Retry:           3,
				MaxTime:         "4",
				NoTls:           "false",
				Urls:            "http://example.com:80 https://example.org:443",
				TlsDisabledUrls: "http://example2.com:80 https://example2.org:443",
			},
			want:    "curl --capath /some/config/path/ --proxy-capath /some/config/path/ --retry 3 --retry-connrefused -t B -Z -s -I -m 4 -w \"%{stderr}@NV@%{json}\\n\" http://example.com:80 https://example.org:443 --proto =http,https,telnet --next --insecure --retry 3 --retry-connrefused -s -I -m 4 -w \"%{stderr}@NV@%{json}\\n\" http://example2.com:80 https://example2.org:443 --proto =https",
			wantErr: false,
		},
		{
			name: "NoTls true with tlsDisabledUrls",
			args: &Options{
				CaPath:          "/some/config/path/",
				ProxyCaPath:     "/some/config/path/",
				Retry:           3,
				MaxTime:         "4",
				NoTls:           "true",
				Urls:            "http://example.com:80 https://example.org:443",
				TlsDisabledUrls: "http://example2.com:80 https://example2.org:443",
			},
			want:    "curl --capath /some/config/path/ --proxy-capath /some/config/path/ --retry 3 --retry-connrefused -t B -Z -s -I -m 4 -w \"%{stderr}@NV@%{json}\\n\" --insecure http://example.com:80 https://example.org:443 http://example2.com:80 https://example2.org:443 --proto =http,https,telnet",
			wantErr: false,
		},
		{
			name: "NoTls False with No TlsDisabledURLs",
			args: &Options{
				CaPath:          "/some/config/path/",
				ProxyCaPath:     "/some/config/path/",
				Retry:           3,
				MaxTime:         "4",
				NoTls:           "false",
				Urls:            "http://example.com:80 https://example.org:443",
				TlsDisabledUrls: "",
			},
			want:    "curl --capath /some/config/path/ --proxy-capath /some/config/path/ --retry 3 --retry-connrefused -t B -Z -s -I -m 4 -w \"%{stderr}@NV@%{json}\\n\" http://example.com:80 https://example.org:443 --proto =http,https,telnet",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateString(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GenerateString() = %v, want %v", got, tt.want)
			}
		})
	}
}
