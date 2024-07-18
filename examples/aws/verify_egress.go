package aws

// --- Example file on how to call egress inluding exampoe on proxy config ---
import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/openshift/osd-network-verifier/pkg/data/cpu"
	"github.com/openshift/osd-network-verifier/pkg/helpers"
	"github.com/openshift/osd-network-verifier/pkg/probes/curl"
	"github.com/openshift/osd-network-verifier/pkg/proxy"
	"github.com/openshift/osd-network-verifier/pkg/verifier"
	awsverifier "github.com/openshift/osd-network-verifier/pkg/verifier/aws"
)

func extendValidateEgress() {
	//---------Initialize required args---------
	// Read AWS creds from environment
	key, _ := os.LookupEnv("AWS_ACCESS_KEY_ID")
	secret, _ := os.LookupEnv("AWS_SECRET_ACCESS_KEY")
	session, _ := os.LookupEnv("AWS_SESSION_TOKEN")

	// Create the Verifier Client pass in cred to client builder
	awsVerifier, err := awsverifier.NewAwsVerifier(key, secret, session, "us-east-1", "", true)
	if err != nil {
		fmt.Printf("Oh no I have an error Err: %s", err)
		os.Exit(1)
	}

	//---------egress verifier usage---------

	//---------Proxy setup if necessary------
	p := proxy.ProxyConfig{
		HttpProxy:  "http://user:pass@x.x.x.x:8888",
		HttpsProxy: "https://user:pass@x.x.x.x:8888",
		Cacert: `-----BEGIN RSA PRIVATE KEY-----
	EXAMPLEcertificatestartingwithcharsKCAQEAtyzg96LnZG9GIICiZmJbCtFvYwZNtzblGBFgcqBHlWMy0wjd
	0mLSC6SJzmbZAiA4XU5pT/BfqKZiZzQ1cjVFmXvp2yo82ZFgccXj61Mx2zQd8eDk
	4nYz790DWRauWCr+7cpkAwcKv8WYHuQwBd+q/lTw3z2/Qk8d/7rvzcQ=
	-----END RSA PRIVATE KEY-----
	-----BEGIN CERTIFICATE-----
	YlsK0W9jBk23NuUYEWByoEeVYzLTCN3SYtILpInOZtkCIDhdTmlP8F+opmJnNDVy
	NUWZe+nbKjzZkWBxxePrUzHbNB3x4OSmqobaNzuxTBHzm27BQN8gfiFxWsgStfbq
	zL2f2OsBvvcmBdLgpwcvK9VYN0mpNXhJm5K0e7aQdjhYTQ93Dw4BG15xOs11CuaS
	i87hWoaGmS4Bx8gdUx0yZnxU9D7sd9/5Nz6s1J4riLWsz/InVw7Rr1NGTpLDojjX
	9hieOYBpwE763AECJrtxyRYHhXZ1DiKEfZWAYWICf8NUGdEohNpWKuUeFbBMlEWW
	TRVfvGGNFuJkfkh4rR09wHvlmyzSVJ6le6iaQ0wlp2S0j9oC2A==
	-----END CERTIFICATE-----`,
		NoTls: false,
	}

	// Create the egress input
	vei := verifier.ValidateEgressInput{
		Ctx:          context.TODO(),
		SubnetID:     "vpcSubnetID",
		CloudImageID: "cloudImageID",
		Timeout:      3 * time.Second,
		Tags:         map[string]string{"key1": "val1"},
		InstanceType: "m5.2xlarge",
		Proxy:        p,
		AWS: verifier.AwsEgressConfig{
			KmsKeyID:         "kmskeyID",
			SecurityGroupIDs: []string{"SecurityGroupID1", "OptionalSecurityGroupID2"},
		},
		PlatformType:    helpers.PlatformAWS,
		Probe:           curl.Probe{},
		CPUArchitecture: cpu.ArchX86,
	}

	// Call egress function with either gcp or aws client
	out := verifier.
		ValidateEgress(awsVerifier, vei)
	if !out.IsSuccessful() {
		// Retrieve errors
		failures, exceptions, errors := out.Parse()

		// Use returned exceptions
		fmt.Println(failures)
		fmt.Println(exceptions)
		fmt.Println(errors)
	}

}
