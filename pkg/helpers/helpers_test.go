package helpers

import (
	_ "embed"
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

func Test_fixLeadingZerosInJSON(t *testing.T) {
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fixLeadingZerosInJSON(tt.strContainingJSON); got != tt.want {
				t.Errorf("fixLeadingZerosInJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}
