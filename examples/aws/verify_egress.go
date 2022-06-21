package aws

import (
	"context"
	"fmt"
	"time"

	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/cloudclient"
)

func extendValidateEgressV1() {
	//---------Set commandline args---------
	cmdOptions := cloudclient.CmdOptions{
		Region:     "us-east-1",
		CloudTags:  map[string]string{"key1": "val1"},
		AwsProfile: "yourAwsProfile",
	}
	instanceType := "m5.2xlarge"
	vpcSubnetId := "subnet-xxxxxxxxxxxxxx"
	logger, _ := ocmlog.NewStdLoggerBuilder().Debug(true).Build()
	//---------create ONV cloud client---------
	cli, _ := cloudclient.NewClient(context.TODO(), logger, instanceType, "aws", cmdOptions)
	// Call egress validator
	out := cli.ValidateEgress(context.TODO(), vpcSubnetId, "cloudImageID", "kmsKeyID", 3*time.Second)
	if !out.IsSuccessful() {
		// Retrieve errors
		failures, exceptions, errors := out.Parse()

		// Use returned exceptions
		fmt.Println(failures)
		fmt.Println(exceptions)
		fmt.Println(errors)
	}
}
