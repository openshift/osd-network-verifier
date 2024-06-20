package gcpverifier

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"time"

	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/clients/gcp"
	handledErrors "github.com/openshift/osd-network-verifier/pkg/errors"
	"github.com/openshift/osd-network-verifier/pkg/helpers"
	"github.com/openshift/osd-network-verifier/pkg/output"
	"golang.org/x/oauth2/google"
	computev1 "google.golang.org/api/compute/v1"
)

var (
	// TODO find a location for future docker images
	networkValidatorImage string = "quay.io/app-sre/osd-network-verifier:v0.1.159-9a6e0eb"
	userdataEndVerifier   string = "USERDATA END"
)

type GcpVerifier struct {
	GcpClient gcp.Client
	Logger    ocmlog.Logger
	Output    output.Output
}

func NewGcpVerifier(creds *google.Credentials, debug bool) (*GcpVerifier, error) {
	// Create logger
	builder := ocmlog.NewStdLoggerBuilder()
	builder.Debug(debug)
	logger, err := builder.Build()
	if err != nil {
		return &GcpVerifier{}, fmt.Errorf("unable to build logger: %s", err.Error())
	}

	gcpClient, err := gcp.NewClient(creds)
	if err != nil {
		return &GcpVerifier{}, err
	}

	return &GcpVerifier{*gcpClient, logger, output.Output{}}, nil
}

func (g *GcpVerifier) validateMachineType(projectID, zone, instanceType string) error {
	g.Logger.Debug(context.TODO(), "Gathering description of instance type %s from ComputeService API in zone %s", instanceType, zone)

	machineTypes, err := g.GcpClient.ListMachineTypes(projectID, zone)
	if err != nil {
		return fmt.Errorf("unable to gather list of supported instance types from ComputeService: %v", err)
	}

	if !machineTypes[instanceType] {
		return fmt.Errorf("instance type %s not found in ComputeService API for zone %s", instanceType, zone)
	}

	g.Logger.Debug(context.TODO(), "Instance type %s supported in zone %s", instanceType, zone)

	return nil
}

type createComputeServiceInstanceInput struct {
	projectID    string
	zone         string
	vpcSubnetID  string
	userdata     string
	machineType  string
	instanceName string
	sourceImage  string
	networkName  string
	tags         map[string]string
}

