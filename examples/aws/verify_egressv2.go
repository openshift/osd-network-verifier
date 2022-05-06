package aws

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/credentials"
	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/cloudclient"
)

// Example config struct
type egressConfigV2 struct {
	debug bool
}

func extendValidateEgressV2() {
	ctx := context.TODO()
	//---------initialize required args---------
	// Read AWS creds from environment
	key, _ := os.LookupEnv("AWS_ACCESS_KEY_ID")
	secret, _ := os.LookupEnv("AWS_SECRET_ACCESS_KEY")
	session, _ := os.LookupEnv("AWS_SESSION_TOKEN")

	// Build the v2 credentials provider
	creds := credentials.NewStaticCredentialsProvider(key, secret, session)

	// Example required values
	logger, _ := ocmlog.NewStdLoggerBuilder().Debug(true).Build()
	region := "us-east-1"
	instanceType := "m5.2xlarge"
	tags := make(map[string]string)
	tags["key1"] = "val1"

	//---------ONV egress verifier usage---------
	cli, _ := cloudclient.NewClient(ctx, logger, creds, region, instanceType, tags)
	// Call egress validator
	out := cli.ValidateEgress(context.TODO(), "vpcSubnetID", "cloudImageID", "kmsKeyID", 600)
	if !out.IsSuccessful() {
		// Retrieve errors
		failures, exceptions, errors := out.Parse()
		// Use returned exceptions
		fmt.Println(failures)
		fmt.Println(exceptions)
		fmt.Println(errors)
	}

}
