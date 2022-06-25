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

	logger, _ := ocmlog.NewStdLoggerBuilder().Debug(true).Build()

	//---------create ONV cloud client---------
	cli, err := cloudclient.NewClient(context.TODO(), logger, cmdOptions)
	if err != nil {
		fmt.Errorf("Error creating cloud client: %s", err.Error())
	}

	// Call egress validator
	out := cli.ValidateEgress(context.TODO())
	if !out.IsSuccessful() {
		// Retrieve errors
		failures, exceptions, errors := out.Parse()

		// Use returned exceptions
		fmt.Println(failures)
		fmt.Println(exceptions)
		fmt.Println(errors)
	}
}
