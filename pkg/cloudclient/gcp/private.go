package gcp
//Features to add - image-id, kms-key-id
import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"time"
	"os"
	"path/filepath"

	compute "cloud.google.com/go/compute/apiv1"
	computev1 "google.golang.org/api/compute/v1"
	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
	"google.golang.org/protobuf/proto"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"

	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/helpers"
	"github.com/openshift/osd-network-verifier/pkg/output"
	handledErrors "github.com/openshift/osd-network-verifier/pkg/errors"
)

type createE2InstanceInput struct {
	amiID       string
	vpcSubnetID string
	userdata    string
	// ebsKmsKeyID   string
	zone         string
	machineType  string
	instanceName string
	sourceImage  string
	networkName  string
}

// //global variable ami image, gcp has region, zone

var (
	defaultAmi = map[string]string{
		// using google container optimized image
		"default": "cos-97-lts",
		//other regions to add
	}
	// TODO find a location for future docker images
	networkValidatorImage string = "quay.io/app-sre/osd-network-verifier:v0.1.159-9a6e0eb"
	userdataEndVerifier   string = "USERDATA END"
)

//newClient method
func newClient(ctx context.Context, logger ocmlog.Logger, credentials *google.Credentials, region, instanceType string, tags map[string]string) (*Client, error) {
	//use oauth2 token in credentials struct to create client,
	// https://pkg.go.dev/golang.org/x/oauth2/google#Credentials
	//env var has path to json file
	absPath, err := filepath.Abs(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))

	if err != nil {
		return nil, err
	}
	computeService, err := computev1.NewService(ctx, option.WithCredentialsFile(absPath))
	if err != nil {
		return nil, err
	}

	c := &Client{
		projectID:      credentials.ProjectID,
		region:         region,
		zone:           fmt.Sprintf("%s-b", region), //append zone b
		instanceType:   instanceType,
		computeService: computeService,
		tags:           tags,
		logger:         logger,
		output:         output.Output{},
	}

	if err := c.validateInstanceType(ctx); err != nil {
		return nil, fmt.Errorf("Instance type %s is invalid: %s", c.instanceType, err)
	}

	return c, nil
}

func (c *Client) validateInstanceType(ctx context.Context) error {
	//  machineTypes List https://cloud.google.com/compute/docs/reference/rest/v1/machineTypes/list

	c.logger.Debug(ctx, "Gathering description of instance type %s from EC2", c.instanceType)

	descOut := c.computeService.MachineTypes.List(c.projectID, c.zone)

	//check if instanceType is in the list
	found := false
	if err := descOut.Pages(ctx, func(page *computev1.MachineTypeList) error {
		for _, machineType := range page.Items {
			if string(machineType.Name) == c.instanceType {
				found = true
				c.logger.Debug(ctx, "Instance type %s supported", c.instanceType)
				break
			}
		}
		c.logger.Debug(ctx, "Fully describe instance types output contains %d instance types", len(page.Items))
		return nil
	}); err != nil {
		return fmt.Errorf("Unable to gather list of supported instance types from E2: %s", err)
	}

	if !found {
		return fmt.Errorf("Instance type %s not found in E2 API", c.instanceType)
	}

	return nil
}

