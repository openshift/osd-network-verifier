package cloudclient

import (
	"context"
	"github.com/openshift/osd-network-verifier/pkg/cloudclient/aws"
	"os"
)

func init() {
	Register(
		aws.ClientIdentifier,
		produceAWS,
	)
}

var (
	DefaultTags     = map[string]string{"osd-network-verifier": "owned", "red-hat-managed": "true", "Name": "osd-network-verifier"}
	RegionEnvVarStr = "AWS_REGION"
	RegionDefault   = "us-east-1"
)

// Precedence: cli > env var > default
func getRegion(options CmdOptions) string {
	if options.Region != "" {
		return options.Region
	}
	val, present := os.LookupEnv(RegionEnvVarStr)
	if present {
		return val
	} else {
		return RegionDefault
	}
}

func produceAWS(options *CmdOptions) (CloudClient, error) {
	if options.AwsProfile != "" {
		options.Logger.Info(context.TODO(), "Using AWS profile: %s.", options.AwsProfile)
	} else {
		options.Logger.Info(context.TODO(), "Using AWS secret key")
	}
	if options.CloudTags == nil {
		options.CloudTags = DefaultTags

	}
	c, err := aws.NewClient(aws.ClientInput{
		Ctx:             options.Ctx,
		Logger:          options.Logger,
		CloudImageID:    options.CloudImageID,
		KmsKeyID:        options.KmsKeyID,
		Region:          getRegion(*options),
		InstanceType:    options.InstanceType,
		CloudTags:       options.CloudTags,
		Profile:         options.AwsProfile,                 //todo create env getter similar to region
		AccessKeyId:     os.Getenv("AWS_ACCESS_KEY_ID"),     //todo create env getter similar to region
		SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"), //todo create env getter similar to region
		SessionToken:    os.Getenv("AWS_SESSION_TOKEN"),     //todo create env getter similar to region
	})

	if err != nil {
		return nil, err
	}

	return c, nil
}
