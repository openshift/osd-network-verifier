package egress

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/credentials"
	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/cloudclient"
	"github.com/openshift/osd-network-verifier/pkg/proxy"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2/google"
)

var (
	defaultTags               = map[string]string{"osd-network-verifier": "owned", "red-hat-managed": "true", "name": "osd-network-verifier"}
	awsRegionEnvVarStr string = "AWS_REGION"
	awsRegionDefault   string = "us-east-2"
	gcpRegionEnvVarStr string = "GCP_REGION"
	gcpRegionDefault   string = "us-east1"
)

type egressConfig struct {
	vpcSubnetID  string
	cloudImageID string
	instanceType string
	cloudTags    map[string]string
	debug        bool
	region       string
	timeout      time.Duration
	kmsKeyID     string
	httpProxy    string
	httpsProxy   string
	CaCert       string
	noTls        bool
	gcp          bool
	awsProfile   string
}

func getDefaultRegion(cloudProvider string) string {
	if cloudProvider != "gcp" {
		//aws region
		val, present := os.LookupEnv(awsRegionEnvVarStr)
		if present {
			return val
		} else {
			return awsRegionDefault
		}
	} else {
		//gcp region
		val, present := os.LookupEnv(gcpRegionEnvVarStr)
		if present {
			return val
		} else {
			return gcpRegionDefault
		}
	}

}

func NewCmdValidateEgress() *cobra.Command {
	config := egressConfig{}

	validateEgressCmd := &cobra.Command{
		Use:   "egress",
		Short: "Verify essential openshift domains are reachable from given subnet ID.",
		Long:  `Verify essential openshift domains are reachable from given subnet ID.`,
		Example: `For AWS, ensure your credential environment vars 
AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY (also AWS_SESSION_TOKEN for STS credentials) 
are set correctly before execution.

# Verify that essential openshift domains are reachable from a given SUBNET_ID
./osd-network-verifier egress --subnet-id $(SUBNET_ID) --image-id $(IMAGE_ID)`,
		Run: func(cmd *cobra.Command, args []string) {
			// ctx
			ctx := context.TODO()

			// Create logger
			builder := ocmlog.NewStdLoggerBuilder()
			builder.Debug(config.debug)
			logger, err := builder.Build()
			if err != nil {
				fmt.Printf("Unable to build logger: %s\n", err.Error())
				os.Exit(1)
			}

			var creds interface{}

			if !config.gcp {
				//AWS stuff
				if config.region == "" {
					config.region = getDefaultRegion("aws")
				}
				//default aws machine t3
				if config.instanceType == "" {
					config.instanceType = "t3.micro"
				}
				if config.awsProfile != "" {
					creds = config.awsProfile
					logger.Info(ctx, "Using AWS profile: %s", config.awsProfile)
				} else {
					creds = credentials.NewStaticCredentialsProvider(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), os.Getenv("AWS_SESSION_TOKEN"))
				}
				if err != nil {
					logger.Error(ctx, err.Error())
					os.Exit(1)
				}
			} else {
				// GCP stuff
				if config.region == "" {
					config.region = getDefaultRegion("gcp")
				}
				if os.Getenv("GCP_VPC_NAME") == "" {
					logger.Error(ctx, "please set environment variable GCP_VPC_NAME to the name of VPC")
					os.Exit(1)
				}

				if os.Getenv("GCP_PROJECT_ID") == "" {
					logger.Error(ctx, "please set environment variable GCP_PROJECT_ID to the project ID of VPC")
					os.Exit(1)
				}
				creds = &google.Credentials{ProjectID: os.Getenv("GCP_PROJECT_ID")}

				if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
					logger.Info(ctx, "GOOGLE_APPLICATION_CREDENTIALS not set; using service account attached to %s", os.Getenv("GCP_PROJECT_ID"))
				} else {
					logger.Info(ctx, "Using GCP credential json file from %s", os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
				}
				//default gcp machine e2
				if config.instanceType == "" {
					config.instanceType = "e2-standard-2"
				}
				logger.Info(ctx, "Using Project ID %s", os.Getenv("GCP_PROJECT_ID"))
			}

			logger.Info(ctx, "Using region: %s", config.region)

			cli, err := cloudclient.NewClient(ctx, logger, creds, config.region, config.instanceType, config.cloudTags)
			if err != nil {
				logger.Error(ctx, err.Error())
				os.Exit(1)
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

			out := cli.ValidateEgress(ctx, config.vpcSubnetID, config.cloudImageID, config.kmsKeyID, config.timeout, p)

			out.Summary()
			if !out.IsSuccessful() {
				logger.Error(ctx, "Failure!")
				os.Exit(1)
			}

			logger.Info(ctx, "Success")
		},
	}

	validateEgressCmd.Flags().StringVar(&config.vpcSubnetID, "subnet-id", "", "source subnet ID")
	validateEgressCmd.Flags().StringVar(&config.cloudImageID, "image-id", "", "(optional) cloud image for the compute instance")
	validateEgressCmd.Flags().StringVar(&config.instanceType, "instance-type", "", "(optional) compute instance type")
	validateEgressCmd.Flags().StringVar(&config.region, "region", "", fmt.Sprintf("(optional) compute instance region. If absent, environment var %[1]v = %[2]v and %[3]v = %[4]v will be used", awsRegionEnvVarStr, awsRegionDefault, gcpRegionEnvVarStr, gcpRegionDefault))
	validateEgressCmd.Flags().StringToStringVar(&config.cloudTags, "cloud-tags", defaultTags, "(optional) comma-seperated list of tags to assign to cloud resources e.g. --cloud-tags key1=value1,key2=value2")
	validateEgressCmd.Flags().BoolVar(&config.debug, "debug", false, "(optional) if true, enable additional debug-level logging")
	validateEgressCmd.Flags().DurationVar(&config.timeout, "timeout", 2*time.Second, "(optional) timeout for individual egress verification requests")
	validateEgressCmd.Flags().StringVar(&config.kmsKeyID, "kms-key-id", "", "(optional) ID of KMS key used to encrypt root volumes of compute instances. Defaults to cloud account default key")
	validateEgressCmd.Flags().StringVar(&config.httpProxy, "http-proxy", "", "(optional) http-proxy to be used upon http requests being made by verifier, format: http://user:pass@x.x.x.x:8978")
	validateEgressCmd.Flags().StringVar(&config.httpsProxy, "https-proxy", "", "(optional) https-proxy to be used upon https requests being made by verifier, format: https://user:pass@x.x.x.x:8978")
	validateEgressCmd.Flags().StringVar(&config.CaCert, "cacert", "", "(optional) path to cacert file to be used upon https requests being made by verifier")
	validateEgressCmd.Flags().BoolVar(&config.noTls, "no-tls", false, "(optional) if true, ignore all ssl certificate validations on client-side.")
	validateEgressCmd.Flags().BoolVar(&config.gcp, "gcp", false, "Set to true if cluster is GCP")
	validateEgressCmd.Flags().StringVar(&config.awsProfile, "profile", "", "(optional) AWS profile. If present, any credentials passed with CLI will be ignored.")

	if err := validateEgressCmd.MarkFlagRequired("subnet-id"); err != nil {
		validateEgressCmd.PrintErr(err)
		os.Exit(1)
	}

	return validateEgressCmd

}