//ToDo func createE2Instance
func (c *Client) createE2Instance(ctx context.Context, input createE2InstanceInput) (createE2InstanceInput, error) {

	instancesClient, err := compute.NewInstancesRESTClient(ctx)

	if err != nil {
		fmt.Errorf("NewInstancesRESTClient: %v", err)
	}
	// defer instancesClient.Close()

	req := &computepb.InsertInstanceRequest{
		Project: c.projectID,
		Zone:    c.zone,
		InstanceResource: &computepb.Instance{
			Name: proto.String(input.instanceName),
			// Tags: &computepb.Tags{
			// 	Items: []string{"http-server", "https-server"},
			// },
			Disks: []*computepb.AttachedDisk{
				{
					InitializeParams: &computepb.AttachedDiskInitializeParams{
						DiskSizeGb:  proto.Int64(10),
						SourceImage: proto.String(input.sourceImage),
					},
					AutoDelete: proto.Bool(true),
					Boot:       proto.Bool(true),
					Type:       proto.String(computepb.AttachedDisk_PERSISTENT.String()),
				},
			},
			MachineType: proto.String(fmt.Sprintf("zones/%s/machineTypes/%s", c.zone, input.machineType)),
			NetworkInterfaces: []*computepb.NetworkInterface{
				{
					Name:       proto.String(input.networkName),
					Subnetwork: proto.String(input.vpcSubnetID),
				},
			},
			//pass gcpuserdata.yaml cloud-init script
			Metadata: &computepb.Metadata{
				Items: []*computepb.Items{
					//can pass startup script
					// {
					// 	Key: proto.String("startup-script"),
					// 	Value: proto.String("#!/bin/bash\n"),
					// },

					//pass gcpuserdata.yaml
					{
						Key:   proto.String("user-data"),
						Value: proto.String(input.userdata),
					},
				},
			},
		},
	}

	instanceResp, err := instancesClient.Insert(ctx, req)
	if err != nil {
		fmt.Errorf("unable to create instance: %v %v", err, instanceResp)
	}

	c.logger.Info(ctx, "Created instance with ID: %s", input.instanceName)

	//get fingerprint from instance
	inst, err := c.computeService.Instances.Get(c.projectID, c.zone, input.instanceName).Do()
	if err != nil {
		fmt.Errorf("Failed to get fingerprint to apply tags to instance", err)
	}

	//Add tags - known as labels in gcp
	c.logger.Info(ctx, "Applying labels")

	reqbody := &computev1.InstancesSetLabelsRequest{
		LabelFingerprint: inst.LabelFingerprint,
		Labels:           c.tags,
	}

	//send request to apply tags
	resp, err := c.computeService.Instances.SetLabels(c.projectID, c.zone, input.instanceName, reqbody).Context(ctx).Do()
	if err != nil {
		fmt.Errorf("unable to create label: %v", err)
		fmt.Printf("error occured when creating labels ", err)
	}

	if resp != nil {
		c.logger.Info(ctx, "Successfully applied labels ")
	}

	return input, nil

}

//ToDo func describeE2Instances - check status code meaning and return
// Returns instance state
func (c *Client) describeE2Instances(ctx context.Context, instanceName string) (string, error) {
	// States
	//PROVISIONING, STAGING, RUNNING, STOPPING, STOPPED, TERMINATED, SUSPENDED
	// https://cloud.google.com/compute/docs/instances/instance-life-cycle

	//Error Codes https://cloud.google.com/apis/design/errors

	resp, err := c.computeService.Instances.Get(c.projectID, c.zone, instanceName).Context(ctx).Do()
	if err != nil {
		c.logger.Error(ctx, "Errors while describing the instance status: %s", err.Error())
		return "PERMISSION DENIED", err
	}

	// Get status of vm
	status := resp.Status
	if len(status) < 1 {
		fmt.Errorf("Errors while describing the instance status: %v", err.Error())
	}
	switch status {
	case "PROVISIONING", "STAGING":
		c.logger.Debug(ctx, "Waiting on VM operation: ", status)

	case "STOPPING", "STOPPED", "TERMINATED", "SUSPENDED":
		c.logger.Debug(ctx, "Fatal - Instance status: ", instanceName)
		return "FATAL", fmt.Errorf(status)
	}

	if len(status) == 0 {
		c.logger.Debug(ctx, "Instance %s has no status yet", instanceName)
	}
	return status, nil
}

