package aws

import (
	"context"
	"fmt"

	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/cloudclient"
	"github.com/openshift/osd-network-verifier/pkg/utils"
)

func extendValidateEgressV1() {
	//---------Set client and execution configs---------
	awsClientConfig := utils.AWSClientConfig{ // optional
		Region:     "us-east-1",
		CloudTags:  map[string]string{"key1": "val1"},
		AwsProfile: "yourAwsProfile",
	}
	clientConfig := cloudclient.ClientConfig{AWSConfig: &awsClientConfig}
	logger, _ := ocmlog.NewStdLoggerBuilder().Debug(true).Build()
	execConfig := cloudclient.ExecConfig{
		Logger: logger,
		Ctx:    context.Background(),
	}
	//---------Set test parameters---------
	params := cloudclient.ValidateEgress{VpcSubnetID: "test-subnet-id"}

	//---------call ONV cloud client factory---------
	client, err := cloudclient.GetClientFor(&clientConfig, &execConfig)
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
