package gcp

import (
	"context"
	// "encoding/base64"
	// "errors"
	"fmt"
	// "regexp"
	"time"
	// "io"

	compute "cloud.google.com/go/compute/apiv1"
	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
	"google.golang.org/protobuf/proto"
	"os"
	// "cloud.google.com/go/storage"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"

	// "golang.org/x/net/context"
	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/helpers"
	"github.com/openshift/osd-network-verifier/pkg/output"
	//cli
	computev1 "google.golang.org/api/compute/v1"
	// "io/ioutil"
	// "io/ioutil"
	"path/filepath"
	// "reflect"
	// "encoding/base64"
)

/*	// "github.com/aws/aws-sdk-go-v2/aws"
	// "github.com/aws/aws-sdk-go-v2/config"
	// "github.com/aws/aws-sdk-go-v2/credentials"
	// "github.com/aws/aws-sdk-go-v2/service/ec2"
	// ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	// "github.com/aws/aws-sdk-go/aws/awserr"
	//


	// handledErrors "github.com/openshift/osd-network-verifier/pkg/errors"
)
*/
// type createE2InstanceInput struct {
// 	amiID         string
// 	vpcSubnetID   string
// 	userdata      string
// 	ebsKmsKeyID   string

// }

// //global variable ami image, gcp has region, zone

var (
	defaultAmi = map[string]string{
		// using Amazon Linux 2 AMI (HVM) - Kernel 5.10
		"us-east1": "cos-97-lts",
		//other regions to add
	}
	// TODO find a location for future docker images
	// networkValidatorImage string = "quay.io/app-sre/osd-network-verifier:v0.1.159-9a6e0eb"
	// userdataEndVerifier   string = "USERDATA END"
)

//build labels/tags method TODO