//ToDo func waitForEC2InstanceCompletion - check for timeout
func (c *Client) waitForE2InstanceCompletion(ctx context.Context, instanceName string) error {
	//wait for the instance to run
	err := helpers.PollImmediate(5*time.Second, 2*time.Minute, func() (bool, error) {
		code, descError := c.describeE2Instances(ctx, instanceName)
		switch code {
		case "RUNNING":
			//instance is running, break
			c.logger.Info(ctx, "E2 Instance: %s %s", instanceName, code)
			return true, nil

		case "FATAL":
			return false, fmt.Errorf("Instance %s already exists with %s state. Please run again", instanceName, descError)

		case "PERMISSION DENIED":
			return false, fmt.Errorf("missing required permissions for account: %s", descError)
		}

		if descError != nil {
			return false, descError // unhandled
		}

		return false, nil // continue loop
	})

	return err
}

//ToDo func generateUserData - helpers.usersdatatemplateGcp
func generateUserData(variables map[string]string) (string, error) {
	variableMapper := func(varName string) string {
		return variables[varName]
	}
	data := os.Expand(helpers.UserdataTemplateGcp, variableMapper)

	return data, nil
}

//ToDo func findUnreachableEndpoints
func (c *Client) findUnreachableEndpoints(ctx context.Context, instanceName string) error {
	// Compile the regular expressions once
	reVerify := regexp.MustCompile(userdataEndVerifier)
	reUnreachableErrors := regexp.MustCompile(`Unable to reach (\S+)`)

	// getConsoleOutput then parse, use c.output to store result of the execution
	err := helpers.PollImmediate(30*time.Second, 4*time.Minute, func() (bool, error) {
		output, err := c.computeService.Instances.GetSerialPortOutput(c.projectID, c.zone, instanceName).Context(ctx).Do()
		if err != nil {
			return false, err
		}
		// fmt.Println(output)
		if output != nil {
			// First, gather the e2 console output
			scriptOutput := fmt.Sprintf("%#v", output)

			if err != nil {
				// unable to decode output. we will try again
				c.logger.Debug(ctx, "Error while collecting console output, will retry on next check interval: %s", err)
				return false, nil
			}

			// In the early stages, an e2 instance may be running but the console is not populated with any data, retry if that is the case
			if len(scriptOutput) < 1 {
				c.logger.Debug(ctx, "E2 console output not yet populated with data, continuing to wait...")
				return false, nil
			}

			// Check for the specific string we output in the generated userdata file at the end to verify the userdata script has run
			// It is possible we get EC2 console output, but the userdata script has not yet completed.
			verifyMatch := reVerify.FindString(string(scriptOutput))
			if len(verifyMatch) < 1 {
				c.logger.Debug(ctx, "E2 console output contains data, but end of userdata script not seen, continuing to wait...")
				return false, nil
			}

			// check output failures, report as exception if they occurred
			var rgx = regexp.MustCompile(`(?m)^(.*Cannot.*)|(.*Could not.*)|(.*Failed.*)|(.*command not found.*)`)
			notFoundMatch := rgx.FindAllStringSubmatch(string(scriptOutput), -1)

			if len(notFoundMatch) > 0 { //&& len(success) < 1
				c.output.AddException(handledErrors.NewEgressURLError("internet connectivity problem: please ensure there's internet access in given vpc subnets"))
			}

			// If debug logging is enabled, output the full console log that appears to include the full userdata run
			c.logger.Debug(ctx, "Full E2 console output:\n---\n%s\n---", scriptOutput)

			c.output.SetEgressFailures(reUnreachableErrors.FindAllString(string(scriptOutput), -1))
			return true, nil
		}
		c.logger.Debug(ctx, "Waiting for UserData script to complete...")
		return false, nil
	})

	return err
}

// terminateE2Instance terminates target ec2 instance
// uses c.output to store result of the execution
func (c *Client) terminateE2Instance(ctx context.Context, instanceName string) {
	c.logger.Info(ctx, "Terminating e2 instance with id %s", instanceName)

	_, err := c.computeService.Instances.Stop(c.projectID, c.zone, instanceName).Context(ctx).Do()
	if err != nil {
		fmt.Errorf("Unable to terminate instance: %s", err)
	}

	c.output.AddError(err)

}

