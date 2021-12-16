package gcp

import (
	"context"
	"testing"

	gomock "github.com/golang/mock/gomock"
	mock_cloudclient "github.com/openshift/osd-network-verifier/pkg/cloudclient/mock_cloudclient"

	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"golang.org/x/oauth2/google"
)

func TestByoVPCValidator(t *testing.T) {
	ctx := context.TODO()
	logger := &ocmlog.StdLogger{}
	client := &Client{logger: logger}
	err := client.ByoVPCValidator(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateEgress(t *testing.T) {
	ctx := context.TODO()
	subnetID := "subnet-id"
	cloudImageID := "image-id"
	mock := mock_cloudclient.NewMockCloudClient(gomock.NewController(t))
	mock.EXPECT().ValidateEgress(ctx, subnetID, cloudImageID).Return(nil)
	err := mock.ValidateEgress(ctx, subnetID, cloudImageID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewClient(t *testing.T) {
	ctx := context.TODO()
	logger := &ocmlog.StdLogger{}
	credentials := &google.Credentials{ProjectID: "my-sample-project-191923"}
	region := "superstable-region1-z"
	tags := map[string]string{"osd-network-verifier": "owned"}
	client, err := NewClient(ctx, logger, credentials, region, tags)
	if err != nil {
		t.Errorf("unexpected error creating client: %v", err)
	}
	if client.projectID != credentials.ProjectID {
		t.Errorf("unexpected project ID: %v", client.projectID)
	}
	if client.region != region {
		t.Errorf("unexpected region: %v", client.region)
	}
	if client.tags["osd-network-verifier"] != "owned" {
		t.Errorf("unexpected tags: %v", client.tags)
	}
}
