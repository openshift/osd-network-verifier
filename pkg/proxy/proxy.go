package proxy

type ProxyConfig struct {
	HttpProxy  string
	HttpsProxy string
	Cacert     string
	NoTls      bool
}
