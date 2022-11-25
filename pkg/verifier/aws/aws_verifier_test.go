package awsverifier

import (
	"context"
	"testing"

	awss "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/golang/mock/gomock"
	"github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/clients/aws"
	"github.com/openshift/osd-network-verifier/pkg/mocks"
)

func TestFindUnreachableEndpointsSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	FakeEC2Cli := mocks.NewMockEC2Client(ctrl)
	out := &ec2.GetConsoleOutputOutput{
		InstanceId: awss.String("dummy-instance"),
		// USERDATA BEGIN
		// Using IMAGE : e4d93a35c482
		// Validating route53domains.us-east-1.amazonaws.com:443 ...
		// Success!
		// USERDATA END
		Output: awss.String(`VVNFUkRBVEEgQkVHSU4KVXNpbmcgSU1BR0UgOiBlNGQ5M2EzNWM0ODIKVmFsaWRhdGluZyByb3V0ZTUzZG9tYWlucy51cy1lYXN0LTEuYW1hem9uYXdzLmNvbTo0NDMKVmFsaWRhdGluZyByZWdpc3RyeS5yZWRoYXQuaW86NDQzClZhbGlkYXRpbmcgcXVheS5pbzo0NDMKVmFsaWRhdGluZyBzc28ucmVkaGF0LmNvbTo0NDMKVmFsaWRhdGluZyBzc28ucmVkaGF0LmNvbTo4MApWYWxpZGF0aW5nIHB1bGwucTF3Mi5xdWF5LnJoY2xvdWQuY29tOjQ0MwpWYWxpZGF0aW5nIG9wZW5zaGlmdC5vcmc6NDQzClZhbGlkYXRpbmcgY29uc29sZS5yZWRoYXQuY29tOjQ0MwpWYWxpZGF0aW5nIGNvbnNvbGUucmVkaGF0LmNvbTo4MApWYWxpZGF0aW5nIHF1YXktcmVnaXN0cnkuczMuYW1hem9uYXdzLmNvbTo0NDMKVmFsaWRhdGluZyBjZXJ0LWFwaS5hY2Nlc3MucmVkaGF0LmNvbTo0NDMKVmFsaWRhdGluZyBhcGkuYWNjZXNzLnJlZGhhdC5jb206NDQzClZhbGlkYXRpbmcgaW5mb2d3LmFwaS5vcGVuc2hpZnQuY29tOjQ0MwpWYWxpZGF0aW5nIG1pcnJvci5vcGVuc2hpZnQuY29tOjQ0MwpWYWxpZGF0aW5nIHN0b3JhZ2UuZ29vZ2xlYXBpcy5jb206NDQzClZhbGlkYXRpbmcgYXBpLm9wZW5zaGlmdC5jb206NDQzClZhbGlkYXRpbmcgY2FydC1yaGNvcy1jaS5zMy5hbWF6b25hd3MuY29tOjQ0MwpWYWxpZGF0aW5nIHJlZ2lzdHJ5LmFjY2Vzcy5yZWRoYXQuY29tOjQ0MwpWYWxpZGF0aW5nIGNtLXF1YXktcHJvZHVjdGlvbi1zMy5zMy5hbWF6b25hd3MuY29tOjQ0MwpWYWxpZGF0aW5nIGVjMi5hbWF6b25hd3MuY29tOjQ0MwpWYWxpZGF0aW5nIGlhbS5hbWF6b25hd3MuY29tOjQ0MwpWYWxpZGF0aW5nIHJvdXRlNTMuYW1hem9uYXdzLmNvbTo0NDMKVmFsaWRhdGluZyBzdHMuYW1hem9uYXdzLmNvbTo0NDMKVmFsaWRhdGluZyBldmVudHMucGFnZXJkdXR5LmNvbTo0NDMKVmFsaWRhdGluZyBhcGkuZGVhZG1hbnNzbml0Y2guY29tOjQ0MwpWYWxpZGF0aW5nIG5vc25jaC5pbjo0NDMKVmFsaWRhdGluZyBpbnB1dHMxLm9zZHNlY3VyaXR5bG9ncy5zcGx1bmtjbG91ZC5jb206OTk5NwpWYWxpZGF0aW5nIGh0dHAtaW5wdXRzLW9zZHNlY3VyaXR5bG9ncy5zcGx1bmtjbG91ZC5jb206NDQzClZhbGlkYXRpbmcgb2JzZXJ2YXRvcml1bS5hcGkub3BlbnNoaWZ0LmNvbTo0NDMKVmFsaWRhdGluZyBlYzIuZXUtd2VzdC0xLmFtYXpvbmF3cy5jb206NDQzClZhbGlkYXRpbmcgZWxhc3RpY2xvYWRiYWxhbmNpbmcuZXUtd2VzdC0xLmFtYXpvbmF3cy5jb206NDQzClZhbGlkYXRpbmcgZXZlbnRzLmV1LXdlc3QtMS5hbWF6b25hd3MuY29tOjQ0MwpWYWxpZGF0aW5nIHRhZ2dpbmcudXMtZWFzdC0xLmFtYXpvbmF3cy5jb206NDQzClN1Y2Nlc3MhClVTRVJEQVRBIEVORAo=`),
	}
	FakeEC2Cli.EXPECT().GetConsoleOutput(gomock.Any(), gomock.Any()).Times(1).Return(out, nil)

	cli := AwsVerifier{
		AwsClient: &aws.Client{
			Region: "us-east-1",
		},
	}

	cli.AwsClient.SetClient(FakeEC2Cli)
	cli.Logger = &logging.GlogLogger{}

	err := cli.findUnreachableEndpoints(context.TODO(), "dummy-instance")
	if err != nil {
		t.Errorf("err should be nil when there's success in output, got: %v", err)
	}
}

func TestFindUnreachableEndpointsNoSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	FakeEC2Cli := mocks.NewMockEC2Client(ctrl)
	out := &ec2.GetConsoleOutputOutput{
		InstanceId: awss.String("dummy-instance"),
		// USERDATA BEGIN
		// Using IMAGE : e4d93a35c482
		// Validating route53domains.us-east-1.amazonaws.com:443
		// USERDATA END
		Output: awss.String(`VVNFUkRBVEEgQkVHSU4KVXNpbmcgSU1BR0UgOiBlNGQ5M2EzNWM0ODIKVmFsaWRhdGluZyByb3V0ZTUzZG9tYWlucy51cy1lYXN0LTEuYW1hem9uYXdzLmNvbTo0NDMKVVNFUkRBVEEgRU5ECg==`),
	}
	FakeEC2Cli.EXPECT().GetConsoleOutput(gomock.Any(), gomock.Any()).Times(1).Return(out, nil)

	cli := AwsVerifier{
		AwsClient: &aws.Client{
			Region: "us-east-1",
		},
	}

	cli.AwsClient.SetClient(FakeEC2Cli)
	cli.Logger = &logging.GlogLogger{}

	err := cli.findUnreachableEndpoints(context.TODO(), "dummy-instance")
	if err != nil {
		t.Errorf("Success! not found, but userdata end exists, err should be nil, got: %v", err)
	}

	if !cli.Output.IsSuccessful() {
		t.Errorf("Success! not found, userdata end exists but no regex match for failure, it means success, got : %v", cli.Output)
	}
}
