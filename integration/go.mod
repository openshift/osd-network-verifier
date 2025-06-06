module github.com/openshift/osd-network-verifier/integration

go 1.23.0

toolchain go1.23.6

require (
	github.com/aws/aws-sdk-go-v2 v1.32.2
	github.com/aws/aws-sdk-go-v2/config v1.28.0
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.184.0
	github.com/aws/aws-sdk-go-v2/service/networkfirewall v1.43.2
	github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi v1.25.2
	github.com/aws/smithy-go v1.22.0
	github.com/jmespath/go-jmespath v0.4.0
	github.com/openshift-online/ocm-sdk-go v0.1.446
	github.com/openshift/osd-network-verifier v1.1.2
)

replace github.com/openshift/osd-network-verifier => ../

require (
	github.com/aws/aws-sdk-go-v2/credentials v1.17.41 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.24.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.28.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.32.2 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator v9.31.0+incompatible // indirect
	github.com/golang/glog v1.2.2 // indirect
	github.com/google/go-github/v63 v63.0.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/stretchr/testify v1.9.0 // indirect
	golang.org/x/net v0.38.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
