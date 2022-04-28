package main

// Usage
// $ network-verifier --timeout=1s --config=config/config.yaml

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"time"

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
	// need to verify any CDN such as `cdn01.quay.io` should be available?
	//  We don't need to. We just best-effort check what we can.

	failures := []error{}
	for _, e := range config.Endpoints {
		for _, port := range e.Ports {
			err := VerifyReachability(e.Host, port)
			if err != nil {
				failures = append(failures, err)
			}
		}
	}

	if len(failures) < 1 {
		fmt.Println("Success!")
		return
	}
	fmt.Println("\nNot all endpoints were reachable:")
	for _, f := range failures {
		fmt.Println(f)
	}
	// NOTE even though not all endpoints were reachable, the script still completed successfully. To ensure
	// the docker image run doesn't abort when not all endpoints are reachable, we exit with a 0 code here.
	// This may make it difficult when directly using the script to rely on the exit code to determine if
	// endpoints were reachable or not, but there is not a use case for that as of this writing.
	os.Exit(0)
}

func VerifyReachability(host string, port int) error {
	endpoint := fmt.Sprintf("%s:%d", host, port)
	fmt.Printf("Verifying %s\n", endpoint)
	_, err := net.DialTimeout("tcp", endpoint, *timeout)
	if err != nil {
		return fmt.Errorf("Unable to reach %s within specified timeout: %s", endpoint, err)
	}
	return nil
}
