package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/cloudclient/mocks"

	"github.com/golang/mock/gomock"
)

func TestCreateEC2Instance(t *testing.T) {
	testID := "aws-docs-example-instanceID"
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	FakeEC2Cli := mocks.NewMockEC2Client(ctrl)

	FakeEC2Cli.EXPECT().RunInstances(gomock.Any(), gomock.Any()).Times(1).Return(&ec2.RunInstancesOutput{
		Instances: []types.Instance{{
			InstanceId: aws.String(testID),
		}},
	}, nil)

	cli := Client{
		ec2Client: FakeEC2Cli,
		logger:    &logging.GlogLogger{},
	}
	out, err := cli.createEC2Instance(context.Background(), "test-ami", 1, "", "test", map[string]string{})
	if err == nil {
		t.Errorf("instance should be created")
	}

	if aws.ToString(out.Instances[0].InstanceId) != testID {
		t.Errorf("instance ID mismatch")
	}
}
