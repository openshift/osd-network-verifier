package main

// Usage
// $ network-validator --timeout=1s --config=config/config.yaml

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
	// need to validate any CDN such as `cdn01.quay.io` should be available?
	//  We don't need to. We just best-effort check what we can.

	failures := []error{}
	for _, e := range config.Endpoints {
		for _, port := range e.Ports {
			err := ValidateReachability(e.Host, port)
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
	os.Exit(1)
}

func ValidateReachability(host string, port int) error {
	endpoint := fmt.Sprintf("%s:%d", host, port)
	fmt.Printf("Validating %s\n", endpoint)
	_, err := net.DialTimeout("tcp", endpoint, *timeout)
	if err != nil {
		return fmt.Errorf("Unable to reach %s within specified timeout: %s", endpoint, err)
	}
	return nil
}
