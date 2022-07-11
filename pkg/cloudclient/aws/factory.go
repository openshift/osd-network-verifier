package aws

import (
	"context"
	"fmt"
	"os"

	"github.com/openshift/osd-network-verifier/pkg/cloudclient"
	"github.com/openshift/osd-network-verifier/pkg/utils"
)

func init() {
	cloudclient.Register(
		ClientIdentifier,
		produceAWS,
	)
}

var (
	DefaultTagsAWS     = map[string]string{"osd-network-verifier": "owned", "red-hat-managed": "true", "Name": "osd-network-verifier"}
	RegionEnvVarStrAWS = "AWS_REGION"
	RegionDefaultAWS   = "us-east-1"
)

// Precedence: cli > env var > default
func getAWSRegion(options utils.AWSClientConfig) string {
	if options.Region != "" {
		return options.Region
	}
	val, present := os.LookupEnv(RegionEnvVarStrAWS)
	if present {
		return val
	} else {
		return RegionDefaultAWS
	}
}

// produceAWS isolates the AWS specific logic from cloudclient GetClientFor.
// This is the factory function for cloudclient.GetClientFor()
// where utils.platformType() decided by cmdOptions or env vars returns "AWS"
func produceAWS(clientConfig *cloudclient.ClientConfig, execConfig *cloudclient.ExecConfig) (cloudclient.CloudClient, error) {
	if clientConfig.AWSConfig.AwsProfile != "" {
		execConfig.Logger.Info(context.TODO(), "Using AWS profile: %s.", clientConfig.AWSConfig.AwsProfile)
	} else {
		execConfig.Logger.Info(context.TODO(), "Using AWS secret key")
	}
	if clientConfig.AWSConfig.CloudTags == nil {
		clientConfig.AWSConfig.CloudTags = DefaultTagsAWS

	}
	clientConfig.AWSConfig.SecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY") //https://issues.redhat.com/browse/OSD-12432
	clientConfig.AWSConfig.AccessKeyId = os.Getenv("AWS_ACCESS_KEY_ID")         //https://issues.redhat.com/browse/OSD-12432
	clientConfig.AWSConfig.SecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY") //https://issues.redhat.com/browse/OSD-12432
	input := ClientInput{
		Ctx:    execConfig.Ctx,
		Logger: execConfig.Logger,

		ExecConfig:   execConfig,
		ClientConfig: clientConfig,
	}
	client, err := newClient(&input)
	if err != nil {
		return nil, fmt.Errorf("unable to create AWS client %w", err)
	}

	if err != nil {
		return nil, err
	}

	return client, nil
}
