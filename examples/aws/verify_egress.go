package aws

import (
	"context"
	"fmt"

	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/cloudclient"
)

func extendValidateEgressV1() {
	//---------Set commandline args---------
	cmdOptions := cloudclient.CmdOptions{
		Region:     "us-east-1",                       // optional
		CloudTags:  map[string]string{"key1": "val1"}, // optional
		AwsProfile: "yourAwsProfile",                  // optional
		CloudType:  "aws",                             // optional
	}
	ctx := context.TODO()
	ctx = context.WithValue(ctx, "VpcSubnetID", "example-subnet-id")
	ctx = context.WithValue(ctx, "CloudImageID", "example-cloudImageID")
	ctx = context.WithValue(ctx, "Timeout", "example-timeout")
	ctx = context.WithValue(ctx, "KmsKeyID", "example-kmsKeyID")

	logger, _ := ocmlog.NewStdLoggerBuilder().Debug(true).Build()

	//---------create ONV cloud client---------

	cli, _ := cloudclient.NewClient(ctx, logger, cmdOptions)
	// Call egress validator
	out := cli.ValidateEgress(ctx)
	if !out.IsSuccessful() {
		// Retrieve errors
		failures, exceptions, errors := out.Parse()

		// Use returned exceptions
		fmt.Println(failures)
		fmt.Println(exceptions)
		fmt.Println(errors)
	}
}