//newClient method
func newClient(ctx context.Context, logger ocmlog.Logger, credentials *google.Credentials, region, instanceType string, tags map[string]string) (*Client, error) {
	//use oauth2 token in credentials struct to create client, JSON optional
	// https://pkg.go.dev/golang.org/x/oauth2/google#Credentials
	//env var has path to json file
	absPath, err := filepath.Abs(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
	// data, err := ioutil.ReadFile(absPath)
	// cred, err := google.CredentialsFromJSON(
	// 	ctx, data,
	// 	computev1.ComputeScope)
	if err != nil {
		return nil, err
	}
	computeService, err := computev1.NewService(ctx, option.WithCredentialsFile(absPath))
	if err != nil {
		return nil, err
	}
	// fmt.Println("working!", region, instanceType, computeService)
	// time.Sleep(30 * time.Second)

	return &Client{
		projectID:      credentials.ProjectID,
		region:         region,
		instanceType:   instanceType,
		computeService: computeService,
		tags:           tags,
		logger:         logger,
		output:         output.Output{},
	}, nil

	//ToDo
	//call validate instance type then return
}



//ToDo func findUnreachableEndpoints
func (c *Client) findUnreachableEndpoints(ctx context.Context, projectID string, instanceName string, zone string) error {
	// Compile the regular expressions once
	// reVerify := regexp.MustCompile(userdataEndVerifier)
	// reUnreachableErrors := regexp.MustCompile(`Unable to reach (\S+)`)

	// latest := true

	// getConsoleOutput then parse, use c.output to store result of the execution
	err := helpers.PollImmediate(30*time.Second, 30*time.Second, func() (bool, error) {
		output, err := c.computeService.Instances.GetSerialPortOutput(projectID, zone, instanceName).Context(ctx).Do()
		if err != nil {
			return false, err
		}
		fmt.Println(output)
		// if output != nil {
		// 	// First, gather the ec2 console output
		// 	scriptOutput := output
		// 	if err != nil {
		// 		// unable to decode output. we will try again
		// 		c.logger.Debug(ctx, "Error while collecting console output, will retry on next check interval: %s", err)
		// 		return false, nil
		// 	}

		// 	// In the early stages, an ec2 instance may be running but the console is not populated with any data, retry if that is the case
		// 	if len(scriptOutput) < 1 {
		// 		c.logger.Debug(ctx, "EC2 console output not yet populated with data, continuing to wait...")
		// 		return false, nil
		// 	}

		// 	// Check for the specific string we output in the generated userdata file at the end to verify the userdata script has run
		// 	// It is possible we get EC2 console output, but the userdata script has not yet completed.
		// 	verifyMatch := reVerify.FindString(string(scriptOutput))
		// 	if len(verifyMatch) < 1 {
		// 		c.logger.Debug(ctx, "EC2 console output contains data, but end of userdata script not seen, continuing to wait...")
		// 		return false, nil
		// 	}

		// 	// check output failures, report as exception if they occurred
		// 	// var rgx = regexp.MustCompile(`(?m)^(.*Cannot.*)|(.*Could not.*)|(.*Failed.*)|(.*command not found.*)`)
		// 	// notFoundMatch := rgx.FindAllStringSubmatch(string(scriptOutput), -1)
		// 	// if len(notFoundMatch) > 0 {
		// 	// 	c.output.AddException(handledErrors.NewEgressURLError("internet connectivity problem: please ensure there's internet access in given vpc subnets"))
		// 	// }

		// 	// If debug logging is enabled, output the full console log that appears to include the full userdata run
		// 	c.logger.Debug(ctx, "Full EC2 console output:\n---\n%s\n---", scriptOutput)

		// 	c.output.SetFailures(reUnreachableErrors.FindAllString(string(scriptOutput), -1))
		// 	return true, nil
		// }
		// c.logger.Debug(ctx, "Waiting for UserData script to complete...")
		return false, nil
	})

	return err
}

// terminateE2Instance terminates target ec2 instance
// uses c.output to store result of the execution
func (c *Client) terminateE2Instance(ctx context.Context, projectID string, instanceName string, zone string) {
	c.logger.Info(ctx, "Terminating ec2 instance with id %s", instanceName)

	sp, err := c.computeService.Instances.Stop(projectID, zone, instanceName).Context(ctx).Do()
	if err != nil {
		fmt.Errorf("unable to stop instance: %v", err, sp)
	}

	fmt.Println(instanceName, " Instance stopped\n")

	c.output.AddError(err)
	// return &c.output

}

func (c *Client) setCloudImage(cloudImageID string) (string, error) {
	// If a cloud image wasn't provided by the caller,
	if cloudImageID == "" {
		// use defaultAmi for the region instead
		// cloudImageID = defaultAmi[c.region]
		cloudImageID = defaultAmi["us-east1"]
		if cloudImageID == "" {
			return "", fmt.Errorf("no default ami found for region %s ", c.region)
		}
	}

	return cloudImageID, nil
}

// sudo docker run --env "AWS_REGION=us-east-2" -e "START_VERIFIER=VALIDATOR START" -e "END_VERIFIER=VALIDATOR END" ${VALIDATOR_IMAGE} --timeout="2s"  >> ~/userdata-output || echo "Failed to successfully run the docker container"
// docker run --env "AWS_REGION=us-east-2" -e "START_VERIFIER=VALIDATOR START" -e "END_VERIFIER=VALIDATOR END" ${VALIDATOR_IMAGE} --timeout="error"  >> /var/log/userdata-output || echo "Failed to successfully run the docker container"

//creates and terminates a VMwith startup script
func (c *Client) validateEgress(ctx context.Context, vpcSubnetID, cloudImageID string, kmsKeyID string, timeout time.Duration) *output.Output {
	c.logger.Debug(ctx, "Using configured timeout of %s for each egress request", timeout.String())
	fmt.Println("using subnet ", vpcSubnetID)

	// userDataVariables := map[string]string{
	// 	"AWS_REGION":               "us-east-2",
	// 	"USERDATA_BEGIN":           "USERDATA BEGIN",
	// 	"USERDATA_END":             userdataEndVerifier,
	// 	"VALIDATOR_START_VERIFIER": "VALIDATOR START",
	// 	"VALIDATOR_END_VERIFIER":   "VALIDATOR END",
	// 	"VALIDATOR_IMAGE":          networkValidatorImage,
	// 	"TIMEOUT":                  timeout.String(),
	// }
	var er error
	cloudImageID, er = c.setCloudImage(cloudImageID)
	if er != nil {
		return c.output.AddError(er) // fatal
	}

	// if err != nil {
	// 	fmt.Println(err)
	// }
	projectID := "himanshub3"
	zone := "us-east1-b"
	instanceName := "works-joe"
	machineType := "e2-standard-2" //https://cloud.google.com/compute/docs/general-purpose-machines#e2-standard

	// image code reference doc https://cloud.google.com/compute/docs/reference/rest/v1/instances
	// https://cloud.google.com/compute/docs/images/os-details#red_hat_enterprise_linux_rhel - image list
	
	sourceImage := "projects/cos-cloud/global/images/family/cos-97-lts" 
	//container opt images https://cloud.google.com/compute/docs/containers#container_images
	networkName := fmt.Sprintf("projects/%s/global/networks/hb-gcp-test-lzncg-network", projectID)
	subnetworkName := fmt.Sprintf("projects/%s/regions/us-east1/subnetworks/%s", projectID, vpcSubnetID)


	fmt.Println("working!", projectID, zone, instanceName, machineType, sourceImage, networkName)

	var instancesClient, err = compute.NewInstancesRESTClient(ctx)
	// fmt.Println(instancesClient)
	// fmt.Println(reflect.TypeOf(instancesClient))
	// time.Sleep(120 * time.Second)
	if err != nil {
		fmt.Errorf("NewInstancesRESTClient: %v", err)
	}
	defer instancesClient.Close()

	req := &computepb.InsertInstanceRequest{
		Project: projectID,
		Zone:    zone,
		InstanceResource: &computepb.Instance{
			Name: proto.String(instanceName),

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
			// can call docker using startup script or pass cloud-init script to user-data
			Metadata: &computepb.Metadata{
				Items: []*computepb.Items{
					// {
					// 	Key: proto.String("startup-script"),
					// 	Value: proto.String("#!/bin/bash\n" +
					// 		"sudo mkdir  ~/../home/test; sudo docker run --env 'AWS_REGION=us-east-2' -e 'START_VERIFIER=VALIDATOR START' -e 'END_VERIFIER=VALIDATOR END' 'quay.io/app-sre/osd-network-verifier:v0.1.159-9a6e0eb' --timeout='2s'  >> ~/../home/test/userdata-output || echo 'Failed to successfully run the docker container';\n" +
					// 		"cat ~/../home/test/userdata-output;"),
					// },
					// //How to pass gcpuserdata.yaml
					{
						Key: proto.String("user-data"),
						Value: proto.String("#cloud-config\n" +
							"repo_update: true\n" +
							"package_update: true\n" +

							"packages:\n" +
							"  - docker\n" +

							"runcmd:\n" +
							"  - sudo service docker start\n" +
							"  - sudo mkdir  ~/../home/test\n" +
							"  - echo 'VALIDATOR START' >> ~/../home/test/userdata-output\n" +
							"  # - sudo docker pull ${VALIDATOR_IMAGE}\n" +
							"  # Use `|| true` to ignore failure exit codes, we want the script to continue either way\n" +
							"  - sudo docker run --env 'AWS_REGION=us-east-2' -e 'START_VERIFIER=VALIDATOR START' -e 'END_VERIFIER=VALIDATOR END' 'quay.io/app-sre/osd-network-verifier:v0.1.159-9a6e0eb' --timeout='2s'  >> ~/../home/test/userdata-output || echo 'Failed to successfully run the docker container'\n" +
							"  - echo 'VALIDATOR END' >> ~/../home/test/userdata-output\n" +
							"  - cat ~/../home/test/userdata-output #>/dev/console"),
					},
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

	fmt.Println("Instance created\n")

	//wait for instance to start - 40 seeconds
	time.Sleep(40 * time.Second)
	defer instancesClient.Close()

	c.findUnreachableEndpoints(ctx, projectID, instanceName, zone)

	c.terminateE2Instance(ctx, projectID, instanceName, zone)

	return &c.output
}
