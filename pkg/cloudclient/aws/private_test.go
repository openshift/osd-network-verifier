package aws

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/cloudclient/mocks"
	"github.com/openshift/osd-network-verifier/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/golang/mock/gomock"
)

const err string = "error"
const exc string = "exception"
const fail string = "failure"

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
	out, err := cli.createEC2Instance(context.Background(), createEC2InstanceInput{
		amiID:         "test-ami",
		vpcSubnetID:   "test",
		instanceCount: 1,
	})
	if err != nil {
		t.Errorf("instance should be created")
	}

	if aws.ToString(out.Instances[0].InstanceId) != testID {
		t.Errorf("instance ID mismatch")
	}
}

func TestValidateEgress(t *testing.T) {
	testID := "aws-docs-example-instanceID"
	vpcSubnetID, cloudImageID := "dummy-id", "dummy-id"
	consoleOut := `[   48.062407] cloud-init[2472]: Cloud-init v. 19.3-44.amzn2 running 'modules:final' at Mon, 07 Feb 2022 12:30:22 +0000. Up 48.00 seconds.
	[   48.077429] cloud-init[2472]: USERDATA BEGIN
	[   48.138248] cloud-init[2472]: USERDATA END`

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	FakeEC2Cli := mocks.NewMockEC2Client(ctrl)

	FakeEC2Cli.EXPECT().RunInstances(gomock.Any(), gomock.Any()).Times(1).Return(&ec2.RunInstancesOutput{
		Instances: []types.Instance{{
			InstanceId: aws.String(testID),
		}},
	}, nil)

	FakeEC2Cli.EXPECT().DescribeInstanceStatus(gomock.Any(), gomock.Any()).Times(1).Return(&ec2.DescribeInstanceStatusOutput{
		InstanceStatuses: []types.InstanceStatus{{
			InstanceId: aws.String(testID),
			InstanceState: &types.InstanceState{
				Code: aws.Int32(16),
			},
		},
		},
	}, nil)

	encodedconsoleOut := base64.StdEncoding.EncodeToString([]byte(consoleOut))
	FakeEC2Cli.EXPECT().GetConsoleOutput(gomock.Any(), gomock.Any()).Times(1).Return(&ec2.GetConsoleOutputOutput{
		Output: aws.String(encodedconsoleOut),
	}, nil)

	FakeEC2Cli.EXPECT().TerminateInstances(gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)

	cli := Client{
		ec2Client: FakeEC2Cli,
		logger:    &logging.GlogLogger{},
	}

	if !cli.validateEgress(context.TODO(), vpcSubnetID, cloudImageID, "", time.Duration(1*time.Second)).IsSuccessful() {
		t.Errorf("validateEgress(): should pass")
	}
}

func TestValidateOutputErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testID := "aws-docs-example-instanceID"
	tests := []struct {
		name            string
		consoleOut      string
		vpcSubnetID     string
		cloudImageID    string
		testID          string
		expectError     error
		expectErrorType string
	}{
		{
			name: "testGenericError",
			consoleOut: `[   48.062407] cloud-init[2472]: Cloud-init v. 19.3-44.amzn2 running 'modules:final' at Mon, 
07 Feb 2022 12:30:22 +0000. Up 48.00 seconds.
	[   48.077429] cloud-init[2472]: USERDATA BEGIN
Could not do X.
	[   48.138248] cloud-init[2472]: USERDATA END`,
			vpcSubnetID:     "dummy-id",
			cloudImageID:    "dummy-id",
			testID:          "aws-docs-example-instanceID",
			expectError:     (errors.NewGenericError("")),
			expectErrorType: exc,
		},
		{
			name:         "testEgressError",
			vpcSubnetID:  "dummy-id",
			cloudImageID: "dummy-id",
			consoleOut: `[   48.062407] cloud-init[2472]: Cloud-init v. 19.3-44.amzn2 running 'modules:final' at Mon, 07 Feb 2022 12:30:22 +0000. Up 48.00 seconds.
	[   48.077429] cloud-init[2472]: USERDATA BEGIN
Unable to reach somesample.endpoint
	[   48.138248] cloud-init[2472]: USERDATA END`,
			testID:          "aws-docs-example-instanceID",
			expectError:     errors.NewEgressURLError(""),
			expectErrorType: fail,
		},
	}
	for _, test := range tests {

		encodedconsoleOut := base64.StdEncoding.EncodeToString([]byte(test.consoleOut))
		FakeEC2Cli := mocks.NewMockEC2Client(ctrl)
		FakeEC2Cli.EXPECT().GetConsoleOutput(gomock.Any(), gomock.Any()).Times(1).Return(&ec2.GetConsoleOutputOutput{
			Output: aws.String(encodedconsoleOut),
		}, nil)
		FakeEC2Cli.EXPECT().RunInstances(gomock.Any(), gomock.Any()).Times(1).Return(&ec2.RunInstancesOutput{
			Instances: []types.Instance{{
				InstanceId: aws.String(testID),
			}},
		}, nil)

		FakeEC2Cli.EXPECT().DescribeInstanceStatus(gomock.Any(), gomock.Any()).Times(1).Return(&ec2.DescribeInstanceStatusOutput{
			InstanceStatuses: []types.InstanceStatus{{
				InstanceId: aws.String(testID),
				InstanceState: &types.InstanceState{
					Code: aws.Int32(16),
				},
			},
			},
		}, nil)

		FakeEC2Cli.EXPECT().TerminateInstances(gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)
		cli := Client{
			ec2Client: FakeEC2Cli,
			logger:    &logging.GlogLogger{},
		}
		if cli.validateEgress(context.TODO(), test.vpcSubnetID, test.cloudImageID, "",
			time.Duration(1*time.Second)).IsSuccessful() {
			t.Errorf("failed %s: validateEgress(): should fail", test.name)
		}
		var allErrors []error
		switch test.expectErrorType {
		case fail:
			{
				allErrors, _, _ = cli.output.Parse()
			}
		case exc:
			{
				_, allErrors, _ = cli.output.Parse()
			}
		default:
			_, _, allErrors = cli.output.Parse()
		}

		assert.NotEmpty(t, allErrors)
		for _, e := range allErrors {
			assert.IsType(t, test.expectError, e, "Failed "+test.name)
		}

	}
}

	 
 
 

 