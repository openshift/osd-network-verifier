package gcp_curl

import (
	"encoding/json"
	"fmt"
	"strings"
)

// A CurlJSONProbeResult represents all the data the curl probe had to offer regarding its
// attempt(s) to reach a single URL. This struct is based on the fields curl v7.76.1
// prints when given the `--write-out "%{json}"` flag. We only use a small fraction of
// the fields listed below; all others are included for potential future use
type CurlJSONProbeResult struct {
	ContentType          string  `json:"content_type"`
	ErrorMsg             string  `json:"errormsg"`
	ExitCode             int     `json:"exitcode"`
	FilenameEffective    string  `json:"filename_effective"`
	FTPEntryPath         string  `json:"ftp_entry_path"`
	HTTPCode             int     `json:"http_code"`
	HTTPConnect          int     `json:"http_connect"`
	HTTPVersion          string  `json:"http_version"`
	LocalIP              string  `json:"local_ip"`
	LocalPort            int     `json:"local_port"`
	Method               string  `json:"method"`
	NumConnects          int     `json:"num_connects"`
	NumHeaders           int     `json:"num_headers"`
	NumRedirects         int     `json:"num_redirects"`
	ProxySSLVerifyResult int     `json:"proxy_ssl_verify_result"`
	RedirectURL          string  `json:"redirect_url"`
	Referer              string  `json:"referer"`
	RemoteIP             string  `json:"remote_ip"`
	RemotePort           int     `json:"remote_port"`
	ResponseCode         int     `json:"response_code"`
	Scheme               string  `json:"scheme"`
	SizeDownload         int     `json:"size_download"`
	SizeHeader           int     `json:"size_header"`
	SizeRequest          int     `json:"size_request"`
	SizeUpload           int     `json:"size_upload"`
	SpeedDownload        int     `json:"speed_download"`
	SpeedUpload          int     `json:"speed_upload"`
	SSLVerifyResult      int     `json:"ssl_verify_result"`
	TimeAppConnect       float64 `json:"time_appconnect"`
	TimeConnect          float64 `json:"time_connect"`
	TimeNameLookup       float64 `json:"time_namelookup"`
	TimePreTransfer      float64 `json:"time_pretransfer"`
	TimeRedirect         float64 `json:"time_redirect"`
	TimeStartTransfer    float64 `json:"time_starttransfer"`
	TimeTotal            float64 `json:"time_total"`
	URL                  string  `json:"url"`
	URLEffective         string  `json:"url_effective"`
	URLNum               int     `json:"urlnum"`
	CurlVersion          string  `json:"curl_version"`
}

// IsSuccessfulConnection returns true if the CurlJSONProbeResult reports a successful
// connection to its URLEffective, based on curl's exit code
func (res CurlJSONProbeResult) IsSuccessfulConnection() bool {
	// Null/empty scheme string always indicates failed connection
	if len(res.Scheme) < 1 {
		return false
	}

	scheme := strings.ToUpper(res.Scheme)
	if strings.Contains(scheme, "HTTP") {
		// HTTP(S): 0 is the only "fully successful" exit code
		return res.ExitCode == 0
	}
	if strings.Contains(scheme, "TELNET") {
		// TELNET: this probe uses a telnet/curl hack to check
		// egress to non-HTTP(S) endpoints (e.g., Splunk's port
		// 9997 protocol). See "Why telnet?" in the description of
		// https://github.com/openshift/osd-network-verifier/pull/204
		// tl;dr: exit code 49 == successful connection. Any other
		// exit code means we can assume the egress is blocked. Ugly,
		// but it works.
		return res.ExitCode == 49
	}

	// TODO report error here (unknown protocol)
	return false
}

// bulkDeserializeCurlJSONProbeResult wraps deserializeCurlJSONProbeResult, creating a
// CurlJSONProbeResult from a each line (containing prefixed JSON) of the provided
// string. A slice of successfully-deserialized CurlJSONProbeResult-pointers is returned
// along with a mapping between any malformed lines and their line numbers
func bulkDeserializeCurlJSONProbeResult(serializedLines string) ([]*CurlJSONProbeResult, map[int]error) {
	var results []*CurlJSONProbeResult
	deserializationErrs := make(map[int]error)
	for lineNum, serializedLine := range strings.Split(serializedLines, "\n") {
		probeResultPtr, err := deserializeCurlJSONProbeResult(serializedLine)
		if err != nil {
			deserializationErrs[lineNum] = err
		}
		if probeResultPtr != nil {
			results = append(results, probeResultPtr)
		}
	}
	return results, deserializationErrs
}

// deserializeCurlJSONProbeResult creates a CurlJSONProbeResult from a single line of
// probe console output, which should start with outputLinePrefix followed by a
// serialized JSON string. If the prefix is missing or JSON deserialization
// (unmarshalling) fails, (nil, error) is returned
func deserializeCurlJSONProbeResult(prefixedCurlJSON string) (*CurlJSONProbeResult, error) {
	jsonStr, prefixFound := strings.CutPrefix(strings.TrimSpace(prefixedCurlJSON), outputLinePrefix)
	if !prefixFound {
		return nil, fmt.Errorf("missing prefix '%s': %s", outputLinePrefix, prefixedCurlJSON)
	}
	var result CurlJSONProbeResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, err
	}
	return &result, nil
}
