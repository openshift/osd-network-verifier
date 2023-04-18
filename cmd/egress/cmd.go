package egress

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/openshift/osd-network-verifier/cmd/utils"
	"github.com/openshift/osd-network-verifier/pkg/helpers"
	"github.com/openshift/osd-network-verifier/pkg/proxy"
	"github.com/openshift/osd-network-verifier/pkg/verifier"
	gcpverifier "github.com/openshift/osd-network-verifier/pkg/verifier/gcp"
	"golang.org/x/oauth2/google"

	"github.com/spf13/cobra"
)

var (
	awsDefaultTags      = map[string]string{"osd-network-verifier": "owned", "red-hat-managed": "true", "Name": "osd-network-verifier"}
	gcpDefaultTags      = map[string]string{"osd-network-verifier": "owned", "red-hat-managed": "true", "name": "osd-network-verifier"}
	awsRegionEnvVarStr  = "AWS_REGION"
	awsRegionDefault    = "us-east-2"
	gcpRegionEnvVarStr  = "GCP_REGION"
	gcpRegionDefault    = "us-east1"
	platformTypeDefault = helpers.PlatformAWS
)

type egressConfig struct {
	vpcSubnetID                string
	cloudImageID               string
	instanceType               string
	securityGroupId            string
	cloudTags                  map[string]string
	debug                      bool
	region                     string
	timeout                    time.Duration
	kmsKeyID                   string
	httpProxy                  string
	httpsProxy                 string
	CaCert                     string
	noTls                      bool
	platformType               string
	awsProfile                 string
	gcpVpcName                 string
	skipAWSInstanceTermination bool
	terminateDebugInstance     string
}

func getDefaultRegion(platformType string) string {

	if platformType == helpers.PlatformGCP {
		//gcp region
		dRegion, ok := os.LookupEnv(gcpRegionEnvVarStr)
		if !ok {
			return gcpRegionDefault
		}
		return dRegion
	}
	//aws region
	dRegion, ok := os.LookupEnv(awsRegionEnvVarStr)
	if !ok {
		return awsRegionDefault
	}
	return dRegion
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
./osd-network-verifier egress --subnet-id ${SUBNET_ID} --security-group-id ${SECURITY_GROUP}`,
		Run: func(cmd *cobra.Command, args []string) {

			// Set Region
			if config.region == "" {
				config.region = getDefaultRegion(config.platformType)
			}

			// Set Up Proxy
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
			}

			// setup non cloud config options
			vei := verifier.ValidateEgressInput{
				Ctx:          context.TODO(),
				SubnetID:     config.vpcSubnetID,
				CloudImageID: config.cloudImageID,
				Timeout:      config.timeout,
				Tags:         config.cloudTags,
				InstanceType: config.instanceType,
				PlatformType: config.platformType,
				Proxy:        p,
			}

			// AWS workflow
			if config.platformType == helpers.PlatformAWS || config.platformType == helpers.PlatformHostedCluster {

				if len(vei.Tags) == 0 {
					vei.Tags = awsDefaultTags
				}

				//Setup AWS Specific Configs
				vei.AWS = verifier.AwsEgressConfig{
					KmsKeyID:        config.kmsKeyID,
					SecurityGroupId: config.securityGroupId,
				}

				awsVerifier, err := utils.GetAwsVerifier(config.region, config.awsProfile, config.debug)
				if err != nil {
					fmt.Printf("could not build awsVerifier %v\n", err)
					os.Exit(1)
				}

				awsVerifier.Logger.Warn(context.TODO(), "Using region: %s", config.region)

				vei.SkipInstanceTermination = config.skipAWSInstanceTermination
				vei.TerminateDebugInstance = config.terminateDebugInstance

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
			if config.platformType == helpers.PlatformGCP {

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

			// Unknown platformType specified
			fmt.Printf("unknown platform type '%v'\n", config.platformType)
			os.Exit(1)
		},
	}

	validateEgressCmd.Flags().StringVar(&config.platformType, "platform", platformTypeDefault, fmt.Sprintf("(optional) infra platform type, which determines which endpoints to test. Either '%v', '%v', or '%v' (hypershift)", helpers.PlatformAWS, helpers.PlatformGCP, helpers.PlatformHostedCluster))
	validateEgressCmd.Flags().StringVar(&config.vpcSubnetID, "subnet-id", "", "source subnet ID")
	validateEgressCmd.Flags().StringVar(&config.cloudImageID, "image-id", "", "(optional) cloud image for the compute instance")
	validateEgressCmd.Flags().StringVar(&config.instanceType, "instance-type", "", "(optional) compute instance type")
	validateEgressCmd.Flags().StringVar(&config.securityGroupId, "security-group-id", "", "security group ID to attach to the created EC2 instance")
	validateEgressCmd.Flags().StringVar(&config.region, "region", "", fmt.Sprintf("(optional) compute instance region. If absent, environment var %[1]v = %[2]v and %[3]v = %[4]v will be used", awsRegionEnvVarStr, awsRegionDefault, gcpRegionEnvVarStr, gcpRegionDefault))
	validateEgressCmd.Flags().StringToStringVar(&config.cloudTags, "cloud-tags", map[string]string{}, "(optional) comma-seperated list of tags to assign to cloud resources e.g. --cloud-tags key1=value1,key2=value2")
	validateEgressCmd.Flags().BoolVar(&config.debug, "debug", false, "(optional) if true, enable additional debug-level logging")
	validateEgressCmd.Flags().DurationVar(&config.timeout, "timeout", 2*time.Second, "(optional) timeout for individual egress verification requests")
	validateEgressCmd.Flags().StringVar(&config.kmsKeyID, "kms-key-id", "", "(optional) ID of KMS key used to encrypt root volumes of compute instances. Defaults to cloud account default key")
	validateEgressCmd.Flags().StringVar(&config.httpProxy, "http-proxy", "", "(optional) http-proxy to be used upon http requests being made by verifier, format: http://user:pass@x.x.x.x:8978")
	validateEgressCmd.Flags().StringVar(&config.httpsProxy, "https-proxy", "", "(optional) https-proxy to be used upon https requests being made by verifier, format: https://user:pass@x.x.x.x:8978")
	validateEgressCmd.Flags().StringVar(&config.CaCert, "cacert", "", "(optional) path to cacert file to be used upon https requests being made by verifier")
	validateEgressCmd.Flags().BoolVar(&config.noTls, "no-tls", false, "(optional) if true, skip client-side SSL certificate validation")
	validateEgressCmd.Flags().StringVar(&config.awsProfile, "profile", "", "(optional) AWS profile. If present, any credentials passed with CLI will be ignored")
	validateEgressCmd.Flags().StringVar(&config.gcpVpcName, "vpc-name", "", "(optional unless --platform='gcp') VPC name where GCP cluster is installed")
	validateEgressCmd.Flags().BoolVar(&config.skipAWSInstanceTermination, "skip-termination", false, "(optional) Skip instance termination to allow further debugging")
	validateEgressCmd.Flags().StringVar(&config.terminateDebugInstance, "terminate-debug", "", "(optional) Takes the debug instance ID and terminates it")

	if err := validateEgressCmd.MarkFlagRequired("subnet-id"); err != nil {
		validateEgressCmd.PrintErr(err)
	}

	if err := validateEgressCmd.MarkFlagRequired("security-group-id"); err != nil {
		validateEgressCmd.PrintErr(err)
	}

	return validateEgressCmd
}
