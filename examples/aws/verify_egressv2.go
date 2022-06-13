package aws

import (
	"context"
	"fmt"
	"time"

	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/cloudclient"
)

func extendValidateEgressV2() {
	//---------initialize required args---------

	// Example values
	logger, _ := ocmlog.NewStdLoggerBuilder().Debug(true).Build()
	region := "us-east-1"
	instanceType := "m5.2xlarge"
	tags := map[string]string{"key1": "val1"}
	awsProfile := "yourAwsProfile"

	//---------ONV egress verifier usage---------
	cli, _ := cloudclient.NewClient(context.TODO(), logger, region, instanceType, tags, "aws", awsProfile)
	// Call egress validator
	out := cli.ValidateEgress(context.TODO(), "vpcSubnetID", "cloudImageID", "kmsKeyID", 3*time.Second)
	if !out.IsSuccessful() {
		// Retrieve errors
		failures, exceptions, errors := out.Parse()
		// Use returned exceptions
		fmt.Println(failures)
		fmt.Println(exceptions)
		fmt.Println(errors)
	}
}
