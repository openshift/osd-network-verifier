package gcp

//tests for ValidateEgress, NewClient have been skipped because it calls gcp api
import (
	"context"
	"testing"
	"time"

	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/proxy"
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
	t.Skip("Skipping testing for ValidateEgress as it calls gcp api")
	ctx := context.TODO()
	subnetID := "subnet-id"
	cloudImageID := "image-id"
	cli := Client{}
	timeout := 1 * time.Second
	if !cli.ValidateEgress(ctx, subnetID, cloudImageID, "", "", timeout, proxy.ProxyConfig{}).IsSuccessful() {
		t.Errorf("validation should have been successful")
	}
}

func TestNewClient(t *testing.T) {
	t.Skip("Skipping testing for NewClient as it calls gcp api")
	ctx := context.TODO()
	logger := &ocmlog.StdLogger{}
	credentials := &google.Credentials{ProjectID: "my-sample-project-191923"}
	region := "superstable-region1-z"
	instanceType := "test-instance"
	tags := map[string]string{"osd-network-verifier": "owned"}
	client, err := NewClient(ctx, logger, credentials, region, instanceType, tags)
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
