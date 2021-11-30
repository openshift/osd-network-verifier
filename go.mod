module github.com/openshift/osd-network-verifier

go 1.16

require (
	github.com/aws/aws-sdk-go v1.41.19
	github.com/aws/aws-sdk-go-v2 v1.11.1
	github.com/aws/aws-sdk-go-v2/config v1.10.3
	github.com/aws/aws-sdk-go-v2/credentials v1.6.3
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.24.0
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/openshift/api v0.0.0-20211108165917-be1be0e89115
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	golang.org/x/oauth2 v0.0.0-20210402161424-2e8d93401602
	golang.org/x/sys v0.0.0-20210817190340-bfb29a6856f2 // indirect
	google.golang.org/api v0.44.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/apimachinery v0.22.3
	k8s.io/cli-runtime v0.22.3
)
