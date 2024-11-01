package gcpverifier

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/openshift/osd-network-verifier/pkg/data/cloud"
	"github.com/openshift/osd-network-verifier/pkg/data/cpu"
	"github.com/openshift/osd-network-verifier/pkg/data/egress_lists"
	"github.com/openshift/osd-network-verifier/pkg/output"
	"github.com/openshift/osd-network-verifier/pkg/probes/curl"
	"github.com/openshift/osd-network-verifier/pkg/verifier"
)

const (
	DEFAULT_TIMEOUT = 5 * time.Second
)

// validateEgress performs validation process for egress
// Basic workflow is:
// - prepare for ComputeService instance creation
// - create instance and wait till it gets ready, wait for startup script execution
// - find unreachable endpoints & parse output, then terminate instance
// - return `g.output` which stores the execution results
func (g *GcpVerifier) ValidateEgress(vei verifier.ValidateEgressInput) *output.Output {
	// Validate cloud platform type and default to PlatformGCP if not specified
	if !vei.PlatformType.IsValid() {
		vei.PlatformType = cloud.GCPClassic
	}
	// Validate CPUArchitecture and default to ArchX86 if not specified
	if !vei.CPUArchitecture.IsValid() {
		vei.CPUArchitecture = cpu.ArchX86
	}

	// Default to curl.Probe if no Probe specified
	if vei.Probe == nil {
		vei.Probe = curl.Probe{}
		g.Logger.Debug(vei.Ctx, "defaulted to curl probe")
	}

	// Set timeout to default if not specified
	if vei.Timeout <= 0 {
		vei.Timeout = DEFAULT_TIMEOUT
	}
	g.Logger.Debug(vei.Ctx, "configured a %s timeout for each egress request", vei.Timeout)

	// Set instance type to default if not specified and validate it
	if vei.InstanceType == "" {
		var err error
		vei.InstanceType, err = vei.CPUArchitecture.DefaultInstanceType(cloud.GCPClassic)
		if err != nil {
			return g.Output.AddError(err)
		}
		g.Logger.Debug(vei.Ctx, fmt.Sprintf("defaulted to instance type %s", vei.InstanceType))
	}

	// Validate machine type
	if err := g.validateMachineType(vei.GCP.ProjectID, vei.GCP.Zone, vei.InstanceType); err != nil {
		return g.Output.AddError(fmt.Errorf("instance type %s is invalid: %s", vei.InstanceType, err))
	}

	// Fetch the egress URL list from github, falling back to local lists in the event of a failure.
	egressListYaml := vei.EgressListYaml
	var egressListStr, tlsDisabledEgressListStr string
	if egressListYaml == "" {
		githubEgressList, err := egress_lists.GetGithubEgressList(vei.PlatformType)
		if err != nil {
			g.Logger.Error(vei.Ctx, "Failed to get egress list from GitHub, falling back to local list: %v", err)

			egressListYaml, err = egress_lists.GetLocalEgressList(vei.PlatformType)
			if err != nil {
				return g.Output.AddError(err)
			}
		} else {
			egressListYaml, err = githubEgressList.GetContent()
			if err != nil {
				return g.Output.AddError(err)
			}

			g.Logger.Info(vei.Ctx, "Using egress URL list from %s at SHA %s", githubEgressList.GetURL(), githubEgressList.GetSHA())
		}
	}

	egressListStr, tlsDisabledEgressListStr, err := egress_lists.EgressListToString(egressListYaml, map[string]string{})
	if err != nil {
		return g.Output.AddError(err)
	}

	// Generate the userData file
	// Expand replaces all ${var} (using empty string for unknown ones), adding the env variables used in startup-script.sh
	userDataVariables := map[string]string{
		"TIMEOUT":          vei.Timeout.String(),
		"HTTP_PROXY":       vei.Proxy.HttpProxy,
		"HTTPS_PROXY":      vei.Proxy.HttpsProxy,
		"CACERT":           base64.StdEncoding.EncodeToString([]byte(vei.Proxy.Cacert)),
		"NO_PROXY":         vei.Proxy.NoProxyAsString(),
		"NOTLS":            strconv.FormatBool(vei.Proxy.NoTls),
		"DELAY":            "5",
		"URLS":             egressListStr,
		"TLSDISABLED_URLS": tlsDisabledEgressListStr,
		// Add fake userDatavariables to replace normal shell variables in startup-script.sh which will otherwise be erased by os.Expand
		"ret":         "${ret}",
		"?":           "$?",
		"array[@]":    "${array[@]}",
		"value":       "$value",
		"USE_SYSTEMD": "true",
	}

	userData, err := vei.Probe.GetExpandedUserData(userDataVariables)
	if err != nil {
		return g.Output.AddError(err)
	}
	g.Logger.Debug(vei.Ctx, "Generated userdata script:\n---\n%s\n---", userData)

	// if no cloudImageID specified, get string ID of the VM image to be used for the probe instance
	// image list https://cloud.google.com/compute/docs/images/os-details#red_hat_enterprise_linux_rhel
	if vei.CloudImageID == "" {
		vei.CloudImageID, err = vei.Probe.GetMachineImageID(vei.PlatformType, vei.CPUArchitecture, vei.GCP.Region)
		if err != nil {
			return g.Output.AddError(err)
		}
	}

	// Create the ComputeService instance
	instance, err := g.createComputeServiceInstance(createComputeServiceInstanceInput{
		projectID:        vei.GCP.ProjectID,
		zone:             vei.GCP.Zone,
		vpcSubnetID:      fmt.Sprintf("projects/%s/regions/%s/subnetworks/%s", vei.GCP.ProjectID, vei.GCP.Region, vei.SubnetID),
		userdata:         userData,
		machineType:      vei.InstanceType,
		instanceName:     fmt.Sprintf("verifier-%v", rand.Intn(10000)),
		sourceImage:      fmt.Sprintf("projects/rhel-cloud/global/images/%s", vei.CloudImageID),
		networkName:      fmt.Sprintf("projects/%s/global/networks/%s", vei.GCP.ProjectID, vei.GCP.VpcName),
		tags:             vei.Tags,
		serialportenable: "true",
	})
	// Try to terminate instance if instance creation fails
	if err != nil {
		g.Output.AddError(err)
		err = g.GcpClient.TerminateComputeServiceInstance(vei.GCP.ProjectID, vei.GCP.Zone, instance.Name)
		return g.Output.AddError(err) // fatal
	}

	// Wait for the ComputeService instance to be running
	g.Logger.Debug(vei.Ctx, "Waiting for ComputeService instance %s to be running", instance.Name)
	if instanceReadyErr := g.waitForComputeServiceInstanceCompletion(vei.GCP.ProjectID, vei.GCP.Zone, instance.Name); instanceReadyErr != nil {
		// try to terminate instance if instance is not running
		err = g.GcpClient.TerminateComputeServiceInstance(vei.GCP.ProjectID, vei.GCP.Zone, instance.Name)
		if err != nil {
			g.Output.AddError(err)
		}
		return g.Output.AddError(instanceReadyErr) // fatal
	}

	// Wait for console output and parse
	g.Logger.Info(vei.Ctx, "Gathering and parsing console log output...")
	err = g.findUnreachableEndpoints(vei.GCP.ProjectID, vei.GCP.Zone, instance.Name, vei.Probe)
	if err != nil {
		g.Output.AddError(err)
	}

	// Terminate the ComputeService instance after probe output is parsed and stored
	err = g.GcpClient.TerminateComputeServiceInstance(vei.GCP.ProjectID, vei.GCP.Zone, instance.Name)
	if err != nil {
		g.Output.AddError(err)
	}

	return &g.Output
}

// TODO():
func (g *GcpVerifier) VerifyDns(vdi verifier.VerifyDnsInput) *output.Output {
	return &output.Output{}
}
