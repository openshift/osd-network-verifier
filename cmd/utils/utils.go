package utils

import (
	"errors"
	"os"

	awsverifier "github.com/openshift/osd-network-verifier/pkg/verifier/aws"
)

// GetAwsVerifier returns a verifier client from a profile or ENV vars if set
func GetAwsVerifier(region, profile string, debug bool) (*awsverifier.AwsVerifier, error) {
	accessKey := ""
	secretAccessKey := ""
	sessionsToken := ""
	if profile == "" {
		accessKey = os.Getenv("AWS_ACCESS_KEY_ID")
		secretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
		sessionsToken = os.Getenv("AWS_SESSION_TOKEN")
		if accessKey == "" || secretAccessKey == "" {
			return &awsverifier.AwsVerifier{}, errors.New("no Profile provide and no ENV set for AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY")
		}
	}

	return awsverifier.NewAwsVerifier(accessKey, secretAccessKey, sessionsToken, region, profile, debug)
}
