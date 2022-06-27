package main

import (
	"context"
	// "encoding/base64"
	// "errors"
	"fmt"
	// "regexp"
	"time"
	// "io"

	// "os"
	compute "cloud.google.com/go/compute/apiv1"
	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
	"google.golang.org/protobuf/proto"
	// "cloud.google.com/go/storage"
	"google.golang.org/api/option"
	// "golang.org/x/oauth2/google"
	
	// "golang.org/x/net/context"
	cli "google.golang.org/api/compute/v1"
	// "github.com/aws/aws-sdk-go-v2/aws"
	// "github.com/aws/aws-sdk-go-v2/config"
	// "github.com/aws/aws-sdk-go-v2/credentials"
	// "github.com/aws/aws-sdk-go-v2/service/ec2"
	// ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	// "github.com/aws/aws-sdk-go/aws/awserr"
	// ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	// "github.com/openshift/osd-network-verifier/pkg/helpers"
	// "github.com/openshift/osd-network-verifier/pkg/output"

	// handledErrors "github.com/openshift/osd-network-verifier/pkg/errors"
)

type createEC2InstanceInput struct {
	amiID         string
	vpcSubnetID   string
	userdata      string
	ebsKmsKeyID   string
	instanceCount int
}

//global variable ami image
var (
	instanceCount int = 1
	defaultAmi        = map[string]string{
		// using Amazon Linux 2 AMI (HVM) - Kernel 5.10
		"us-east1":      "rhel-9-v20220524",
		//other regions to add
	}
	// TODO find a location for future docker images
	networkValidatorImage string = "quay.io/app-sre/osd-network-verifier:v0.1.159-9a6e0eb"
	userdataEndVerifier   string = "USERDATA END"
)


func main() {
	ctx := context.Background()
			computeService, err := cli.NewService(ctx, option.WithAPIKey("1234..5532432"))
			if err != nil {
				fmt.Println( err)
			}

	projectID := "himanshub3"
	zone := "us-east1-b"
	instanceName := "instance-test-fedora"
	machineType := "e2-standard-2" //https://cloud.google.com/compute/docs/general-purpose-machines#e2-standard
	
	// image code reference doc https://cloud.google.com/compute/docs/reference/rest/v1/instances
	// https://cloud.google.com/compute/docs/images/os-details#red_hat_enterprise_linux_rhel - image list
	sourceImage := "projects/fedora-coreos-cloud/global/images/family/fedora-coreos-stable" //latest rhel image premium?
	networkName := "projects/" + projectID + "global/networks/hb-g7kzw-network"
	subnetworkName := "projects/" + projectID + "/regions/us-east1/subnetworks/hb-g7kzw-master-subnet"
	// tags := "https-server"

	
	fmt.Println("working!", projectID, zone, instanceName, machineType, sourceImage, networkName, computeService)

	instancesClient, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
			 fmt.Errorf("NewInstancesRESTClient: %v", err)
	}
	defer instancesClient.Close()

	req := &computepb.InsertInstanceRequest{
			Project: projectID,
			Zone:    zone,
			InstanceResource: &computepb.Instance{
					Name: proto.String(instanceName),
					//How to add network tag ? http/https?
					// Tags: &computepb.Tags{ 
					// 	items: [
					// 		"https-server",
					// 	],
					// },
					Disks: []*computepb.AttachedDisk{
							{
									InitializeParams: &computepb.AttachedDiskInitializeParams{
											DiskSizeGb:  proto.Int64(10),
											SourceImage: proto.String(sourceImage),
									},
									AutoDelete: proto.Bool(true),
									Boot:       proto.Bool(true),
									Type:       proto.String(computepb.AttachedDisk_PERSISTENT.String()),
							},
					},
					MachineType: proto.String(fmt.Sprintf("zones/%s/machineTypes/%s", zone, machineType)),
					NetworkInterfaces: []*computepb.NetworkInterface{
							{
									Name: proto.String(networkName),
									Subnetwork: proto.String(subnetworkName),
							},
					},
			},
	}

	op, err := instancesClient.Insert(ctx, req)
	if err != nil {
			 fmt.Errorf("unable to create instance: %v", err)
	}

	if err = op.Wait(ctx); err != nil {
			fmt.Errorf("unable to wait for the operation: %v", err)
	}

	fmt.Println( "Instance created\n")


	//stop instance after 120 seeconds
	time.Sleep(120 * time.Second)
	defer instancesClient.Close()

	reqs := &computepb.StopInstanceRequest{
			Project:  projectID,
			Zone:     zone,
			Instance: instanceName,
	}

	sp, err := instancesClient.Stop(ctx, reqs)
	if err != nil {
			fmt.Errorf("unable to stop instance: %v", err)
	}

	if err = op.Wait(ctx); err != nil {
			fmt.Errorf("unable to wait for the operation: %v", err)
	}

	fmt.Println( sp, "Instance stopped\n")
}