package kube

import (
	"context"

	"k8s.io/client-go/kubernetes"
)

// Client represents a KubeAPI Client
type Client struct {
	clientset *kubernetes.Clientset
}

func NewClient(clientset *kubernetes.Clientset) (*Client, error) {
	return &Client{clientset: clientset}, nil
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

// Creates a Job
func (c *Client) CreateJob(projectID, zone string, instance *computev1.Instance) error {
	_, err := c.computeService.Instances.Insert(projectID, zone, instance).Do()
	if err != nil {
		return err
	}
	return nil
}

// Gets instance given an ID , zone , and instance name
func (c *Client) GetPodLogs(projectID, zone, instanceName string) (computev1.Instance, error) {
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
