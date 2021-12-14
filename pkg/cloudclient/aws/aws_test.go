package aws

import (
	"testing"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

func TestNewClient(t *testing.T) {

	creds := credentials.NewStaticCredentialsProvider("dummyID", "dummyPassKey", "dummyToken")
	region := "us-east-1"
	cli, err := NewClient(creds, region)

	if err != nil {

		t.Error("err occured while creating cli:", err)

	}

	if cli == nil {
		t.Errorf("cli should have been initialized")
	}
}