// this fuciton is a logic function that lieves some where else
func (g *GcpVerifier) createComputeServiceInstance(input createComputeServiceInstanceInput) (computev1.Instance, error) {

	req := &computev1.Instance{
		Name:        input.instanceName,
		MachineType: fmt.Sprintf("zones/%s/machineTypes/%s", input.zone, input.machineType),

		Disks: []*computev1.AttachedDisk{
			{
				InitializeParams: &computev1.AttachedDiskInitializeParams{
					DiskSizeGb:  10,
					SourceImage: input.sourceImage,
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

		Metadata: &computev1.Metadata{
			Items: []*computev1.MetadataItems{
				{
					Key:   "user-data",
					Value: &input.userdata,
				},
			},
		},
	}

	//send request to computeService

	err := g.GcpClient.CreateInstance(input.projectID, input.zone, req)
	if err != nil {
		return computev1.Instance{}, fmt.Errorf("unable to create instance: %v", err)
	}

	g.Logger.Info(context.TODO(), "Created instance with ID: %s", input.instanceName)

	//get fingerprint from instance
	inst, err := g.GcpClient.GetInstance(input.projectID, input.zone, input.instanceName)
	if err != nil {
		g.Logger.Debug(context.TODO(), "Failed to get fingerprint to apply tags to instance %v", err)
	}

	//Add tags - known as labels in gcp
	g.Logger.Info(context.TODO(), "Applying labels")

	labelReq := &computev1.InstancesSetLabelsRequest{
		LabelFingerprint: inst.LabelFingerprint,
		Labels:           input.tags,
	}

	//send request to apply tags, return error if tags are invalid
	err = g.GcpClient.SetInstanceLabels(input.projectID, input.zone, input.instanceName, labelReq)
	if err != nil {
		return computev1.Instance{}, fmt.Errorf("unable to create labels: %v", err)
	}

	g.Logger.Info(context.TODO(), "Successfully applied labels ")

	return inst, nil

}

func (g *GcpVerifier) findUnreachableEndpoints(projectID, zone, instanceName string) error {
	// Compile the regular expressions once
	reVerify := regexp.MustCompile(userdataEndVerifier)
	reUnreachableErrors := regexp.MustCompile(`Unable to reach (\S+)`)

	// getConsoleOutput then parse, use c.output to store result of the execution
	err := helpers.PollImmediate(30*time.Second, 4*time.Minute, func() (bool, error) {
		output, err := g.GcpClient.GetInstancePorts(projectID, zone, instanceName)
		if err != nil {
			return false, err
		}

		if output != nil {
			// First, gather the ComputeService console output
			scriptOutput := fmt.Sprintf("%#v", output)
			if err != nil {
				// unable to decode output. we will try again
				g.Logger.Debug(context.TODO(), "Error while collecting console output, will retry on next check interval: %s", err)
				return false, nil
			}

			// In the early stages, an ComputeService instance may be running but the console is not populated with any data, retry if that is the case
			if len(scriptOutput) < 1 {
				g.Logger.Debug(context.TODO(), "ComputeService console output not yet populated with data, continuing to wait...")
				return false, nil
			}

			// Check for the specific string we output in the generated userdata file at the end to verify the userdata script has run
			// It is possible we get EC2 console output, but the userdata script has not yet completed.
			verifyMatch := reVerify.FindString(string(scriptOutput))
			if len(verifyMatch) < 1 {
				g.Logger.Debug(context.TODO(), "ComputeService console output contains data, but end of userdata script not seen, continuing to wait...")
				return false, nil
			}

			// check output failures, report as exception if they occurred
			var rgx = regexp.MustCompile(`(?m)^(.*Cannot.*)|(.*Could not.*)|(.*Failed.*)|(.*command not found.*)`)
			notFoundMatch := rgx.FindAllStringSubmatch(string(scriptOutput), -1)

			if len(notFoundMatch) > 0 { //&& len(success) < 1
				g.Output.AddException(handledErrors.NewEgressURLError("internet connectivity problem: please ensure there's internet access in given vpc subnets"))
			}

			// If debug logging is enabled, output the full console log that appears to include the full userdata run
			g.Logger.Debug(context.TODO(), "Full ComputeService console output:\n---\n%s\n---", output)

			g.Output.SetEgressFailures(reUnreachableErrors.FindAllString(string(scriptOutput), -1))
			return true, nil
		}
		g.Logger.Debug(context.TODO(), "Waiting for UserData script to complete...")
		return false, nil
	})

	return err
}

func (c *GcpVerifier) describeComputeServiceInstances(projectID, zone, instanceName string) (string, error) {
	// States
	//PROVISIONING, STAGING, RUNNING, STOPPING, STOPPED, TERMINATED, SUSPENDED
	// https://cloud.google.com/compute/docs/instances/instance-life-cycle

	//Error Codes https://cloud.google.com/apis/design/errors

	resp, err := c.GcpClient.GetInstance(projectID, zone, instanceName)
	if err != nil {
		c.Logger.Error(context.TODO(), "Errors while describing the instance status: %v", err.Error())
		return "PERMISSION DENIED", err
	}
	switch resp.Status {
	case "PROVISIONING", "STAGING":
		c.Logger.Debug(context.TODO(), "Waiting on VM operation: %s", resp.Status)

	case "STOPPING", "STOPPED", "TERMINATED", "SUSPENDED":
		c.Logger.Debug(context.TODO(), "Fatal - Instance status: ", instanceName)
		return "FATAL", fmt.Errorf(resp.Status)
	}

	if len(resp.Status) == 0 {
		c.Logger.Debug(context.TODO(), "Instance %s has no status yet", instanceName)
	}
	return resp.Status, nil
}

func (c *GcpVerifier) waitForComputeServiceInstanceCompletion(projectID, zone, instanceName string) error {
	//wait for the instance to run
	err := helpers.PollImmediate(5*time.Second, 2*time.Minute, func() (bool, error) {
		code, descError := c.describeComputeServiceInstances(projectID, zone, instanceName)
		switch code {
		case "RUNNING":
			//instance is running, break
			c.Logger.Info(context.TODO(), "ComputeService Instance: %s %s", instanceName, code)
			return true, nil

		case "FATAL":
			return false, fmt.Errorf("instance %s already exists with %v state. Please run again", instanceName, descError)

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

func generateUserData(variables map[string]string) (string, error) {
	variableMapper := func(varName string) string {
		return variables[varName]
	}
	// TODO: REPLACE JUNK VALUE "helpers.UserdataTemplate" BELOW
	data := os.Expand("helpers.UserdataTemplate", variableMapper)

	return data, nil
}
