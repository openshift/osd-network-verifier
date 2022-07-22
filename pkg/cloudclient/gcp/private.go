package gcp

//Features to add - image-id, kms-key-id
import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"time"

	"golang.org/x/oauth2/google"
	computev1 "google.golang.org/api/compute/v1"

	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	handledErrors "github.com/openshift/osd-network-verifier/pkg/errors"
	"github.com/openshift/osd-network-verifier/pkg/helpers"
	"github.com/openshift/osd-network-verifier/pkg/output"
)

type createComputeServiceInstanceInput struct {
	ContOptImageID string
	vpcSubnetID    string
	userdata       string
	// ebsKmsKeyID   string
	zone         string
	machineType  string
	instanceName string
	sourceImage  string
	networkName  string
}

// //global variable ContOptImage image, gcp has region, zone

var (
	defaultContOptImage = map[string]string{
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

	// https://cloud.google.com/docs/authentication/production
	//service account credentials order - env variable, service account attached to resource, error

	computeService, err := computev1.NewService(ctx)
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

	if err := c.validateMachineType(ctx); err != nil {
		return nil, fmt.Errorf("Instance type %s is invalid: %v", c.instanceType, err)
	}

	return c, nil
}

func (c *Client) validateMachineType(ctx context.Context) error {
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
		return fmt.Errorf("Unable to gather list of supported instance types from ComputeService: %v", err)
	}

	if !found {
		return fmt.Errorf("Instance type %s not found in ComputeService API", c.instanceType)
	}

	return nil
}

//ToDo func createComputeServiceInstance
func (c *Client) createComputeServiceInstance(ctx context.Context, input createComputeServiceInstanceInput) (createComputeServiceInstanceInput, error) {

	req := &computev1.Instance{
		Name:        input.instanceName,
		MachineType: fmt.Sprintf("zones/%s/machineTypes/%s", c.zone, input.machineType),

		// Tags: &computev1.Tags{
		// 	Items: []string{"http-server", "https-server"},
		// },
		Disks: []*computev1.AttachedDisk{
			{
				InitializeParams: &computev1.AttachedDiskInitializeParams{
					DiskSizeGb:  10,
					SourceImage: input.sourceImage,
					// sourceImageEncryptionKey: &computepb.
				},
				AutoDelete: true,
				Boot:       true,
				Type:       "PERSISTENT",
			},
		},

		NetworkInterfaces: []*computev1.NetworkInterface{
			{
				Name:       input.networkName,
				Subnetwork: input.vpcSubnetID,
			},
		},
		//pass gcpuserdata.yaml cloud-init script
		Metadata: &computev1.Metadata{
			Items: []*computev1.MetadataItems{
				//can pass startup script
				// {
				// 	Key: proto.String("startup-script"),
				// 	Value: proto.String("#!/bin/bash\n"),
				// },

				//pass gcpuserdata.yaml
				{
					Key:   "user-data",
					Value: &input.userdata,
				},
			},
		},
	}

	//send request to computeService
	instanceResp, err := c.computeService.Instances.Insert(c.projectID, c.zone, req).Context(ctx).Do()
	if err != nil {
		fmt.Errorf("unable to create instance: %v %v", err, instanceResp)
	}

	c.logger.Info(ctx, "Created instance with ID: %s", input.instanceName)

	//get fingerprint from instance
	inst, err := c.computeService.Instances.Get(c.projectID, c.zone, input.instanceName).Do()
	if err != nil {
		fmt.Errorf("Failed to get fingerprint to apply tags to instance %v", err)
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
	}

	if resp != nil {
		c.logger.Info(ctx, "Successfully applied labels ")
	}

	return input, nil

}

