package main

// Usage
// $ network-validator --timeout=1s --config=config/config.yaml

import (
	"flag"
	"fmt"
	"io/ioutil"
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
	Host  string `yaml:"host"`
	Ports []int  `yaml:"ports"`
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

	failures := make(chan error, len(config.Endpoints))
	for _, e := range config.Endpoints {
		for _, port := range e.Ports {
			waitGroup.Add(1)
			// Validate the endpoints in parallel
			go func(host string, port int, failures chan<- error) {
				defer waitGroup.Done()
				err := ValidateReachability(host, port)
				if err != nil {
					failures <- err
				}
			}(e.Host, port, failures)
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

func ValidateReachability(host string, port int) error {
	var err error
	endpoint := fmt.Sprintf("%s:%d", host, port)
	httpClient := http.Client{
		Timeout: *timeout,
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
		return fmt.Errorf("unable to reach %s within specified timeout: %s", endpoint, err)
	}

	return nil
}
