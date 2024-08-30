package gcp

import (
	"context"

	"golang.org/x/oauth2/google"
	computev1 "google.golang.org/api/compute/v1"
)

// Client represents a GCP Client
type Client struct {
	computeService *computev1.Service
}

func NewClient(credentials *google.Credentials) (*Client, error) {
	// Use oauth2 token in credentials struct to create a client,
	// https://pkg.go.dev/golang.org/x/oauth2/google#Credentials

	// https://cloud.google.com/docs/authentication/production
	// Service account credentials order/priority - env variable, service account attached to resource, error

	computeService, err := computev1.NewService(context.TODO())
	if err != nil {
		return nil, err
	}

	return &Client{computeService: computeService}, nil
}

// Terminates target ComputeService instance
// Uses c.output to store result of the execution
func (c *Client) TerminateComputeServiceInstance(projectID, zone, instanceName string) error {
	_, err := c.computeService.Instances.Delete(projectID, zone, instanceName).Context(context.TODO()).Do()
	return err
}

// Returns a map of all machineTypes with the machinetype string as the key and bool true if found
func (c *Client) ListMachineTypes(projectID, zone string) (map[string]bool, error) {
	machineTypesMap := map[string]bool{}
	req := c.computeService.MachineTypes.List(projectID, zone)
	err := req.Pages(context.TODO(), func(page *computev1.MachineTypeList) error {
		for _, machineType := range page.Items {
			machineTypesMap[machineType.Name] = true
		}
		return nil
	})
	if err != nil {
		return map[string]bool{}, err
	}
	return machineTypesMap, nil
}

// Creates an instance resource in the specified project using the data included in the request.
func (c *Client) CreateInstance(projectID, zone string, instance *computev1.Instance) error {
	_, err := c.computeService.Instances.Insert(projectID, zone, instance).Do()
	if err != nil {
		return err
	}
	return nil
}

// Gets instance given an ID , zone , and instance name
func (c *Client) GetInstance(projectID, zone, instanceName string) (computev1.Instance, error) {
	instance, err := c.computeService.Instances.Get(projectID, zone, instanceName).Do()
	if err != nil {
		return computev1.Instance{}, err
	}
	return *instance, nil
}

// Send request to apply tags, return error if tags are invalid
func (c *Client) SetInstanceLabels(projectID, zone, instanceName string, labelReq *computev1.InstancesSetLabelsRequest) error {
	_, err := c.computeService.Instances.SetLabels(projectID, zone, instanceName, labelReq).Do()
	if err != nil {
		return err
	}
	return nil
}

// Gets serial port output for the specified instance
func (c *Client) GetInstancePorts(projectID, zone, instanceName string) (*computev1.SerialPortOutput, error) {
	resp, err := c.computeService.Instances.GetSerialPortOutput(projectID, zone, instanceName).Do()
	if err != nil {
		return &computev1.SerialPortOutput{}, err
	}
	return resp, nil
}