//ToDo func describeComputeServiceInstances - check status code meaning and return
// Returns instance state
func (c *Client) describeComputeServiceInstances(ctx context.Context, instanceName string) (string, error) {
	// States
	//PROVISIONING, STAGING, RUNNING, STOPPING, STOPPED, TERMINATED, SUSPENDED
	// https://cloud.google.com/compute/docs/instances/instance-life-cycle

	//Error Codes https://cloud.google.com/apis/design/errors

	resp, err := c.computeService.Instances.Get(c.projectID, c.zone, instanceName).Context(ctx).Do()
	if err != nil {
		c.logger.Error(ctx, "Errors while describing the instance status: %v", err.Error())
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
func (c *Client) waitForComputeServiceInstanceCompletion(ctx context.Context, instanceName string) error {
	//wait for the instance to run
	err := helpers.PollImmediate(5*time.Second, 2*time.Minute, func() (bool, error) {
		code, descError := c.describeComputeServiceInstances(ctx, instanceName)
		switch code {
		case "RUNNING":
			//instance is running, break
			c.logger.Info(ctx, "ComputeService Instance: %s %s", instanceName, code)
			return true, nil

		case "FATAL":
			return false, fmt.Errorf("Instance %s already exists with %v state. Please run again", instanceName, descError)

		case "PERMISSION DENIED":
			return false, fmt.Errorf("missing required permissions for account: %v", descError)
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
			// First, gather the ComputeService console output
			scriptOutput := fmt.Sprintf("%#v", output)

			if err != nil {
				// unable to decode output. we will try again
				c.logger.Debug(ctx, "Error while collecting console output, will retry on next check interval: %s", err)
				return false, nil
			}

			// In the early stages, an ComputeService instance may be running but the console is not populated with any data, retry if that is the case
			if len(scriptOutput) < 1 {
				c.logger.Debug(ctx, "ComputeService console output not yet populated with data, continuing to wait...")
				return false, nil
			}

			// Check for the specific string we output in the generated userdata file at the end to verify the userdata script has run
			// It is possible we get EC2 console output, but the userdata script has not yet completed.
			verifyMatch := reVerify.FindString(string(scriptOutput))
			if len(verifyMatch) < 1 {
				c.logger.Debug(ctx, "ComputeService console output contains data, but end of userdata script not seen, continuing to wait...")
				return false, nil
			}

			// check output failures, report as exception if they occurred
			var rgx = regexp.MustCompile(`(?m)^(.*Cannot.*)|(.*Could not.*)|(.*Failed.*)|(.*command not found.*)`)
			notFoundMatch := rgx.FindAllStringSubmatch(string(scriptOutput), -1)

			if len(notFoundMatch) > 0 { //&& len(success) < 1
				c.output.AddException(handledErrors.NewEgressURLError("internet connectivity problem: please ensure there's internet access in given vpc subnets"))
			}

			// If debug logging is enabled, output the full console log that appears to include the full userdata run
			c.logger.Debug(ctx, "Full ComputeService console output:\n---\n%s\n---", output)

			c.output.SetEgressFailures(reUnreachableErrors.FindAllString(string(scriptOutput), -1))
			return true, nil
		}
		c.logger.Debug(ctx, "Waiting for UserData script to complete...")
		return false, nil
	})

	return err
}

// terminateComputeServiceInstance terminates target ec2 instance
// uses c.output to store result of the execution
func (c *Client) terminateComputeServiceInstance(ctx context.Context, instanceName string) {
	c.logger.Info(ctx, "Terminating ComputeService instance with id %s", instanceName)

	_, err := c.computeService.Instances.Stop(c.projectID, c.zone, instanceName).Context(ctx).Do()
	if err != nil {
		fmt.Errorf("Unable to terminate instance: %v", err)
	}

	c.output.AddError(err)

}

func (c *Client) setCloudImage(cloudImageID string) (string, error) {
	// If a cloud image wasn't provided by the caller,
	// if cloudImageID == "" {
	// use defaultContOptImage for the region instead
	// cloudImageID = defaultContOptImage[c.region]
	cloudImageID = defaultContOptImage["default"]
	if cloudImageID == "" {
		return "", fmt.Errorf("no default container optimized image (ContOptImage) found for region %s ", c.region)
	}
	// }

	return cloudImageID, nil
}

// validateEgress performs validation process for egress
// Basic workflow is:
// - prepare for ComputeService instance creation
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

	instance, err := c.createComputeServiceInstance(ctx, createComputeServiceInstanceInput{
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

	c.logger.Debug(ctx, "Waiting for ComputeService instance %s to be running", instance.instanceName)
	if instanceReadyErr := c.waitForComputeServiceInstanceCompletion(ctx, instance.instanceName); instanceReadyErr != nil {
		c.terminateComputeServiceInstance(ctx, instance.instanceName) // try to terminate the created instance
		return c.output.AddError(instanceReadyErr)                    // fatal
	}

	c.logger.Info(ctx, "Gathering and parsing console log output...")

	err = c.findUnreachableEndpoints(ctx, instance.instanceName)
	if err != nil {
		c.output.AddError(err)
	}

	c.terminateComputeServiceInstance(ctx, instance.instanceName)

	return &c.output
}
