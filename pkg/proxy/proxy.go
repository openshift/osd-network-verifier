package proxy

import "strings"

type ProxyConfig struct {
	HttpProxy  string
	HttpsProxy string
	Cacert     string
	NoTls      bool

	// NoProxy is a slice of domains or IPs that should not be proxied through the configured Proxy
	NoProxy []string
}

// NoProxyAsString joins the slice of domains and/or IPs on the delimeter "," for easy passing to the configured Probe.
func (p *ProxyConfig) NoProxyAsString() string {
	return strings.Join(p.NoProxy, ",")
}
