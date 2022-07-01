package gcp

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
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"

	// "golang.org/x/net/context"
	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/output"
	//cli
	computev1 "google.golang.org/api/compute/v1"

	// "io/ioutil"
	"path/filepath"
)

/*	// "github.com/aws/aws-sdk-go-v2/aws"
	// "github.com/aws/aws-sdk-go-v2/config"
	// "github.com/aws/aws-sdk-go-v2/credentials"
	// "github.com/aws/aws-sdk-go-v2/service/ec2"
	// ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	// "github.com/aws/aws-sdk-go/aws/awserr"
	//
	// "github.com/openshift/osd-network-verifier/pkg/helpers"

	// handledErrors "github.com/openshift/osd-network-verifier/pkg/errors"
)

type createE2InstanceInput struct {
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
		"us-east1": "rhel-9-v20220524",
		//other regions to add
	}
	// TODO find a location for future docker images
	networkValidatorImage string = "quay.io/app-sre/osd-network-verifier:v0.1.159-9a6e0eb"
	userdataEndVerifier   string = "USERDATA END"
)

//build labels/tags method TODO
*/

//newClient method
func newClient(ctx context.Context, logger ocmlog.Logger, credentials *google.Credentials, region, instanceType string, tags map[string]string) (*Client, error) {
	absPath, _ := filepath.Abs("./himanshub3-gcp-new.json")
	// data, err := ioutil.ReadFile(absPath)
	// cred, err := google.CredentialsFromJSON(
	// 	ctx, data,
	// 	computev1.ComputeScope)
	// if err != nil {
	// 	return nil, err
	// }
	computeService, err := computev1.NewService(ctx, option.WithCredentialsFile(absPath))
	if err != nil {
		return nil, err
	}
	// fmt.Println("working!", region, instanceType, computeService)

	return &Client{
		projectID:      credentials.ProjectID,
		region:         region,
		instanceType:   instanceType,
		computeService: computeService,
		tags:           tags,
		logger:         logger,
	}, nil

	//ToDo
	//validate instance type then return
}

//creates and terminates a VMwith startup script
func (c *Client) validateEgress(ctx context.Context, vpcSubnetID, cloudImageID string, kmsKeyID string, timeout time.Duration) *output.Output {
	fmt.Println("using subnet ", vpcSubnetID)
	// ctx := context.Background()

	// if err != nil {
	// 	fmt.Println(err)
	// }
	projectID := "himanshub3"
	zone := "us-east1-b"
	instanceName := "instance-test-cool"
	machineType := "e2-standard-2" //https://cloud.google.com/compute/docs/general-purpose-machines#e2-standard

	// image code reference doc https://cloud.google.com/compute/docs/reference/rest/v1/instances
	// https://cloud.google.com/compute/docs/images/os-details#red_hat_enterprise_linux_rhel - image list
	sourceImage := "projects/fedora-coreos-cloud/global/images/family/fedora-coreos-stable" //latest rhel image premium?
	networkName := "projects/" + projectID + "global/networks/hb-g7kzw-network"
	subnetworkName := "projects/" + projectID + "/regions/us-east1/subnetworks/" + vpcSubnetID
	// tags := "https-server"

	fmt.Println("working!", projectID, zone, instanceName, machineType, sourceImage, networkName)

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
			//How to add network tag ? http/https? -- Not needed yet
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
					Name:       proto.String(networkName),
					Subnetwork: proto.String(subnetworkName),
				},
			},
			//Startup script
			// Metadata: &computepb.Metadata{
			// 	Items: []*computepb.Items{
			// 		{
			// 			Key: proto.String("hello"),
			// 			Value: proto.String("world"),
			// 		},
			// 		{
			// 			Key: proto.String("startup-script"),
			// 			Value: proto.String("#! /bin/bash apt-get"),
			// 		},
			// 	},
			// },
		},
	}

	op, err := instancesClient.Insert(ctx, req)
	if err != nil {
		fmt.Errorf("unable to create instance: %v", err)
	}

	if err = op.Wait(ctx); err != nil {
		fmt.Errorf("unable to wait for the operation: %v", err)
	}

	fmt.Println("Instance created\n")

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

	fmt.Println(sp, "Instance stopped\n")

	return &c.output
}

/*
func newClient() {
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
*/
