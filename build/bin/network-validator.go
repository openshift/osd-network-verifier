package main

// Usage
// $ AWS_REGION=us-east-1  ./network-validator --timeout=3s --config=config/config.yaml

// validations under proxy:
// - assuming you have proxy server & https tls certs:
// export CACERT=`cat mitmproxy-ca.pem`
// $ CACERT=${CACERT} HTTP_PROXY=http://user:pass@x.x.x.x:8888 HTTPS_PROXY=https://user:pass@x.x.x.x:8888 AWS_REGION=us-east-1 ./network-validator --timeout=3s --config=../config/config.yaml --no-tls

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"gopkg.in/yaml.v2"
)

var (
	timeout        = flag.Duration("timeout", 1000*time.Millisecond, "Timeout for each dial request made")
	configFilePath = flag.String("config", "config.yaml", "Path to configuration file")
	noTls          = flag.Bool("no-tls", false, "option to ignore all ssl certificate validations on client-side. Proxy can still be passed alongside")
)

type reachabilityConfig struct {
	Endpoints []endpoint `yaml:"endpoints"`
}

func (c *reachabilityConfig) LoadFromYaml(filePath string) error {
	buf, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	// expand environment variables
	buf = []byte(os.ExpandEnv(string(buf)))
	err = yaml.Unmarshal(buf, c)
	if err != nil {
		return err
	}
	return nil
}

type endpoint struct {
	Host        string `yaml:"host"`
	Ports       []int  `yaml:"ports"`
	TLSDisabled bool   `yaml:"tlsDisabled"`
}

func main() {
	flag.Parse()
	config := reachabilityConfig{}
	err := config.LoadFromYaml(*configFilePath)
	if err != nil {
		err = fmt.Errorf("Unable to reach config file %v: %v", configFilePath, err)
		fmt.Println(err)
		os.Exit(1)
	}

	TestEndpoints(config)
}

func TestEndpoints(config reachabilityConfig) {
	// TODO how would we check for wildcard entries like the `.quay.io` entry, where we
	// need to validate any CDN such as `cdn01.quay.io` should be available?
	//  We don't need to. We just best-effort check what we can.

	var waitGroup sync.WaitGroup

	failures := make(chan error, len(config.Endpoints))
	for _, e := range config.Endpoints {
		for _, port := range e.Ports {
			waitGroup.Add(1)
			// tls decision
			tls := *noTls || e.TLSDisabled
			// Validate the endpoints in parallel
			go func(host string, port int, tlsDisabled bool, failures chan<- error) {
				defer waitGroup.Done()
				err := ValidateReachability(host, port, tlsDisabled)
				if err != nil {
					failures <- err
				}
			}(e.Host, port, tls, failures)
		}
	}
	waitGroup.Wait()
	close(failures)

	if len(failures) < 1 {
		fmt.Println("Success!")
		return
	}
	fmt.Println("\nNot all endpoints were reachable:")
	for f := range failures {
		fmt.Println(f)
	}
	// NOTE even though not all endpoints were reachable, the script still completed successfully. To ensure
	// the docker image run doesn't abort when not all endpoints are reachable, we exit with a 0 code here.
	// This may make it difficult when directly using the script to rely on the exit code to determine if
	// endpoints were reachable or not, but there is not a use case for that as of this writing.
	os.Exit(0)
}

func embedProxyCertificate(cacert string) (*x509.CertPool, error) {
	// Get the SystemCertPool, continue with an empty pool on error
	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}

	// Append our cert to the system pool
	if ok := rootCAs.AppendCertsFromPEM([]byte(cacert)); !ok {
		log.Println("No certs appended, using system certs only")
	}

	return rootCAs, nil
}

func ValidateReachability(host string, port int, tlsDisabled bool) error {
	var err error
	endpoint := fmt.Sprintf("%s:%d", host, port)
	httpClient := http.Client{
		Timeout: *timeout,
	}

	// Setup Certs
	cacert := os.Getenv("CACERT")
	var rootCAs *x509.CertPool
	if cacert != "" {
		rootCAs, err = embedProxyCertificate(cacert)
		if err != nil {
			log.Fatalf("Failed to append %q to RootCAs: %v", cacert, err)
			return err
		}
	}

	// ProxyFromEnvironment enables reading configuration from env
	// such as export HTTP_PROXY='http://us:pass@prox-server:8888' / export HTTPS_PROXY='http://us:pass@prox-server:8888'
	// Insecure mod would be enabled if tlsDisabled was put as true. This is for dev purposes: e.g when the certificate is not known by certificate authorities.
	httpClient.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: tlsDisabled,
			// RootCAs defines the set of root certificate authorities that clients use when verifying server certificates.
			// If RootCAs is nil, TLS uses the host's root CA set.
			RootCAs: rootCAs,
		},
	}

	fmt.Printf("Validating %s\n", endpoint)

	switch port {
	case 80:
		_, err = httpClient.Get(fmt.Sprintf("%s://%s", "http", host))
	case 443:
		_, err = httpClient.Get(fmt.Sprintf("%s://%s", "https", host))
	case 22:
		_, err = ssh.Dial("tcp", endpoint, &ssh.ClientConfig{HostKeyCallback: ssh.InsecureIgnoreHostKey(), Timeout: *timeout})
		if err.Error() == "ssh: handshake failed: EOF" {
			// at this point, connectivity is available
			err = nil
		}
	default:
		_, err = net.DialTimeout("tcp", endpoint, *timeout)
	}

	if err != nil {
		return fmt.Errorf("Unable to reach %s within specified timeout: %s", endpoint, err)
	}

	return nil
}
