package gcpverifier

import (
	_ "embed"
	"encoding/base64"
	"fmt"
	"math/rand"
	"strconv"

	"github.com/openshift/osd-network-verifier/pkg/data/egress_lists"
	"github.com/openshift/osd-network-verifier/pkg/helpers"
	"github.com/openshift/osd-network-verifier/pkg/output"
	"github.com/openshift/osd-network-verifier/pkg/probes/curl"
	"github.com/openshift/osd-network-verifier/pkg/verifier"
)

//go:embed startup-script.sh
var startupScript string

const (
	DEFAULT_CLOUDIMAGEID  = "rhel-9-v20240703"
	DEFAULT_INSTANCE_TYPE = "e2-micro"
	DEFAULT_TIMEOUT       = 5
)

// validateEgress performs validation process for egress
// Basic workflow is:
// - prepare for ComputeService instance creation
// - create instance and wait till it gets ready, wait for gcpUserData script execution
// - find unreachable endpoints & parse output, then terminate instance
// - return `g.output` which stores the execution results
func (g *GcpVerifier) ValidateEgress(vei verifier.ValidateEgressInput) *output.Output {
	// Validate cloud platform type and default to PlatformGCP if not specified
	if vei.PlatformType == "" {
		vei.PlatformType = helpers.PlatformGCP
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
		vei.InstanceType = DEFAULT_INSTANCE_TYPE
	}
	if err := g.validateMachineType(vei.GCP.ProjectID, vei.GCP.Zone, vei.InstanceType); err != nil {
		return g.Output.AddError(fmt.Errorf("instance type %s is invalid: %s", vei.InstanceType, err))
	}

	// Fetch the egress URL list from github, falling back to local lists in the event of a failure.
	egressListYaml := vei.EgressListYaml
	var egressListStr, tlsDisabledEgressListStr string
	if egressListYaml == "" {
		githubEgressList, githubListErr := egress_lists.GetGithubEgressList(vei.PlatformType)
		if githubListErr == nil {
			egressListYaml, githubListErr = githubEgressList.GetContent()
			if githubListErr == nil {
				g.Logger.Debug(vei.Ctx, "Using egress URL list from %s at SHA %s", githubEgressList.GetURL(), githubEgressList.GetSHA())
				egressListStr, tlsDisabledEgressListStr, githubListErr = egress_lists.EgressListToString(egressListYaml, map[string]string{})
			}
		}
		var err error
		if githubListErr != nil {
			g.Output.AddError(fmt.Errorf("failed to get egress list from GitHub, falling back to local list: %v", githubListErr))
			egressListYaml, err = egress_lists.GetLocalEgressList(vei.PlatformType)
			if err != nil {
				return g.Output.AddError(err)
			}
			egressListStr, tlsDisabledEgressListStr, err = egress_lists.EgressListToString(egressListYaml, map[string]string{})
			if err != nil {
				return g.Output.AddError(err)
			}
		}
	}

	// Generate the userData file
	// Expand replaces all ${var} (using empty string for unknown ones), adding the env variables used in startup-script.sh
	// Must add fake userDatavariables to replace parts of startup-script.sh that are not env variables but start with $
	userDataVariables := map[string]string{
		"AWS_REGION":       "us-east-2", // Not sure if this is the correct data
		"TIMEOUT":          vei.Timeout.String(),
		"HTTP_PROXY":       vei.Proxy.HttpProxy,
		"HTTPS_PROXY":      vei.Proxy.HttpsProxy,
		"CACERT":           base64.StdEncoding.EncodeToString([]byte(vei.Proxy.Cacert)),
		"NOTLS":            strconv.FormatBool(vei.Proxy.NoTls),
		"DELAY":            "5",
		"URLS":             egressListStr,
		"TLSDISABLED_URLS": tlsDisabledEgressListStr,
		"ret":              "${ret}",
		"?":                "$?",
		"array[@]":         "${array[@]}",
		"value":            "$value",
	}

	userData, err := vei.Probe.GetExpandedUserData(userDataVariables, startupScript)
	if err != nil {
		return g.Output.AddError(err)
	}
	g.Logger.Debug(vei.Ctx, "Generated userdata script:\n---\n%s\n---", userData)

	if vei.CloudImageID == "" {
		vei.CloudImageID = DEFAULT_CLOUDIMAGEID
	}

	// Create ComputeService instance
	// Image list https://cloud.google.com/compute/docs/images/os-details#red_hat_enterprise_linux_rhel
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
	if err != nil {
		g.Output.AddError(err)
		err = g.GcpClient.TerminateComputeServiceInstance(vei.GCP.ProjectID, vei.GCP.Zone, instance.Name)
		return g.Output.AddError(err) // fatal
	}

	g.Logger.Debug(vei.Ctx, "Waiting for ComputeService instance %s to be running", instance.Name)

	if instanceReadyErr := g.waitForComputeServiceInstanceCompletion(vei.GCP.ProjectID, vei.GCP.Zone, instance.Name); instanceReadyErr != nil {
		// try to terminate instance if instance not running
		err = g.GcpClient.TerminateComputeServiceInstance(vei.GCP.ProjectID, vei.GCP.Zone, instance.Name)
		if err != nil {
			g.Output.AddError(err)
		}
		return g.Output.AddError(instanceReadyErr) // fatal
	}

	g.Logger.Info(vei.Ctx, "Gathering and parsing console log output...")

	err = g.findUnreachableEndpoints(vei.GCP.ProjectID, vei.GCP.Zone, instance.Name, vei.Probe)
	if err != nil {
		g.Output.AddError(err)
	}

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
