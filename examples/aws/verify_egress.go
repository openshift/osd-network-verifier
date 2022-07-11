package aws

import (
	"context"
	"fmt"

	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/cloudclient"
)

func extendValidateEgressV1() {
	//---------Set client options---------
	logger, _ := ocmlog.NewStdLoggerBuilder().Debug(true).Build()
	cmdOptions := cloudclient.CmdOptions{
		Region:     "us-east-1",                       // optional
		CloudTags:  map[string]string{"key1": "val1"}, // optional
		AwsProfile: "yourAwsProfile",                  // optional
		CloudType:  "aws",
		Logger:     logger,
		Ctx:        context.Background(),
	}

	//---------Set test parameters---------
	params := cloudclient.ValidateEgress{VpcSubnetID: "test-subnet-id"}

	//---------create ONV cloud client---------
	client, err := cloudclient.GetClientFor(&cmdOptions)
	if err != nil {
		fmt.Errorf("error creating cloud client: %s", err.Error())
	}

	// Call egress validator
	out := client.ValidateEgress(params)

	// Interpret output
	if !out.IsSuccessful() {
		// Retrieve errors
		failures, exceptions, errors := out.Parse()

		// Use returned exceptions
		fmt.Println(failures)
		fmt.Println(exceptions)
		fmt.Println(errors)
	}
}
