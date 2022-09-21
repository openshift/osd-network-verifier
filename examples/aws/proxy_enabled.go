package aws

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/cloudclient"
	"github.com/openshift/osd-network-verifier/pkg/proxy"
)

func ValidateEgressWithProxy() {
	//---------Initialize required args---------
	// Read AWS creds from environment
	key, _ := os.LookupEnv("AWS_ACCESS_KEY_ID")
	secret, _ := os.LookupEnv("AWS_SECRET_ACCESS_KEY")
	session, _ := os.LookupEnv("AWS_SESSION_TOKEN")

	// Build the v1 credentials
	creds := credentials.NewStaticCredentials(key, secret, session)

	// Example required values
	logger, _ := ocmlog.NewStdLoggerBuilder().Debug(true).Build()
	region := "us-east-1"
	instanceType := "m5.2xlarge"
	tags := map[string]string{"key1": "val1"}

	//---------ONV egress verifier usage---------
	cli, _ := cloudclient.NewClient(context.TODO(), logger, *creds, region, instanceType, tags)

	// Proxy Setup
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

	// Call egress validator
	out := cli.ValidateEgress(context.TODO(), "vpcSubnetID", "cloudImageID", "kmsKeyID", "securityGroupId", 3*time.Second, p)
	if !out.IsSuccessful() {
		// Retrieve errors
		failures, exceptions, errors := out.Parse()

		// Use returned exceptions
		fmt.Println(failures)
		fmt.Println(exceptions)
		fmt.Println(errors)
	}
}
