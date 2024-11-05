package gcpverifier

import (
	"context"
	"fmt"
	"strings"
	"time"

	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"golang.org/x/oauth2/google"
	computev1 "google.golang.org/api/compute/v1"

	"github.com/openshift/osd-network-verifier/pkg/clients/gcp"
	handledErrors "github.com/openshift/osd-network-verifier/pkg/errors"
	"github.com/openshift/osd-network-verifier/pkg/helpers"
	"github.com/openshift/osd-network-verifier/pkg/output"
	"github.com/openshift/osd-network-verifier/pkg/probes"
)

type GcpVerifier struct {
	GcpClient gcp.Client
	Logger    ocmlog.Logger
	Output    output.Output
}

type createComputeServiceInstanceInput struct {
	projectID        string
	zone             string
	vpcSubnetID      string
	userdata         string
	machineType      string
	instanceName     string
	sourceImage      string
	networkName      string
	tags             map[string]string
	serialportenable string
}

// Creates new GCP verifier with ocm logger
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

// Check that instance type is supported in zone
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

// This function is a logic function that lives somewhere else
func (g *GcpVerifier) createComputeServiceInstance(input createComputeServiceInstanceInput) (computev1.Instance, error) {

	req := &computev1.Instance{
		Name:        input.instanceName,
		MachineType: fmt.Sprintf("zones/%s/machineTypes/%s", input.zone, input.machineType),

		Disks: []*computev1.AttachedDisk{
			{
				InitializeParams: &computev1.AttachedDiskInitializeParams{
					DiskSizeGb:  20,
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
				// Only one accessConfigs exist which is ONE_TO_ONE_NAT
				// Needed for external internet access including egress
				AccessConfigs: []*computev1.AccessConfig{
					{
						Name: "External NAT",
					},
				},
			},
		},
		ServiceAccounts: []*computev1.ServiceAccount{
			{
				Email: "default",
				Scopes: []string{
					"https://www.googleapis.com/auth/cloud-platform",
				},
			},
		},
		Metadata: &computev1.Metadata{
			Items: []*computev1.MetadataItems{
				{
					Key:   "serial-port-enable",
					Value: &input.serialportenable,
				},
				{
					Key:   "startup-script",
					Value: &input.userdata,
				},
			},
		},
	}

	// Send request to create instance
	err := g.GcpClient.CreateInstance(input.projectID, input.zone, req)
	if err != nil {
		return computev1.Instance{}, fmt.Errorf("unable to create instance: %v", err)
	}
	g.Logger.Info(context.TODO(), "Created instance with ID: %s", input.instanceName)

	// Get fingerprint from instance
	inst, err := g.GcpClient.GetInstance(input.projectID, input.zone, input.instanceName)
	if err != nil {
		g.Logger.Debug(context.TODO(), "Failed to get fingerprint to apply tags to instance %v", err)
	}

	// Add tags - known as labels in gcp
	g.Logger.Info(context.TODO(), "Applying labels")
	labelReq := &computev1.InstancesSetLabelsRequest{
		LabelFingerprint: inst.LabelFingerprint,
		Labels:           input.tags,
	}

	// Send request to apply tags, return error if tags are invalid
	err = g.GcpClient.SetInstanceLabels(input.projectID, input.zone, input.instanceName, labelReq)
	if err != nil {
		return computev1.Instance{}, fmt.Errorf("unable to create labels: %v", err)
	}
	g.Logger.Info(context.TODO(), "Successfully applied labels ")

	return inst, nil

}

// Get the console output from the ComputeService instance and scrape it for the probe's output and parse
func (g *GcpVerifier) findUnreachableEndpoints(projectID, zone, instanceName string, probe probes.Probe) error {
	var consoleOutput string
	g.Logger.Debug(context.TODO(), "Scraping console output and waiting for user data script to complete...")

	// Scrapes console at specified interval up to specified timeout
	err := helpers.PollImmediate(30*time.Second, 4*time.Minute, func() (bool, error) {
		// Get the console output from the ComputeService instance
		output, err := g.GcpClient.GetInstancePorts(projectID, zone, instanceName)
		if err != nil {
			return false, err
		}

		// Return and resume waiting if console output is still nil
		if output == nil {
			return false, nil
		}

		// In the early stages, an ComputeService instance may be running but the console is not populated with any data
		if len(output.Contents) == 0 {
			g.Logger.Debug(context.TODO(), "ComputeService console output not yet populated with data, continuing to wait...")
			return false, nil
		}
		consoleOutput = output.Contents

		// Check for startingToken and endingToken
		startingTokenSeen := strings.Contains(consoleOutput, probe.GetStartingToken())
		endingTokenSeen := strings.Contains(consoleOutput, probe.GetEndingToken())
		if !startingTokenSeen {
			if endingTokenSeen {
				g.Logger.Debug(context.TODO(), "raw console logs:\n---\n%s\n---", output.Contents)
				g.Output.AddException(handledErrors.NewGenericError(fmt.Errorf("probe output corrupted: endingToken encountered before startingToken")))
				return false, nil
			}
			g.Logger.Debug(context.TODO(), "consoleOutput contains data, but probe has not yet printed startingToken, continuing to wait...")
			return false, nil
		}
		if !endingTokenSeen {
			g.Logger.Debug(context.TODO(), "consoleOutput contains data, but probe has not yet printed endingToken, continuing to wait...")
			return false, nil
		}
		// If we make it this far, we know that both startingTokenSeen and endingTokenSeen are true

		// Separate the probe's output from the rest of the console output (using startingToken and endingToken)
		rawProbeOutput := strings.TrimSpace(helpers.CutBetween(consoleOutput, probe.GetStartingToken(), probe.GetEndingToken()))
		if len(rawProbeOutput) < 1 {
			g.Logger.Debug(context.TODO(), "raw console logs:\n---\n%s\n---", consoleOutput)
			g.Output.AddException(handledErrors.NewGenericError(fmt.Errorf("probe output corrupted: no data between startingToken and endingToken")))
			return false, nil
		}

		// Send probe's output off to the Probe interface for parsing
		g.Logger.Debug(context.TODO(), "probe output:\n---\n%s\n---", rawProbeOutput)
		probe.ParseProbeOutput(false, rawProbeOutput, &g.Output)

		return true, nil
	})

	return err
}

// Describes the instance status
// States: PROVISIONING, STAGING, RUNNING, STOPPING, STOPPED, TERMINATED, SUSPENDED
// https://cloud.google.com/compute/docs/instances/instance-life-cycle
// Error Codes: https://cloud.google.com/apis/design/errors
func (g *GcpVerifier) describeComputeServiceInstances(projectID, zone, instanceName string) (string, error) {
	resp, err := g.GcpClient.GetInstance(projectID, zone, instanceName)
	if err != nil {
		g.Logger.Error(context.TODO(), "Errors while describing the instance status: %v", err.Error())
		return "PERMISSION DENIED", err
	}
	switch resp.Status {
	case "PROVISIONING", "STAGING":
		g.Logger.Debug(context.TODO(), "Waiting on VM operation: %s", resp.Status)

	case "STOPPING", "STOPPED", "TERMINATED", "SUSPENDED":
		g.Logger.Debug(context.TODO(), "Fatal - Instance status: ", instanceName)
		return "FATAL", fmt.Errorf(resp.Status)
	}

	if len(resp.Status) == 0 {
		g.Logger.Debug(context.TODO(), "Instance %s has no status yet", instanceName)
	}
	return resp.Status, nil
}

// Waits for the ComputeService instance to be in a RUNNING state
func (g *GcpVerifier) waitForComputeServiceInstanceCompletion(projectID, zone, instanceName string) error {
	err := helpers.PollImmediate(5*time.Second, 2*time.Minute, func() (bool, error) {
		code, descError := g.describeComputeServiceInstances(projectID, zone, instanceName)
		switch code {
		case "RUNNING":
			// Instance is running, break
			g.Logger.Info(context.TODO(), "ComputeService Instance: %s %s", instanceName, code)
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
