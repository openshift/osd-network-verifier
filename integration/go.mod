module github.com/openshift/osd-network-verifier/integration

go 1.19

require (
	github.com/aws/aws-sdk-go-v2 v1.17.6
	github.com/aws/aws-sdk-go-v2/config v1.18.18
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.90.0
	github.com/aws/aws-sdk-go-v2/service/networkfirewall v1.24.6
	github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi v1.14.6
	github.com/aws/smithy-go v1.13.5
	github.com/jmespath/go-jmespath v0.4.0
	github.com/openshift-online/ocm-sdk-go v0.1.325
	github.com/openshift/osd-network-verifier v0.1.1-0.20230307184731-c061a2224398
)

replace github.com/openshift/osd-network-verifier => ../

require (
	github.com/aws/aws-sdk-go-v2/credentials v1.13.17 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.13.0 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.30 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.24 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.31 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.24 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.12.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.14.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.18.6 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator v9.31.0+incompatible // indirect
	github.com/golang/glog v1.1.0 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
