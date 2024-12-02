package egress

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2/google"

	"github.com/openshift/osd-network-verifier/cmd/utils"
	"github.com/openshift/osd-network-verifier/pkg/data/cloud"
	"github.com/openshift/osd-network-verifier/pkg/data/cpu"
	"github.com/openshift/osd-network-verifier/pkg/probes/curl"
	"github.com/openshift/osd-network-verifier/pkg/probes/legacy"
	"github.com/openshift/osd-network-verifier/pkg/proxy"
	"github.com/openshift/osd-network-verifier/pkg/verifier"
	gcpverifier "github.com/openshift/osd-network-verifier/pkg/verifier/gcp"
)

var (
	awsDefaultTags = map[string]string{"osd-network-verifier": "owned", "red-hat-managed": "true", "Name": "osd-network-verifier"}
	gcpDefaultTags = map[string]string{"osd-network-verifier": "owned", "red-hat-managed": "true", "name": "osd-network-verifier"}
)

const (
	awsRegionEnvVarStr = "AWS_REGION"
	awsRegionDefault   = "us-east-2"
	gcpRegionEnvVarStr = "GCP_REGION"
	gcpRegionDefault   = "us-east1"
)

type egressConfig struct {
	vpcSubnetID                string
	cloudImageID               string
	instanceType               string
	cpuArchName                string
	securityGroupIDs           []string
	egressListLocation         string
	cloudTags                  map[string]string
	debug                      bool
	region                     string
	timeout                    time.Duration
	kmsKeyID                   string
	httpProxy                  string
	httpsProxy                 string
	CaCert                     string
	noProxy                    []string
	noTls                      bool
	platformType               string
	awsProfile                 string
	gcpVpcName                 string
	skipAWSInstanceTermination bool
	terminateDebugInstance     string
	importKeyPair              string
	ForceTempSecurityGroup     bool
	probeName                  string
}

