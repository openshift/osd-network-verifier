package aws

// verify VPC egress access with AWS SDK v1
import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws/credentials"
	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/cloudclient"
)

type egressConfigV1 struct {
	debug bool
}

// Use egress validator
func extendValidateEgressV1(ctx context.Context) error {
	//initialize required args
	//---------
	//read AWS creds from environment
	key, _ := os.LookupEnv("AWS_ACCESS_KEY_ID")
	secret, _ := os.LookupEnv("AWS_SECRET_ACCESS_KEY")
	session, _ := os.LookupEnv("AWS_SESSION_TOKEN")
	//build the v2 credentials
	creds := credentials.NewStaticCredentials(key, secret, session)
	region := "us-east-1"
	instanceType := "m5.2xlarge"
	tags := make(map[string]string)
	tags["key1"] = "val1"
	builder := ocmlog.NewStdLoggerBuilder()
	config := egressConfigV1{}
	builder.Debug(config.debug)
	logger, _ := builder.Build()
	//---------

	// init cloudclient
	cli, _ := cloudclient.NewClient(ctx, logger, *creds, region, instanceType, tags)
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
