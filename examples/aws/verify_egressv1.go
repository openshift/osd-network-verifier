package aws

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws/credentials"
	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/cloudclient"
)

func extendValidateEgressV1() error {
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
	tags := make(map[string]string)
	tags["key1"] = "val1"

	//---------ONV egress verifier usage---------
	cli, _ := cloudclient.NewClient(context.TODO(), logger, *creds, region, instanceType, tags)
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
	return nil
}