func NewCmdValidateEgress() *cobra.Command {
	config := egressConfig{}

	validateEgressCmd := &cobra.Command{
		Use:   "egress",
		Short: "Verify essential OpenShift domains are reachable from given subnet ID.",
		Long:  `Verify essential OpenShift domains are reachable from given subnet ID.`,
		Example: `For AWS, ensure your credential environment vars
AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY (also AWS_SESSION_TOKEN for STS credentials)
are set correctly before execution.

# Verify that essential OpenShift domains are reachable from a given SUBNET_ID/SECURITY_GROUP association
./osd-network-verifier egress --subnet-id ${SUBNET_ID} --security-group-ids ${SECURITY_GROUP}`,
		Run: func(cmd *cobra.Command, args []string) {
			platformType, err := cloud.ByName(config.platformType)
			if err != nil {
				//Unknown platformType specified
				fmt.Println(err)
				os.Exit(1)
			}

			// Set Region
			if config.region == "" {
				config.region = getDefaultRegion(platformType)
			}

			if config.CaCert != "" {
				// Read in the cert file
				cert, err := os.ReadFile(config.CaCert)
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
				// store string form of it
				// this was agreed with sda that they'll be communicating it as a string.
				config.CaCert = bytes.NewBuffer(cert).String()
			}

			p := proxy.ProxyConfig{
				HttpProxy:  config.httpProxy,
				HttpsProxy: config.httpsProxy,
				Cacert:     config.CaCert,
				NoTls:      config.noTls,
				NoProxy:    config.noProxy,
			}

			// setup non cloud config options
			vei := verifier.ValidateEgressInput{
				Ctx:          context.TODO(),
				SubnetID:     config.vpcSubnetID,
				CloudImageID: config.cloudImageID,
				Timeout:      config.timeout,
				Tags:         config.cloudTags,
				InstanceType: config.instanceType,
				PlatformType: platformType,
				Proxy:        p,
			}

			// AWS workflow
			if platformType == cloud.AWSClassic || platformType == cloud.AWSHCP || platformType == cloud.AWSHCPZeroEgress {

				if len(vei.Tags) == 0 {
					vei.Tags = awsDefaultTags
				}

				//Setup AWS Specific Configs
				vei.AWS = verifier.AwsEgressConfig{
					KmsKeyID:         config.kmsKeyID,
					SecurityGroupIDs: config.securityGroupIDs,
				}

				awsVerifier, err := utils.GetAwsVerifier(config.region, config.awsProfile, config.debug)
				if err != nil {
					fmt.Printf("could not build awsVerifier %v\n", err)
					os.Exit(1)
				}

				awsVerifier.Logger.Warn(context.TODO(), "Using region: %s", config.region)

				vei.SkipInstanceTermination = config.skipAWSInstanceTermination
				vei.TerminateDebugInstance = config.terminateDebugInstance
				vei.ImportKeyPair = config.importKeyPair
				vei.ForceTempSecurityGroup = config.ForceTempSecurityGroup

				// Probe selection
				switch strings.ToLower(config.probeName) {
				case "", "curl", "curlprobe", "curl.probe":
					vei.Probe = curl.Probe{}
					if config.egressListLocation != "" {
						vei.EgressListYaml, err = getCustomEgressListFromFlag(config.egressListLocation)
						if err != nil {
							fmt.Println(err)
							return
						}
					}
				case "legacy", "legacyprobe", "legacy.probe":
					vei.Probe = legacy.Probe{}
				}

				// Map specified CPU architecture name to cpu.Architecture type
				vei.CPUArchitecture = cpu.ArchitectureByName(config.cpuArchName)
				if config.cpuArchName != "" && !vei.CPUArchitecture.IsValid() {
					// Unknown cpu.Architecture specified
					fmt.Printf("unknown CPU architecture '%s'\n", config.cpuArchName)
					os.Exit(1)
				}

				out := verifier.ValidateEgress(awsVerifier, vei)
				out.Summary(config.debug)

				if !out.IsSuccessful() {
					awsVerifier.Logger.Error(context.TODO(), "Failure!")
					os.Exit(1)
				}

				awsVerifier.Logger.Info(context.TODO(), "Success")
				os.Exit(0)
			}

			// GCP workflow
			if platformType == cloud.GCPClassic {

				if len(vei.Tags) == 0 {
					vei.Tags = gcpDefaultTags
				}

				projectID := os.Getenv("GCP_PROJECT_ID")
				if projectID == "" {
					fmt.Println("please set environment variable GCP_PROJECT_ID to the project ID of the VPC")
					os.Exit(1)
				}

				vpcName := config.gcpVpcName
				if vpcName == "" {
					fmt.Println("please pass the flag --vpc-name=<VPC-NAME> to identify the VPC")
					os.Exit(1)
				}

				//Setup GCP Secific Configs
				vei.GCP = verifier.GcpEgressConfig{
					Region: config.region,
					//Zone b is supported by all regions and has the most machine types compared to zone a and c
					//https://cloud.google.com/compute/docs/regions-zones#available
					Zone:      fmt.Sprintf("%s-b", config.region),
					ProjectID: projectID,
					VpcName:   vpcName,
				}

				// Tries to find google credentials in all known locations stating with env "GOOGLE_APPLICATION_CREDENTIALS""
				creds, err := google.FindDefaultCredentials(context.TODO())
				if err != nil {
					fmt.Printf("could not find GCP credentials file: %v\n", err)
					os.Exit(1)
				}
				gcpVerifier, err := gcpverifier.NewGcpVerifier(creds, config.debug)
				if err != nil {
					fmt.Printf("could not build GcpVerifier: %v\n", err)
					os.Exit(1)
				}

				gcpVerifier.Logger.Info(context.TODO(), "Using Project ID %s", vei.GCP.ProjectID)
				out := verifier.ValidateEgress(gcpVerifier, vei)
				out.Summary(config.debug)

				if !out.IsSuccessful() {
					gcpVerifier.Logger.Error(context.TODO(), "Failure!")
					os.Exit(1)
				}

				gcpVerifier.Logger.Info(context.TODO(), "Success")
				os.Exit(0)
			}
		},
	}

	validateEgressCmd.Flags().StringVar(&config.platformType, "platform", cloud.AWSClassic.String(), fmt.Sprintf("(optional) infra platform type, which determines which endpoints to test. "+
		"Either '%s', '%s', '%s', or '%s' (hypershift)", cloud.AWSClassic, cloud.GCPClassic, cloud.AWSHCP, cloud.AWSHCPZeroEgress))
	validateEgressCmd.Flags().StringVar(&config.vpcSubnetID, "subnet-id", "", "target subnet ID")
	validateEgressCmd.Flags().StringVar(&config.cloudImageID, "image-id", "", "(optional) cloud image for the compute instance")
	validateEgressCmd.Flags().StringVar(&config.instanceType, "instance-type", "", "(optional) compute instance type")
	validateEgressCmd.Flags().StringVar(&config.cpuArchName, "cpu-arch", "", "(optional) compute instance CPU architecture. Ignored if valid instance-type specified")
	validateEgressCmd.Flags().StringSliceVar(&config.securityGroupIDs, "security-group-ids", []string{}, "(optional) comma-separated list of sec. group IDs to attach to the created EC2 instance. If absent, one will be created")
	validateEgressCmd.Flags().StringVar(&config.egressListLocation, "egress-list-location", "", "(optional) the location of the egress URL list to use. Can either be a local file path or an external URL starting with http(s). This value is ignored for the legacy probe.")
	validateEgressCmd.Flags().StringVar(&config.region, "region", "", fmt.Sprintf("(optional) compute instance region. If absent, environment var %[1]v = %[2]v and %[3]v = %[4]v will be used", awsRegionEnvVarStr, awsRegionDefault, gcpRegionEnvVarStr, gcpRegionDefault))
	validateEgressCmd.Flags().StringToStringVar(&config.cloudTags, "cloud-tags", map[string]string{}, "(optional) comma-separated list of tags to assign to cloud resources e.g. --cloud-tags key1=value1,key2=value2")
	validateEgressCmd.Flags().BoolVar(&config.debug, "debug", false, "(optional) if true, enable additional debug-level logging")
	validateEgressCmd.Flags().DurationVar(&config.timeout, "timeout", time.Duration(0), "(optional) timeout for individual egress verification requests")
	validateEgressCmd.Flags().StringVar(&config.kmsKeyID, "kms-key-id", "", "(optional) ID of KMS key used to encrypt root volumes of compute instances. Defaults to cloud account default key")
	validateEgressCmd.Flags().StringVar(&config.httpProxy, "http-proxy", "", "(optional) http-proxy to be used upon http requests being made by verifier, format: http://user:pass@x.x.x.x:8978")
	validateEgressCmd.Flags().StringVar(&config.httpsProxy, "https-proxy", "", "(optional) https-proxy to be used upon https requests being made by verifier, format: https://user:pass@x.x.x.x:8978")
	validateEgressCmd.Flags().StringVar(&config.CaCert, "cacert", "", "(optional) path to cacert file to be used upon https requests being made by verifier")
	validateEgressCmd.Flags().BoolVar(&config.noTls, "no-tls", false, "(optional) if true, skip client-side SSL certificate validation")
	validateEgressCmd.Flags().StringSliceVar(&config.noProxy, "no-proxy", []string{}, "(optional) comma-separated list of domains or IPs to not pass through the configured http/https proxy e.g. --no-proxy example.com,test.example.com")
	validateEgressCmd.Flags().StringVar(&config.awsProfile, "profile", "", "(optional) AWS profile. If present, any credentials passed with CLI will be ignored")
	validateEgressCmd.Flags().StringVar(&config.gcpVpcName, "vpc-name", "", "(optional unless --platform='gcp') VPC name where GCP cluster is installed")
	validateEgressCmd.Flags().BoolVar(&config.skipAWSInstanceTermination, "skip-termination", false, "(optional) Skip instance termination to allow further debugging")
	validateEgressCmd.Flags().StringVar(&config.terminateDebugInstance, "terminate-debug", "", "(optional) Takes the debug instance ID and terminates it")
	validateEgressCmd.Flags().StringVar(&config.importKeyPair, "import-keypair", "", "(optional) Takes the path to your public key used to connect to Debug Instance. Automatically skips Termination")
	validateEgressCmd.Flags().BoolVar(&config.ForceTempSecurityGroup, "force-temp-security-group", false, "(optional) Enforces creation of Temporary SG even if --security-group-ids flag is used")
	validateEgressCmd.Flags().StringVar(&config.probeName, "probe", "Curl", "(optional) select the probe to be used for egress testing. Either 'Curl' (default) or 'Legacy'")
	if err := validateEgressCmd.MarkFlagRequired("subnet-id"); err != nil {
		validateEgressCmd.PrintErr(err)
	}

	validateEgressCmd.MarkFlagsMutuallyExclusive("cacert", "no-tls")
	return validateEgressCmd
}