func (c *Client) setCloudImage(cloudImageID string) (string, error) {
	// If a cloud image wasn't provided by the caller,
	// if cloudImageID == "" {
	// use defaultAmi for the region instead
	// cloudImageID = defaultAmi[c.region]
	cloudImageID = defaultAmi["default"]
	if cloudImageID == "" {
		return "", fmt.Errorf("no default ami found for region %s ", c.region)
	}
	// }

	return cloudImageID, nil
}

// validateEgress performs validation process for egress
// Basic workflow is:
// - prepare for e2 instance creation
// - create instance and wait till it gets ready, wait for gcpUserData script execution
// - find unreachable endpoints & parse output, then terminate instance
// - return `c.output` which stores the execution results
func (c *Client) validateEgress(ctx context.Context, vpcSubnetID, cloudImageID string, kmsKeyID string, timeout time.Duration) *output.Output {
	c.logger.Debug(ctx, "Using configured timeout of %s for each egress request", timeout.String())

	userDataVariables := map[string]string{
		"AWS_REGION":               "us-east-2",
		"USERDATA_BEGIN":           "USERDATA BEGIN",
		"USERDATA_END":             userdataEndVerifier,
		"VALIDATOR_START_VERIFIER": "VALIDATOR START",
		"VALIDATOR_END_VERIFIER":   "VALIDATOR END",
		"VALIDATOR_IMAGE":          networkValidatorImage,
		"TIMEOUT":                  timeout.String(),
	}

	userData, err := generateUserData(userDataVariables)
	if err != nil {
		return c.output.AddError(err)
	}
	c.logger.Debug(ctx, "Generated userdata script:\n---\n%s\n---", userData)
	// time.Sleep(40 * time.Second)

	cloudImageID, err = c.setCloudImage(cloudImageID)
	if err != nil {
		return c.output.AddError(err) // fatal
	}

	//for random name
	rand.Seed(time.Now().UnixNano())

	//image list https://cloud.google.com/compute/docs/images/os-details#red_hat_enterprise_linux_rhel

	//container images https://cloud.google.com/compute/docs/containers#container_images

	//sourceImage := "projects/fedora-coreos-cloud/global/images/family/fedora-coreos-stable"

	instance, err := c.createE2Instance(ctx, createE2InstanceInput{
		vpcSubnetID:  fmt.Sprintf("projects/%s/regions/%s/subnetworks/%s", c.projectID, c.region, vpcSubnetID),
		userdata:     userData,
		zone:         c.zone, //Note: gcp zone format is us-east1-b
		machineType:  c.instanceType,
		instanceName: fmt.Sprintf("verifier-%v", rand.Intn(1000)),
		sourceImage:  fmt.Sprintf("projects/cos-cloud/global/images/family/%s", cloudImageID),
		networkName:  fmt.Sprintf("projects/%s/global/networks/%s", c.projectID, os.Getenv("GCP_VPC_NAME")),

		// ebsKmsKeyID:   kmsKeyID,

	})
	if err != nil {
		return c.output.AddError(err) // fatal
	}

	c.logger.Debug(ctx, "Waiting for E2 instance %s to be running", instance.instanceName)
	if instanceReadyErr := c.waitForE2InstanceCompletion(ctx, instance.instanceName); instanceReadyErr != nil {
		c.terminateE2Instance(ctx, instance.instanceName) // try to terminate the created instance
		return c.output.AddError(instanceReadyErr)        // fatal
	}

	c.logger.Info(ctx, "Gathering and parsing console log output...")

	err = c.findUnreachableEndpoints(ctx, instance.instanceName)
	if err != nil {
		c.output.AddError(err)
	}

	c.terminateE2Instance(ctx, instance.instanceName)

	return &c.output
}
