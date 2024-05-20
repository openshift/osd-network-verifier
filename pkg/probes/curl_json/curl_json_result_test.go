package curl_json

import (
	"reflect"
	"slices"
	"testing"
)

func TestCurlJSONProbeResult_isSuccessfulConnection(t *testing.T) {
	tests := []struct {
		name string
		res  CurlJSONProbeResult
		want bool
	}{
		{
			name: "successful http connection",
			res: CurlJSONProbeResult{
				ContentType:       "text/html; charset=utf-8",
				HTTPCode:          200,
				HTTPVersion:       "1.1",
				LocalPort:         58234,
				Method:            "HEAD",
				NumHeaders:        10,
				RemoteIP:          "34.223.124.45",
				RemotePort:        80,
				ResponseCode:      200,
				Scheme:            "HTTP",
				SizeHeader:        300,
				SizeRequest:       81,
				TimeAppConnect:    0.000111,
				TimeConnect:       0.000111,
				TimeNameLookup:    0.000111,
				TimePreTransfer:   0.000146,
				TimeStartTransfer: 0.997047,
				TimeTotal:         0.997072,
				URL:               "http://neverssl.com:80",
				URLEffective:      "http://neverssl.com:80/",
				URLNum:            0,
				CurlVersion:       "libcurl/7.76.1 OpenSSL/3.0.7 zlib/1.2.11 brotli/1.0.9 libidn2/2.3.0 libpsl/0.21.1 (+libidn2/2.3.0) libssh/0.10.4/openssl/zlib nghttp2/1.43.0",
			},
			want: true,
		},
		{
			name: "failed http connection",
			res: CurlJSONProbeResult{
				ErrorMsg:       "Failed to connect to localhost port 80: Connection refused",
				ExitCode:       7,
				HTTPVersion:    "0",
				Method:         "HEAD",
				Scheme:         "",
				TimeNameLookup: 0.000205,
				TimeTotal:      0.000338,
				URL:            "http://localhost:80",
				URLEffective:   "http://localhost:80/",
				CurlVersion:    "libcurl/7.76.1-DEV OpenSSL/1.1.1k zlib/1.2.11 brotli/1.0.9 libssh2/1.9.0 nghttp2/1.41.0",
			},
			want: false,
		},
		{
			name: "successful https connection",
			res: CurlJSONProbeResult{
				ContentType:       "text/html; charset=utf-8",
				HTTPCode:          200,
				HTTPVersion:       "2",
				LocalPort:         -1,
				Method:            "HEAD",
				NumHeaders:        10,
				RemoteIP:          "23.20.243.242",
				RemotePort:        443,
				ResponseCode:      200,
				Scheme:            "HTTPS",
				SizeHeader:        514,
				SizeRequest:       70,
				TimeAppConnect:    0.000111,
				TimeConnect:       0.000111,
				TimeNameLookup:    0.000111,
				TimePreTransfer:   0.000146,
				TimeStartTransfer: 0.997047,
				TimeTotal:         0.997072,
				URL:               "https://quay.io:443",
				URLEffective:      "https://quay.io:443/",
				URLNum:            1,
				CurlVersion:       "libcurl/7.76.1 OpenSSL/3.0.7 zlib/1.2.11 brotli/1.0.9 libidn2/2.3.0 libpsl/0.21.1 (+libidn2/2.3.0) libssh/0.10.4/openssl/zlib nghttp2/1.43.0",
			},
			want: true,
		},
		{
			name: "failed https connection",
			res: CurlJSONProbeResult{
				ErrorMsg:        "SSL certificate problem: unable to get local issuer certificate",
				ExitCode:        60,
				HTTPVersion:     "0",
				LocalIP:         "172.31.2.213",
				LocalPort:       51232,
				Method:          "HEAD",
				NumConnects:     1,
				RemoteIP:        "52.55.72.119",
				RemotePort:      443,
				Scheme:          "HTTPS",
				SSLVerifyResult: 20,
				TimeConnect:     0.053023,
				TimeNameLookup:  0.00945,
				TimeTotal:       0.376118,
				URL:             "https://infogw.api.openshift.com:443",
				URLEffective:    "https://infogw.api.openshift.com:443/",
				URLNum:          13,
				CurlVersion:     "libcurl/7.76.1 OpenSSL/3.0.7 zlib/1.2.11 brotli/1.0.9 libidn2/2.3.0 libpsl/0.21.1 [2024-04-01T19:50:55.991747](+libidn2/2.3.0) libssh/0.10.4/openssl/zlib nghttp2/1.43.0",
			},
			want: false,
		},
		{
			name: "successful non-http(s) connection",
			res: CurlJSONProbeResult{
				ErrorMsg:       "Syntax error in telnet option: B",
				ExitCode:       49,
				HTTPVersion:    "0",
				LocalIP:        "10.0.2.100",
				LocalPort:      50254,
				Method:         "HEAD",
				NumConnects:    1,
				RemoteIP:       "18.232.251.220",
				RemotePort:     9997,
				Scheme:         "TELNET",
				TimeConnect:    0.057533,
				TimeNameLookup: 0.031295,
				TimeTotal:      0.057613,
				URL:            "telnet://inputs1.osdsecuritylogs.splunkcloud.com:9997",
				URLEffective:   "telnet://inputs1.osdsecuritylogs.splunkcloud.com:9997/",
				CurlVersion:    "libcurl/7.76.1-DEV OpenSSL/1.1.1k zlib/1.2.11 brotli/1.0.9 libssh2/1.9.0 nghttp2/1.41.0",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.res.IsSuccessfulConnection(); got != tt.want {
				t.Errorf("CurlJSONProbeResult.isSuccessfulConnection() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_deserializeCurlJSONProbeResult(t *testing.T) {
	tests := []struct {
		name             string
		prefixedCurlJSON string
		want             *CurlJSONProbeResult
		wantErr          bool
	}{
		{
			name:             "good curl error output",
			prefixedCurlJSON: `@NV@{"content_type":null,"errormsg":"SSL certificate problem: unable to get local issuer certificate","exitcode":60,"filename_effective":null,"ftp_entry_path":null,"http_code":0,"http_connect":0,"http_version":"0","local_ip":"172.31.2.213","local_port":51232,"method":"HEAD","num_connects":1,"num_headers":0,"num_redirects":0,"proxy_ssl_verify_result":0,"redirect_url":null,"referer":null,"remote_ip":"52.55.72.119","remote_port":443,"response_code":0,"scheme":"HTTPS","size_download":0,"size_header":0,"size_request":0,"size_upload":0,"speed_download":0,"speed_upload":0,"ssl_verify_result":20,"time_appconnect":0.000000,"time_connect":0.053023,"time_namelookup":0.009450,"time_pretransfer":0.000000,"time_redirect":0.000000,"time_starttransfer":0.000000,"time_total":0.376118,"url":"https://infogw.api.openshift.com:443","url_effective":"https://infogw.api.openshift.com:443/","urlnum":13,"curl_version":"libcurl/7.76.1 OpenSSL/3.0.7 zlib/1.2.11 brotli/1.0.9 libidn2/2.3.0 libpsl/0.21.1 [2024-04-01T19:50:55.991747](+libidn2/2.3.0) libssh/0.10.4/openssl/zlib nghttp2/1.43.0"}`,
			want: &CurlJSONProbeResult{
				ErrorMsg:        "SSL certificate problem: unable to get local issuer certificate",
				ExitCode:        60,
				HTTPVersion:     "0",
				LocalIP:         "172.31.2.213",
				LocalPort:       51232,
				Method:          "HEAD",
				NumConnects:     1,
				RemoteIP:        "52.55.72.119",
				RemotePort:      443,
				Scheme:          "HTTPS",
				SSLVerifyResult: 20,
				TimeConnect:     0.053023,
				TimeNameLookup:  0.00945,
				TimeTotal:       0.376118,
				URL:             "https://infogw.api.openshift.com:443",
				URLEffective:    "https://infogw.api.openshift.com:443/",
				URLNum:          13,
				CurlVersion:     "libcurl/7.76.1 OpenSSL/3.0.7 zlib/1.2.11 brotli/1.0.9 libidn2/2.3.0 libpsl/0.21.1 [2024-04-01T19:50:55.991747](+libidn2/2.3.0) libssh/0.10.4/openssl/zlib nghttp2/1.43.0",
			},
			wantErr: false,
		},
		{
			name:             "good curl success output",
			prefixedCurlJSON: `@NV@{"content_type":"text/html; charset=utf-8","errormsg":null,"exitcode":0,"filename_effective":null,"ftp_entry_path":null,"http_code":200,"http_connect":0,"http_version":"2","local_ip":"","local_port":-1,"method":"HEAD","num_connects":0,"num_headers":10,"num_redirects":0,"proxy_ssl_verify_result":0,"redirect_url":null,"referer":null,"remote_ip":"23.20.243.242","remote_port":443,"response_code":200,"scheme":"HTTPS","size_download":0,"size_header":514,"size_request":70,"size_upload":0,"speed_download":0,"speed_upload":0,"ssl_verify_result":0,"time_appconnect":0.000111,"time_connect":0.000111,"time_namelookup":0.000111,"time_pretransfer":0.000146,"time_redirect":0.000000,"time_starttransfer":0.997047,"time_total":0.997072,"url":"https://quay.io:443","url_effective":"https://quay.io:443/","urlnum":1,"curl_version":"libcurl/7.76.1 OpenSSL/3.0.7 zlib/1.2.11 brotli/1.0.9 libidn2/2.3.0 libpsl/0.21.1 (+libidn2/2.3.0) libssh/0.10.4/openssl/zlib nghttp2/1.43.0"}`,
			want: &CurlJSONProbeResult{
				ContentType:       "text/html; charset=utf-8",
				HTTPCode:          200,
				HTTPVersion:       "2",
				LocalPort:         -1,
				Method:            "HEAD",
				NumHeaders:        10,
				RemoteIP:          "23.20.243.242",
				RemotePort:        443,
				ResponseCode:      200,
				Scheme:            "HTTPS",
				SizeHeader:        514,
				SizeRequest:       70,
				TimeAppConnect:    0.000111,
				TimeConnect:       0.000111,
				TimeNameLookup:    0.000111,
				TimePreTransfer:   0.000146,
				TimeStartTransfer: 0.997047,
				TimeTotal:         0.997072,
				URL:               "https://quay.io:443",
				URLEffective:      "https://quay.io:443/",
				URLNum:            1,
				CurlVersion:       "libcurl/7.76.1 OpenSSL/3.0.7 zlib/1.2.11 brotli/1.0.9 libidn2/2.3.0 libpsl/0.21.1 (+libidn2/2.3.0) libssh/0.10.4/openssl/zlib nghttp2/1.43.0",
			},
			wantErr: false,
		},
		{
			name:             "good curl non-http(s) success output",
			prefixedCurlJSON: `@NV@{"content_type":null,"errormsg":"Syntax error in telnet option: B","exitcode":49,"filename_effective":null,"ftp_entry_path":null,"http_code":0,"http_connect":0,"http_version":"0","local_ip":"10.0.2.100","local_port":50254,"method":"HEAD","num_connects":1,"num_headers":0,"num_redirects":0,"proxy_ssl_verify_result":0,"redirect_url":null,"referer":null,"remote_ip":"18.232.251.220","remote_port":9997,"response_code":0,"scheme":"TELNET","size_download":0,"size_header":0,"size_request":0,"size_upload":0,"speed_download":0,"speed_upload":0,"ssl_verify_result":0,"time_appconnect":0.000000,"time_connect":0.057533,"time_namelookup":0.031295,"time_pretransfer":0.000000,"time_redirect":0.000000,"time_starttransfer":0.000000,"time_total":0.057613,"url":"telnet://inputs1.osdsecuritylogs.splunkcloud.com:9997","url_effective":"telnet://inputs1.osdsecuritylogs.splunkcloud.com:9997/","urlnum":0,"curl_version":"libcurl/7.76.1-DEV OpenSSL/1.1.1k zlib/1.2.11 brotli/1.0.9 libssh2/1.9.0 nghttp2/1.41.0"}`,
			want: &CurlJSONProbeResult{
				ErrorMsg:       "Syntax error in telnet option: B",
				ExitCode:       49,
				HTTPVersion:    "0",
				LocalIP:        "10.0.2.100",
				LocalPort:      50254,
				Method:         "HEAD",
				NumConnects:    1,
				RemoteIP:       "18.232.251.220",
				RemotePort:     9997,
				Scheme:         "TELNET",
				TimeConnect:    0.057533,
				TimeNameLookup: 0.031295,
				TimeTotal:      0.057613,
				URL:            "telnet://inputs1.osdsecuritylogs.splunkcloud.com:9997",
				URLEffective:   "telnet://inputs1.osdsecuritylogs.splunkcloud.com:9997/",
				CurlVersion:    "libcurl/7.76.1-DEV OpenSSL/1.1.1k zlib/1.2.11 brotli/1.0.9 libssh2/1.9.0 nghttp2/1.41.0",
			},
			wantErr: false,
		},
		{
			name:             "good curl http error output",
			prefixedCurlJSON: `@NV@{"content_type":null,"errormsg":"Failed to connect to localhost port 80: Connection refused","exitcode":7,"filename_effective":null,"ftp_entry_path":null,"http_code":0,"http_connect":0,"http_version":"0","local_ip":"","local_port":0,"method":"HEAD","num_connects":0,"num_headers":0,"num_redirects":0,"proxy_ssl_verify_result":0,"redirect_url":null,"referer":null,"remote_ip":"","remote_port":0,"response_code":0,"scheme":null,"size_download":0,"size_header":0,"size_request":0,"size_upload":0,"speed_download":0,"speed_upload":0,"ssl_verify_result":0,"time_appconnect":0.000000,"time_connect":0.000000,"time_namelookup":0.000205,"time_pretransfer":0.000000,"time_redirect":0.000000,"time_starttransfer":0.000000,"time_total":0.000338,"url":"http://localhost:80","url_effective":"http://localhost:80/","urlnum":0,"curl_version":"libcurl/7.76.1-DEV OpenSSL/1.1.1k zlib/1.2.11 brotli/1.0.9 libssh2/1.9.0 nghttp2/1.41.0"}`,
			want: &CurlJSONProbeResult{
				ErrorMsg:       "Failed to connect to localhost port 80: Connection refused",
				ExitCode:       7,
				HTTPVersion:    "0",
				Method:         "HEAD",
				Scheme:         "",
				TimeNameLookup: 0.000205,
				TimeTotal:      0.000338,
				URL:            "http://localhost:80",
				URLEffective:   "http://localhost:80/",
				CurlVersion:    "libcurl/7.76.1-DEV OpenSSL/1.1.1k zlib/1.2.11 brotli/1.0.9 libssh2/1.9.0 nghttp2/1.41.0",
			},
			wantErr: false,
		},
		{
			name:             "good curl error output missing prefix",
			prefixedCurlJSON: `{"content_type":null,"errormsg":"SSL certificate problem: unable to get local issuer certificate","exitcode":60,"filename_effective":null,"ftp_entry_path":null,"http_code":0,"http_connect":0,"http_version":"0","local_ip":"172.31.2.213","local_port":51232,"method":"HEAD","num_connects":1,"num_headers":0,"num_redirects":0,"proxy_ssl_verify_result":0,"redirect_url":null,"referer":null,"remote_ip":"52.55.72.119","remote_port":443,"response_code":0,"scheme":"HTTPS","size_download":0,"size_header":0,"size_request":0,"size_upload":0,"speed_download":0,"speed_upload":0,"ssl_verify_result":20,"time_appconnect":0.000000,"time_connect":0.053023,"time_namelookup":0.009450,"time_pretransfer":0.000000,"time_redirect":0.000000,"time_starttransfer":0.000000,"time_total":0.376118,"url":"https://infogw.api.openshift.com:443","url_effective":"https://infogw.api.openshift.com:443/","urlnum":13,"curl_version":"libcurl/7.76.1 OpenSSL/3.0.7 zlib/1.2.11 brotli/1.0.9 libidn2/2.3.0 libpsl/0.21.1 [2024-04-01T19:50:55.991747](+libidn2/2.3.0) libssh/0.10.4/openssl/zlib nghttp2/1.43.0"}`,
			wantErr:          true,
		},
		{
			name:             "subtly malformed JSON",
			prefixedCurlJSON: `@NV@{"content_type":foo,"errormsg":"SSL certificate problem: unable to get local issuer certificate","exitcode":60,"filename_effective":null,"ftp_entry_path":null,"http_code":0,"http_connect":0,"http_version":"0","local_ip":"172.31.2.213","local_port":51232,"method":"HEAD","num_connects":1,"num_headers":0,"num_redirects":0,"proxy_ssl_verify_result":0,"redirect_url":null,"referer":null,"remote_ip":"52.55.72.119","remote_port":443,"response_code":0,"scheme":"HTTPS","size_download":0,"size_header":0,"size_request":0,"size_upload":0,"speed_download":0,"speed_upload":0,"ssl_verify_result":20,"time_appconnect":0.000000,"time_connect":0.053023,"time_namelookup":0.009450,"time_pretransfer":0.000000,"time_redirect":0.000000,"time_starttransfer":0.000000,"time_total":0.376118,"url":"https://infogw.api.openshift.com:443","url_effective":"https://infogw.api.openshift.com:443/","urlnum":13,"curl_version":"libcurl/7.76.1 OpenSSL/3.0.7 zlib/1.2.11 brotli/1.0.9 libidn2/2.3.0 libpsl/0.21.1 [2024-04-01T19:50:55.991747](+libidn2/2.3.0) libssh/0.10.4/openssl/zlib nghttp2/1.43.0"}`,
			wantErr:          true,
		},
		{
			name:             "garbage input",
			prefixedCurlJSON: "foobar",
			wantErr:          true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := deserializeCurlJSONProbeResult(tt.prefixedCurlJSON)
			if (err != nil) != tt.wantErr {
				t.Errorf("deserializeProbeOutputLine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("deserializeProbeOutputLine() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_bulkDeserializeCurlJSONProbeResult(t *testing.T) {
	tests := []struct {
		name            string
		serializedLines string
		want            []*CurlJSONProbeResult
		wantErrsOnLines []int
	}{
		{
			name:            "single-line garbage input",
			serializedLines: "foobar",
			wantErrsOnLines: []int{0},
		},
		{
			name:            "empty struct surrounded by garbage lines",
			serializedLines: "foobar\n@NV@{}\n@NV@fizz{}buzz",
			want:            []*CurlJSONProbeResult{{}},
			wantErrsOnLines: []int{0, 2},
		},
		{
			name: "legit input",
			serializedLines: `@NV@{"content_type":"text/html; charset=utf-8","errormsg":null,"exitcode":0,"filename_effective":null,"ftp_entry_path":null,"http_code":200,"http_connect":0,"http_version":"2","local_ip":"","local_port":-1,"method":"HEAD","num_connects":0,"num_headers":10,"num_redirects":0,"proxy_ssl_verify_result":0,"redirect_url":null,"referer":null,"remote_ip":"23.20.243.242","remote_port":443,"response_code":200,"scheme":"HTTPS","size_download":0,"size_header":514,"size_request":70,"size_upload":0,"speed_download":0,"speed_upload":0,"ssl_verify_result":0,"time_appconnect":0.000111,"time_connect":0.000111,"time_namelookup":0.000111,"time_pretransfer":0.000146,"time_redirect":0.000000,"time_starttransfer":0.997047,"time_total":0.997072,"url":"https://quay.io:443","url_effective":"https://quay.io:443/","urlnum":1,"curl_version":"libcurl/7.76.1 OpenSSL/3.0.7 zlib/1.2.11 brotli/1.0.9 libidn2/2.3.0 libpsl/0.21.1 (+libidn2/2.3.0) libssh/0.10.4/openssl/zlib nghttp2/1.43.0"}
			@NV@{"content_type":null,"errormsg":"SSL certificate problem: unable to get local issuer certificate","exitcode":60,"filename_effective":null,"ftp_entry_path":null,"http_code":0,"http_connect":0,"http_version":"0","local_ip":"172.31.2.213","local_port":51232,"method":"HEAD","num_connects":1,"num_headers":0,"num_redirects":0,"proxy_ssl_verify_result":0,"redirect_url":null,"referer":null,"remote_ip":"52.55.72.119","remote_port":443,"response_code":0,"scheme":"HTTPS","size_download":0,"size_header":0,"size_request":0,"size_upload":0,"speed_download":0,"speed_upload":0,"ssl_verify_result":20,"time_appconnect":0.000000,"time_connect":0.053023,"time_namelookup":0.009450,"time_pretransfer":0.000000,"time_redirect":0.000000,"time_starttransfer":0.000000,"time_total":0.376118,"url":"https://infogw.api.openshift.com:443","url_effective":"https://infogw.api.openshift.com:443/","urlnum":13,"curl_version":"libcurl/7.76.1 OpenSSL/3.0.7 zlib/1.2.11 brotli/1.0.9 libidn2/2.3.0 libpsl/0.21.1 [2024-04-01T19:50:55.991747](+libidn2/2.3.0) libssh/0.10.4/openssl/zlib nghttp2/1.43.0"}`,
			want: []*CurlJSONProbeResult{
				{
					ContentType:       "text/html; charset=utf-8",
					HTTPCode:          200,
					HTTPVersion:       "2",
					LocalPort:         -1,
					Method:            "HEAD",
					NumHeaders:        10,
					RemoteIP:          "23.20.243.242",
					RemotePort:        443,
					ResponseCode:      200,
					Scheme:            "HTTPS",
					SizeHeader:        514,
					SizeRequest:       70,
					TimeAppConnect:    0.000111,
					TimeConnect:       0.000111,
					TimeNameLookup:    0.000111,
					TimePreTransfer:   0.000146,
					TimeStartTransfer: 0.997047,
					TimeTotal:         0.997072,
					URL:               "https://quay.io:443",
					URLEffective:      "https://quay.io:443/",
					URLNum:            1,
					CurlVersion:       "libcurl/7.76.1 OpenSSL/3.0.7 zlib/1.2.11 brotli/1.0.9 libidn2/2.3.0 libpsl/0.21.1 (+libidn2/2.3.0) libssh/0.10.4/openssl/zlib nghttp2/1.43.0",
				},
				{
					ErrorMsg:        "SSL certificate problem: unable to get local issuer certificate",
					ExitCode:        60,
					HTTPVersion:     "0",
					LocalIP:         "172.31.2.213",
					LocalPort:       51232,
					Method:          "HEAD",
					NumConnects:     1,
					RemoteIP:        "52.55.72.119",
					RemotePort:      443,
					Scheme:          "HTTPS",
					SSLVerifyResult: 20,
					TimeConnect:     0.053023,
					TimeNameLookup:  0.00945,
					TimeTotal:       0.376118,
					URL:             "https://infogw.api.openshift.com:443",
					URLEffective:    "https://infogw.api.openshift.com:443/",
					URLNum:          13,
					CurlVersion:     "libcurl/7.76.1 OpenSSL/3.0.7 zlib/1.2.11 brotli/1.0.9 libidn2/2.3.0 libpsl/0.21.1 [2024-04-01T19:50:55.991747](+libidn2/2.3.0) libssh/0.10.4/openssl/zlib nghttp2/1.43.0",
				}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, errs := bulkDeserializeCurlJSONProbeResult(tt.serializedLines)
			errKeys := mapKeys(errs)
			// Need to sort both slices to account for mapKeys() having indeterminate ordering
			slices.Sort(errKeys)
			slices.Sort(tt.wantErrsOnLines)
			if tt.wantErrsOnLines != nil && !reflect.DeepEqual(tt.wantErrsOnLines, errKeys) {
				t.Errorf("bulkDeserializePrefixedCurlJSON() want errors on lines %v of input %q, instead got %v", tt.wantErrsOnLines, tt.serializedLines, errs)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("bulkDeserializePrefixedCurlJSON() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// Keys returns the keys of the map m. The keys will be an indeterminate order.
// Borrowed from https://cs.opensource.google/go/x/exp/+/39d4317d:maps/maps.go;l=10
func mapKeys[M ~map[K]V, K comparable, V any](m M) []K {
	r := make([]K, 0, len(m))
	for k := range m {
		r = append(r, k)
	}
	return r
}