func getDefaultRegion(platformType cloud.Platform) string {
	switch platformType {
	case cloud.GCPClassic:
		dRegion, ok := os.LookupEnv(gcpRegionEnvVarStr)
		if !ok {
			return gcpRegionDefault
		}
		return dRegion
	default: // All other platforms, but we assume AWS
		dRegion, ok := os.LookupEnv(awsRegionEnvVarStr)
		if !ok {
			return awsRegionDefault
		}
		return dRegion
	}
}

func getCustomEgressListFromFlag(location string) (string, error) {
	var egressListYaml string
	if _, err := os.Stat(location); err == nil {
		egressListYaml, err = getCustomLocalEgressList(location)
		if err != nil {
			return "", fmt.Errorf("failed to fetch egress URL list from %s: %v", location, err)
		}
		absPath, _ := filepath.Abs(location) // if we've gotten this far, we know the path is valid
		fmt.Printf("Using local egress list from %s\n", absPath)
		return egressListYaml, nil
	}

	parsedUrl, err := url.ParseRequestURI(location)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL %s: %w", location, err)
	}
	egressListYaml, err = getCustomExternalEgressList(parsedUrl.String())
	if err != nil {
		return "", fmt.Errorf("failed to fetch egress URL list from %s: %w", parsedUrl.String(), err)
	}
	fmt.Printf("Using external egress list from %s\n", parsedUrl.String())
	return egressListYaml, nil
}

func getCustomLocalEgressList(filePath string) (string, error) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(file), nil
}

func getCustomExternalEgressList(uri string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return "", err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
