package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/openshift/osd-network-verifier/integration/pkg/aws"
	"github.com/openshift/osd-network-verifier/pkg/verifier"
	awsverifier "github.com/openshift/osd-network-verifier/pkg/verifier/aws"

	"github.com/aws/aws-sdk-go-v2/config"
)

func main() {
	region := flag.String("region", "us-east-1", "AWS Region")
	flag.Parse()

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(*region))
	if err != nil {
		panic(err)
	}

	data := aws.NewIntegrationTestData(cfg)
	if err := data.Setup(context.TODO()); err != nil {
		log.Printf("setup err, starting cleanup: %s", err)
		if err := data.Cleanup(context.TODO()); err != nil {
			panic(err)
		}
	}

	if err := onvEgressCheck(*region, *profile, *data.GetPrivateSubnetId()); err != nil {
		panic(err)
	}

	if err := data.Cleanup(context.TODO()); err != nil {
		panic(err)
	}
}

func onvEgressCheck(region, profile, subnetId string) error {
	// Read AWS credentials from environment
	key, _ := os.LookupEnv("AWS_ACCESS_KEY_ID")
	secret, _ := os.LookupEnv("AWS_SECRET_ACCESS_KEY")
	session, _ := os.LookupEnv("AWS_SESSION_TOKEN")

	awsVerifier, err := awsverifier.NewAwsVerifier(key, secret, session, region, profile, false)
	if err != nil {
		return err
	}

	// Example required values
	defaultTags := map[string]string{"osd-network-verifier": "owned", "red-hat-managed": "true", "Name": "osd-network-verifier"}

	vei := verifier.ValidateEgressInput{
		Timeout:      2 * time.Second,
		Ctx:          context.TODO(),
		SubnetID:     subnetId,
		InstanceType: "t3.micro",
		Tags:         defaultTags,
	}

	// Call egress validator
	log.Println("Starting ONV egress validation")
	out := verifier.ValidateEgress(awsVerifier, vei)
	out.Summary(false)

	if out.IsSuccessful() {
		log.Println("ONV egress validation: Success!")
	} else {
		log.Println("ONV egress validation: Failure!")
	}

	return nil
}
