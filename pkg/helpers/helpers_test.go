package helpers

import (
	_ "embed"
	"reflect"
	"testing"

	awsTools "github.com/aws/aws-sdk-go-v2/aws"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestIPPermissionsEquivalent(t *testing.T) {
	type args struct {
		a ec2Types.IpPermission
		b ec2Types.IpPermission
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "identical",
			args: args{
				a: ec2Types.IpPermission{
					FromPort:   awsTools.Int32(80),
					ToPort:     awsTools.Int32(80),
					IpProtocol: awsTools.String("tcp"),
					IpRanges: []ec2Types.IpRange{
						{
							CidrIp:      awsTools.String("0.0.0.0/0"),
							Description: awsTools.String("foo"),
						},
					},
				},
				b: ec2Types.IpPermission{
					FromPort:   awsTools.Int32(80),
					ToPort:     awsTools.Int32(80),
					IpProtocol: awsTools.String("tcp"),
					IpRanges: []ec2Types.IpRange{
						{
							CidrIp:      awsTools.String("0.0.0.0/0"),
							Description: awsTools.String("foo"),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "equivalent diff descriptions",
			args: args{
				a: ec2Types.IpPermission{
					FromPort:   awsTools.Int32(80),
					ToPort:     awsTools.Int32(80),
					IpProtocol: awsTools.String("tcp"),
					IpRanges: []ec2Types.IpRange{
						{
							CidrIp:      awsTools.String("0.0.0.0/0"),
							Description: awsTools.String("foo"),
						},
					},
				},
				b: ec2Types.IpPermission{
					FromPort:   awsTools.Int32(80),
					ToPort:     awsTools.Int32(80),
					IpProtocol: awsTools.String("tcp"),
					IpRanges: []ec2Types.IpRange{
						{
							CidrIp:      awsTools.String("0.0.0.0/0"),
							Description: awsTools.String("bar"),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "equivalent diff iprange ordering",
			args: args{
				a: ec2Types.IpPermission{
					FromPort:   awsTools.Int32(80),
					ToPort:     awsTools.Int32(80),
					IpProtocol: awsTools.String("tcp"),
					IpRanges: []ec2Types.IpRange{
						{
							CidrIp:      awsTools.String("1.1.1.1/23"),
							Description: awsTools.String("foo"),
						},
						{
							CidrIp:      awsTools.String("2.2.2.2/34"),
							Description: awsTools.String("foo"),
						},
					},
				},
				b: ec2Types.IpPermission{
					FromPort:   awsTools.Int32(80),
					ToPort:     awsTools.Int32(80),
					IpProtocol: awsTools.String("tcp"),
					IpRanges: []ec2Types.IpRange{
						{
							CidrIp:      awsTools.String("2.2.2.2/34"),
							Description: awsTools.String("foo"),
						},
						{
							CidrIp:      awsTools.String("1.1.1.1/23"),
							Description: awsTools.String("foo"),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "equivalent diff ipv6range ordering",
			args: args{
				a: ec2Types.IpPermission{
					FromPort:   awsTools.Int32(80),
					ToPort:     awsTools.Int32(80),
					IpProtocol: awsTools.String("tcp"),
					Ipv6Ranges: []ec2Types.Ipv6Range{
						{
							CidrIpv6:    awsTools.String("ff06::c5/128"),
							Description: awsTools.String("foo"),
						},
						{
							CidrIpv6:    awsTools.String("ff03::c1/128"),
							Description: awsTools.String("bar"),
						},
					},
				},
				b: ec2Types.IpPermission{
					FromPort:   awsTools.Int32(80),
					ToPort:     awsTools.Int32(80),
					IpProtocol: awsTools.String("tcp"),
					Ipv6Ranges: []ec2Types.Ipv6Range{
						{
							CidrIpv6:    awsTools.String("ff03::c1/128"),
							Description: awsTools.String("bar"),
						},
						{
							CidrIpv6:    awsTools.String("ff06::c5/128"),
							Description: awsTools.String("foo"),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "not equivalent port",
			args: args{
				a: ec2Types.IpPermission{
					FromPort:   awsTools.Int32(8080),
					ToPort:     awsTools.Int32(8080),
					IpProtocol: awsTools.String("tcp"),
					IpRanges: []ec2Types.IpRange{
						{
							CidrIp:      awsTools.String("0.0.0.0/0"),
							Description: awsTools.String("foo"),
						},
					},
				},
				b: ec2Types.IpPermission{
					FromPort:   awsTools.Int32(80),
					ToPort:     awsTools.Int32(80),
					IpProtocol: awsTools.String("tcp"),
					IpRanges: []ec2Types.IpRange{
						{
							CidrIp:      awsTools.String("0.0.0.0/0"),
							Description: awsTools.String("foo"),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "not equivalent cidr",
			args: args{
				a: ec2Types.IpPermission{
					FromPort:   awsTools.Int32(80),
					ToPort:     awsTools.Int32(80),
					IpProtocol: awsTools.String("tcp"),
					IpRanges: []ec2Types.IpRange{
						{
							CidrIp:      awsTools.String("0.0.1.3/32"),
							Description: awsTools.String("foo"),
						},
					},
				},
				b: ec2Types.IpPermission{
					FromPort:   awsTools.Int32(80),
					ToPort:     awsTools.Int32(80),
					IpProtocol: awsTools.String("tcp"),
					IpRanges: []ec2Types.IpRange{
						{
							CidrIp:      awsTools.String("0.0.0.0/0"),
							Description: awsTools.String("foo"),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "not equivalent range len",
			args: args{
				a: ec2Types.IpPermission{
					FromPort:   awsTools.Int32(80),
					ToPort:     awsTools.Int32(80),
					IpProtocol: awsTools.String("tcp"),
					IpRanges: []ec2Types.IpRange{
						{
							CidrIp:      awsTools.String("0.0.0.0/32"),
							Description: awsTools.String("foo"),
						},
						{
							CidrIp:      awsTools.String("0.0.1.3/32"),
							Description: awsTools.String("bar"),
						},
					},
				},
				b: ec2Types.IpPermission{
					FromPort:   awsTools.Int32(80),
					ToPort:     awsTools.Int32(80),
					IpProtocol: awsTools.String("tcp"),
					IpRanges: []ec2Types.IpRange{
						{
							CidrIp:      awsTools.String("0.0.0.0/32"),
							Description: awsTools.String("foo"),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "not equivalent v6",
			args: args{
				a: ec2Types.IpPermission{
					FromPort:   awsTools.Int32(80),
					ToPort:     awsTools.Int32(80),
					IpProtocol: awsTools.String("tcp"),
					Ipv6Ranges: []ec2Types.Ipv6Range{
						{
							CidrIpv6:    awsTools.String("ff06::c3/128"),
							Description: awsTools.String("foo"),
						},
					},
				},
				b: ec2Types.IpPermission{
					FromPort:   awsTools.Int32(80),
					ToPort:     awsTools.Int32(80),
					IpProtocol: awsTools.String("tcp"),
					Ipv6Ranges: []ec2Types.Ipv6Range{
						{
							CidrIpv6:    awsTools.String("ff06::c5/128"),
							Description: awsTools.String("foo"),
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IPPermissionsEquivalent(tt.args.a, tt.args.b); got != tt.want {
				t.Errorf("IPPermissionsEquivalent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetPlatformType(t *testing.T) {
	tests := []struct {
		name         string
		platformType string
		want         string
		wantErr      bool
	}{
		{
			name:         "Platform type aws",
			platformType: PlatformAWS,
			want:         "aws",
			wantErr:      false,
		},
		{
			name:         "Platform type gcp",
			platformType: PlatformGCP,
			want:         "gcp",
			wantErr:      false,
		},
		{
			name:         "Platform type hostedcluster",
			platformType: PlatformHostedCluster,
			want:         "hostedcluster",
			wantErr:      false,
		},
		{
			name:         "Platform type aws-classic",
			platformType: PlatformAWSClassic,
			want:         "aws",
			wantErr:      false,
		},
		{
			name:         "Platform type gcp-classic",
			platformType: PlatformGCPClassic,
			want:         "gcp",
			wantErr:      false,
		},
		{
			name:         "Platform type aws-hcp",
			platformType: PlatformAWSHCP,
			want:         "hostedcluster",
			wantErr:      false,
		},
		{
			name:         "Invalid platform type",
			platformType: "foobar",
			want:         "",
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetPlatformType(tt.platformType)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetPlatformType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetPlatformType() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_FixLeadingZerosInJSON(t *testing.T) {
	tests := []struct {
		name              string
		strContainingJSON string
		want              string
	}{
		{
			name:              "curl SSL error output",
			strContainingJSON: `@NV@{"content_type":null,"errormsg":"SSL certificate problem: unable to get local issuer certificate","exitcode":60,"filename_effective":null,"ftp_entry_path":null,"http_code":000,"http_connect":000,"http_version":"0","local_ip":"172.31.2.213","local_port":51232,"method":"HEAD","num_connects":1,"num_headers":0,"num_redirects":0,"proxy_ssl_verify_result":0,"redirect_url":null,"referer":null,"remote_ip":"52.55.72.119","remote_port":443,"response_code":000,"scheme":"HTTPS","size_download":0,"size_header":0,"size_request":0,"size_upload":0,"speed_download":0,"speed_upload":0,"ssl_verify_result":20,"time_appconnect":0.000000,"time_connect":0.053023,"time_namelookup":0.009450,"time_pretransfer":0.000000,"time_redirect":0.000000,"time_starttransfer":0.000000,"time_total":0.376118,"url":"https://infogw.api.openshift.com:443","url_effective":"https://infogw.api.openshift.com:443/","urlnum":13,"curl_version":"libcurl/7.76.1 OpenSSL/3.0.7 zlib/1.2.11 brotli/1.0.9 libidn2/2.3.0 libpsl/0.21.1 [2024-04-01T19:50:55.991747](+libidn2/2.3.0) libssh/0.10.4/openssl/zlib nghttp2/1.43.0"}`,
			want:              `@NV@{"content_type":null,"errormsg":"SSL certificate problem: unable to get local issuer certificate","exitcode":60,"filename_effective":null,"ftp_entry_path":null,"http_code":0,"http_connect":0,"http_version":"0","local_ip":"172.31.2.213","local_port":51232,"method":"HEAD","num_connects":1,"num_headers":0,"num_redirects":0,"proxy_ssl_verify_result":0,"redirect_url":null,"referer":null,"remote_ip":"52.55.72.119","remote_port":443,"response_code":0,"scheme":"HTTPS","size_download":0,"size_header":0,"size_request":0,"size_upload":0,"speed_download":0,"speed_upload":0,"ssl_verify_result":20,"time_appconnect":0.000000,"time_connect":0.053023,"time_namelookup":0.009450,"time_pretransfer":0.000000,"time_redirect":0.000000,"time_starttransfer":0.000000,"time_total":0.376118,"url":"https://infogw.api.openshift.com:443","url_effective":"https://infogw.api.openshift.com:443/","urlnum":13,"curl_version":"libcurl/7.76.1 OpenSSL/3.0.7 zlib/1.2.11 brotli/1.0.9 libidn2/2.3.0 libpsl/0.21.1 [2024-04-01T19:50:55.991747](+libidn2/2.3.0) libssh/0.10.4/openssl/zlib nghttp2/1.43.0"}`,
		},
		{
			name:              "curl non-http(s) success output",
			strContainingJSON: `@NV@{"content_type":null,"errormsg":"Syntax error in telnet option: B","exitcode":49,"filename_effective":null,"ftp_entry_path":null,"http_code":000,"http_connect":000,"http_version":"0","local_ip":"10.0.2.100","local_port":50254,"method":"HEAD","num_connects":1,"num_headers":0,"num_redirects":0,"proxy_ssl_verify_result":0,"redirect_url":null,"referer":null,"remote_ip":"18.232.251.220","remote_port":9997,"response_code":000,"scheme":"TELNET","size_download":0,"size_header":0,"size_request":0,"size_upload":0,"speed_download":0,"speed_upload":0,"ssl_verify_result":0,"time_appconnect":0.000000,"time_connect":0.057533,"time_namelookup":0.031295,"time_pretransfer":0.000000,"time_redirect":0.000000,"time_starttransfer":0.000000,"time_total":0.057613,"url":"telnet://inputs1.osdsecuritylogs.splunkcloud.com:9997","url_effective":"telnet://inputs1.osdsecuritylogs.splunkcloud.com:9997/","urlnum":0,"curl_version":"libcurl/7.76.1-DEV OpenSSL/1.1.1k zlib/1.2.11 brotli/1.0.9 libssh2/1.9.0 nghttp2/1.41.0"}`,
			want:              `@NV@{"content_type":null,"errormsg":"Syntax error in telnet option: B","exitcode":49,"filename_effective":null,"ftp_entry_path":null,"http_code":0,"http_connect":0,"http_version":"0","local_ip":"10.0.2.100","local_port":50254,"method":"HEAD","num_connects":1,"num_headers":0,"num_redirects":0,"proxy_ssl_verify_result":0,"redirect_url":null,"referer":null,"remote_ip":"18.232.251.220","remote_port":9997,"response_code":0,"scheme":"TELNET","size_download":0,"size_header":0,"size_request":0,"size_upload":0,"speed_download":0,"speed_upload":0,"ssl_verify_result":0,"time_appconnect":0.000000,"time_connect":0.057533,"time_namelookup":0.031295,"time_pretransfer":0.000000,"time_redirect":0.000000,"time_starttransfer":0.000000,"time_total":0.057613,"url":"telnet://inputs1.osdsecuritylogs.splunkcloud.com:9997","url_effective":"telnet://inputs1.osdsecuritylogs.splunkcloud.com:9997/","urlnum":0,"curl_version":"libcurl/7.76.1-DEV OpenSSL/1.1.1k zlib/1.2.11 brotli/1.0.9 libssh2/1.9.0 nghttp2/1.41.0"}`,
		},
		{
			name:              "curl http error output",
			strContainingJSON: `@NV@{"content_type":null,"errormsg":"Failed to connect to localhost port 80: Connection refused","exitcode":7,"filename_effective":null,"ftp_entry_path":null,"http_code":000,"http_connect":000,"http_version":"0","local_ip":"","local_port":0,"method":"HEAD","num_connects":0,"num_headers":0,"num_redirects":0,"proxy_ssl_verify_result":0,"redirect_url":null,"referer":null,"remote_ip":"","remote_port":0,"response_code":000,"scheme":null,"size_download":0,"size_header":0,"size_request":0,"size_upload":0,"speed_download":0,"speed_upload":0,"ssl_verify_result":0,"time_appconnect":0.000000,"time_connect":0.000000,"time_namelookup":0.000205,"time_pretransfer":0.000000,"time_redirect":0.000000,"time_starttransfer":0.000000,"time_total":0.000338,"url":"http://localhost:80","url_effective":"http://localhost:80/","urlnum":0,"curl_version":"libcurl/7.76.1-DEV OpenSSL/1.1.1k zlib/1.2.11 brotli/1.0.9 libssh2/1.9.0 nghttp2/1.41.0"}`,
			want:              `@NV@{"content_type":null,"errormsg":"Failed to connect to localhost port 80: Connection refused","exitcode":7,"filename_effective":null,"ftp_entry_path":null,"http_code":0,"http_connect":0,"http_version":"0","local_ip":"","local_port":0,"method":"HEAD","num_connects":0,"num_headers":0,"num_redirects":0,"proxy_ssl_verify_result":0,"redirect_url":null,"referer":null,"remote_ip":"","remote_port":0,"response_code":0,"scheme":null,"size_download":0,"size_header":0,"size_request":0,"size_upload":0,"speed_download":0,"speed_upload":0,"ssl_verify_result":0,"time_appconnect":0.000000,"time_connect":0.000000,"time_namelookup":0.000205,"time_pretransfer":0.000000,"time_redirect":0.000000,"time_starttransfer":0.000000,"time_total":0.000338,"url":"http://localhost:80","url_effective":"http://localhost:80/","urlnum":0,"curl_version":"libcurl/7.76.1-DEV OpenSSL/1.1.1k zlib/1.2.11 brotli/1.0.9 libssh2/1.9.0 nghttp2/1.41.0"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FixLeadingZerosInJSON(tt.strContainingJSON); got != tt.want {
				t.Errorf("FixLeadingZerosInJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractRequiredVariablesDirective(t *testing.T) {
	tests := []struct {
		name      string
		yamlStr   string
		wantStr   string
		wantSlice []string
	}{
		{
			name:      "basic single line",
			yamlStr:   "# network-verifier-required-variables=VAR_X,VAR_Y,VAR_Z",
			wantStr:   "",
			wantSlice: []string{"VAR_X", "VAR_Y", "VAR_Z"},
		},
		{
			name:      "spacey single line",
			yamlStr:   " #	network-verifier-required-variables  = VAR_X,VAR_Y,VAR_Z   ",
			wantStr:   "",
			wantSlice: []string{"VAR_X", "VAR_Y", "VAR_Z"},
		},
		{
			name:      "one variable",
			yamlStr:   "# network-verifier-required-variables=VAR_X",
			wantStr:   "",
			wantSlice: []string{"VAR_X"},
		},
		{
			name:      "garbage",
			yamlStr:   "qwetry#eruvh2984jngf",
			wantStr:   "qwetry#eruvh2984jngf",
			wantSlice: []string{},
		},
		{
			name:      "multi-line",
			yamlStr:   "#cloud-config\n#network-verifier-required-variables=A,B\n${CERT}\nruncmd:\n\t- foo",
			wantStr:   "#cloud-config\n\n${CERT}\nruncmd:\n\t- foo",
			wantSlice: []string{"A", "B"},
		},
		{
			name:      "multiple appearances",
			yamlStr:   "#cloud-config\n#network-verifier-required-variables=A\n#network-verifier-required-variables=B",
			wantStr:   "#cloud-config\n\n",
			wantSlice: []string{"A"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStr, gotSlice := ExtractRequiredVariablesDirective(tt.yamlStr)
			if gotStr != tt.wantStr {
				t.Errorf("ExtractRequiredVariablesDirective() got = %v, want %v", gotStr, tt.wantStr)
			}
			if !reflect.DeepEqual(gotSlice, tt.wantSlice) {
				t.Errorf("ExtractRequiredVariablesDirective() got1 = %v, want %v", gotSlice, tt.wantSlice)
			}
		})
	}
}

func TestValidateProvidedVariables(t *testing.T) {
	type args struct {
		providedVarMap   map[string]string
		presetVarMap     map[string]string
		requiredVarSlice []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "happy path",
			args: args{
				providedVarMap:   map[string]string{"A": "a", "B": "b"},
				presetVarMap:     map[string]string{"C": "c", "D": "d"},
				requiredVarSlice: []string{"B", "C"},
			},
			wantErr: false,
		},
		{
			name: "required var not provided",
			args: args{
				providedVarMap:   map[string]string{"A": "a", "X": "x"},
				presetVarMap:     map[string]string{"C": "c", "D": "d"},
				requiredVarSlice: []string{"B", "C"},
			},
			wantErr: true,
		},
		{
			name: "required var not preset",
			args: args{
				providedVarMap:   map[string]string{"A": "a", "B": "b"},
				presetVarMap:     map[string]string{"X": "x", "D": "d"},
				requiredVarSlice: []string{"B", "C"},
			},
			wantErr: true,
		},
		{
			name: "provided overlaps with preset",
			args: args{
				providedVarMap:   map[string]string{"A": "a", "B": "b", "C": "x"},
				presetVarMap:     map[string]string{"C": "c", "D": "d"},
				requiredVarSlice: []string{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateProvidedVariables(tt.args.providedVarMap, tt.args.presetVarMap, tt.args.requiredVarSlice); (err != nil) != tt.wantErr {
				t.Errorf("validateProvidedVariables() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCutBetween(t *testing.T) {
	type args struct {
		s             string
		startingToken string
		endingToken   string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "happy path",
			args: args{
				s:             "foo123bar",
				startingToken: "foo",
				endingToken:   "bar",
			},
			want: "123",
		},
		{
			name: "tokens swapped",
			args: args{
				s:             "bar123foo",
				startingToken: "foo",
				endingToken:   "bar",
			},
			want: "",
		},
		{
			name: "tokens with regexp characters",
			args: args{
				s:             `[].*123\S\D`,
				startingToken: `[].*`,
				endingToken:   `\S\D`,
			},
			want: "123",
		},
		{
			name: "missing startingToken",
			args: args{
				s:             "123bar",
				startingToken: "foo",
				endingToken:   "bar",
			},
			want: "",
		},
		{
			name: "missing endingToken",
			args: args{
				s:             "foo123",
				startingToken: "foo",
				endingToken:   "bar",
			},
			want: "",
		},
		{
			name: "missing both tokens",
			args: args{
				s:             "123",
				startingToken: "foo",
				endingToken:   "bar",
			},
			want: "",
		},
		{
			name: "newlines between tokens",
			args: args{
				s:             "foo\n\n1\t2\n3bar",
				startingToken: "foo",
				endingToken:   "bar",
			},
			want: "\n\n1\t2\n3",
		},
		{
			name: "startingToken between tokens",
			args: args{
				s:             "foo12foo34bar",
				startingToken: "foo",
				endingToken:   "bar",
			},
			want: "12foo34",
		},
		{
			name: "endingToken between tokens",
			args: args{
				s:             "foo12bar34bar",
				startingToken: "foo",
				endingToken:   "bar",
			},
			want: "12bar34",
		},
		{
			name: "tokens between tokens",
			args: args{
				s:             "foo12barfoobarfoobar34bar",
				startingToken: "foo",
				endingToken:   "bar",
			},
			want: "12barfoobarfoobar34",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CutBetween(tt.args.s, tt.args.startingToken, tt.args.endingToken); got != tt.want {
				t.Errorf("CutBetween() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRemoveTimestamps(t *testing.T) {
	tests := []struct {
		name                    string
		strContainingTimestamps string
		want                    string
	}{
		{
			name:                    "happy path",
			strContainingTimestamps: "foo[2024-04-01T17:53:15.971047]bar",
			want:                    "foobar",
		},
		{
			name:                    "multiple timestamps",
			strContainingTimestamps: "foo[2024-04-01T17:53:15.971047]bar[2023-10-11T08:12:10.909847]too",
			want:                    "foobartoo",
		},
		{
			name:                    "no timestamps",
			strContainingTimestamps: "foo[bTr]",
			want:                    "foo[bTr]",
		},
		{
			name:                    "raw CurlJSONProbe output",
			strContainingTimestamps: `@NV@{"content_type":null,"errormsg":"SSL certificate problem: unable to get local issuer certificate","exitcode":60,"filename_effective":null,"ftp_entry_path":null,"http_code":000,"http_connect":000,"http_version":"0","local_ip":"172.31.2.213","local_port":51232,"method":"HEAD","num_connects":1,"num_headers":0,"num_redirects":0,"proxy_ssl_verify_result":0,"redirect_url":null,"referer":null,"remote_ip":"52.55.72.119","remote_port":443,"response_code":000,"scheme":"HTTPS","size_download":0,"size_header":0,"size_request":0,"size_upload":0,"speed_download":0,"speed_upload":0,"ssl_verify_result":20,"time_appconnect":0.000000,"time_connect":0.053023,"time_namelookup":0.009450,"time_pretransfer":0.000000,"time_redirect":0.000000,"time_starttransfer":0.000000,"time_total":0.376118,"url":"https://infogw.api.openshift.com:443","url_effective":"https://infogw.api.openshift.com:443/","urlnum":13,"curl_version":"libcurl/7.76.1 OpenSSL/3.0.7 zlib/1.2.11 brotli/1.0.9 libidn2/2.3.0 libpsl/0.21.1 [2024-04-01T19:50:55.991747](+libidn2/2.3.0) libssh/0.10.4/openssl/zlib nghttp2/1.43.0"}`,
			want:                    `@NV@{"content_type":null,"errormsg":"SSL certificate problem: unable to get local issuer certificate","exitcode":60,"filename_effective":null,"ftp_entry_path":null,"http_code":000,"http_connect":000,"http_version":"0","local_ip":"172.31.2.213","local_port":51232,"method":"HEAD","num_connects":1,"num_headers":0,"num_redirects":0,"proxy_ssl_verify_result":0,"redirect_url":null,"referer":null,"remote_ip":"52.55.72.119","remote_port":443,"response_code":000,"scheme":"HTTPS","size_download":0,"size_header":0,"size_request":0,"size_upload":0,"speed_download":0,"speed_upload":0,"ssl_verify_result":20,"time_appconnect":0.000000,"time_connect":0.053023,"time_namelookup":0.009450,"time_pretransfer":0.000000,"time_redirect":0.000000,"time_starttransfer":0.000000,"time_total":0.376118,"url":"https://infogw.api.openshift.com:443","url_effective":"https://infogw.api.openshift.com:443/","urlnum":13,"curl_version":"libcurl/7.76.1 OpenSSL/3.0.7 zlib/1.2.11 brotli/1.0.9 libidn2/2.3.0 libpsl/0.21.1 (+libidn2/2.3.0) libssh/0.10.4/openssl/zlib nghttp2/1.43.0"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RemoveTimestamps(tt.strContainingTimestamps); got != tt.want {
				t.Errorf("RemoveTimestamps() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDurationToBareSeconds(t *testing.T) {
	tests := []struct {
		name                string
		possibleDurationStr string
		want                float64
	}{
		{
			name:                "simple duration",
			possibleDurationStr: "3s",
			want:                3,
		},
		{
			name:                "compound duration",
			possibleDurationStr: "1m3s",
			want:                63,
		},
		{
			name:                "bare integer",
			possibleDurationStr: "7",
			want:                7,
		},
		{
			name:                "bare float",
			possibleDurationStr: "1.11",
			want:                1.11,
		},
		{
			name:                "noisy number",
			possibleDurationStr: "foo1.23bar",
			want:                1.23,
		},
		{
			name:                "no numbers",
			possibleDurationStr: "foobar",
			want:                0,
		},
		{
			name:                "empty string",
			possibleDurationStr: "",
			want:                0,
		},
		{
			name:                "negative unit",
			possibleDurationStr: "-1m",
			want:                -60.0,
		},
		{
			name:                "negative float",
			possibleDurationStr: "-3.33",
			want:                -3.33,
		},
		{
			name:                "nan",
			possibleDurationStr: "NaN",
			want:                0,
		},
		{
			name:                "infinity",
			possibleDurationStr: "Inf",
			want:                0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DurationToBareSeconds(tt.possibleDurationStr); got != tt.want {
				t.Errorf("DurationToBareSeconds() = %v, want %v", got, tt.want)
			}
		})
	}
}
