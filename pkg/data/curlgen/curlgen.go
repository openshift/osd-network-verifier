package curlgen

import (
	"fmt"
	"strconv"
)

// Options struct contains flag options that will be used to build
// the curl command. Every Option directly maps to a flag that can
// be used to configure your curl command.
type Options struct {
	CaPath          string
	ProxyCaPath     string
	Retry           int
	MaxTime         string
	NoTls           string
	Urls            string
	TlsDisabledUrls string
}

const DefaultCurlOutputSeparator = "@NV@"

// GenerateString function will be used to transform the Configurations (options)
// used to build the Options struct and build a full Curl command and return it as a string
func GenerateString(cfg *Options) (string, error) {
	command := fmt.Sprintf(`curl --capath %s --proxy-capath %s --retry %v --retry-connrefused -t B -Z -s -I -m %s -w "%%{stderr}%s%%{json}\n"`,
		cfg.CaPath,
		cfg.ProxyCaPath,
		cfg.Retry,
		cfg.MaxTime,
		DefaultCurlOutputSeparator,
	)

	if cfg.NoTLS() {
		command += " --insecure"
		// In addition to adding the curl flag, we can merge the list of "tlsDisabled" URLs
		// with the list of "normal" URLs now (since all URLs will be "tlsDisabled")
		cfg.Urls += " " + cfg.TlsDisabledUrls
		cfg.TlsDisabledUrls = ""
	}

	command += " " + cfg.Urls + " --proto =http,https,telnet"

	if cfg.TlsDisabledUrls != "" {
		command += fmt.Sprintf(
			` --next --insecure --retry %v --retry-connrefused -s -I -m %s -w "%%{stderr}%s%%{json}\n" %s --proto =https`,
			cfg.Retry,
			cfg.MaxTime,
			DefaultCurlOutputSeparator,
			cfg.TlsDisabledUrls,
		)
	}

	return command, nil

}
func (o *Options) NoTLS() bool {
	noTLS, _ := strconv.ParseBool(o.NoTls)
	return noTLS
}
