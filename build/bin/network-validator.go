package main

// Usage
// $ network-validator --timeout=1s --config=config/config.yaml

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
)

var (
	maxRetries     = flag.Int("max-retries", 3, "Maximum connection attempts per endpoint")
	timeout        = flag.Duration("timeout", 2000*time.Millisecond, "Timeout for each dial request made")
	configFilePath = flag.String("config", "config.yaml", "Path to configuration file")
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
		err = fmt.Errorf("unable to reach config file %v: %v", configFilePath, err)
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

	endpointPortCount := 0
	for _, e := range config.Endpoints {
		for range e.Ports {
			endpointPortCount++
		}
	}

	failures := make(chan error, endpointPortCount)
	for _, e := range config.Endpoints {
		for _, port := range e.Ports {
			waitGroup.Add(1)
			// Validate the endpoints in parallel
			go func(host string, port int, tlsDisabled bool, failures chan<- error) {
				defer waitGroup.Done()
				err := ValidateReachability(host, port, tlsDisabled)
				if err != nil {
					failures <- err
				}
			}(e.Host, port, e.TLSDisabled, failures)
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

func ValidateReachability(host string, port int, tlsDisabled bool) error {
	var err error
	endpoint := fmt.Sprintf("%s:%d", host, port)
	httpClient := http.Client{
		Timeout: *timeout,
	}

	// #nosec G402 -- Low chance of MITM, as the instance is short-lived, see OHSS-11465
	if tlsDisabled {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	fmt.Printf("Validating %s\n", endpoint)

	// Retry up to maxRetries times
	for i := 0; i < *maxRetries; i++ {
		switch port {
		case 80:
			_, err = httpClient.Get(fmt.Sprintf("%s://%s", "http", host))
		case 443:
			_, err = httpClient.Get(fmt.Sprintf("%s://%s", "https", host))
		default:
			_, err = net.DialTimeout("tcp", endpoint, *timeout)
		}

		// Only continue retrying if there's an error
		if err == nil {
			break
		}
	}

	if err != nil {
		return fmt.Errorf("Unable to reach %s within specified timeout after %d retries: %s", endpoint, *maxRetries, err)
	}

	return nil
}
