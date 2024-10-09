# osd-network-verifier

A cli tool and set of libraries that
verify the pre-configured networking components
for ROSA and OSD CCS clusters.

## Overview

osd-network-verifier can be used prior to or after the installation
of osd/rosa clusters to ensure the network configuration
is correctly set up per OSD requirements listed on https://docs.openshift.com/container-platform/4.6/installing/installing_aws/installing-aws-vpc.html#installation-custom-aws-vpc-requirements_installing-aws-vpc

It currently verifies:
- Egress from VPC subnets to [essential OSD domains](https://docs.openshift.com/rosa/rosa_install_access_delete_clusters/rosa_getting_started_iam/rosa-aws-prereqs.html#osd-aws-privatelink-firewall-prerequisites_prerequisites)
- DNS resolution in a [VPC](https://docs.openshift.com/container-platform/4.10/installing/installing_aws/installing-aws-vpc.html)

The recommended workflow of diagnostic use of ONV is shown in the following flow diagram:

![shift](https://user-images.githubusercontent.com/87340776/168323039-ec5269a8-2cf9-44db-ab5f-e490c88d4342.jpg)

## Cloud Provider Specific READMEs
-  [AWS](docs/aws/aws.md)
-  [GCP](docs/gcp/gcp.md)

### Building
`make build`: Builds `osd-network-verifier` executable in base directory

## Contributing and Maintenance
If interested, please fork this repo and create pull requests to the `main` branch.

### Egress Lists

This lists of essential domains for egress verification should be maintained in [pkg/data/egress_lists](https://github.com/openshift/osd-network-verifier/tree/main/pkg/data/egress_lists). The network verifier will dynamically pull down the list of endpoints from the most recent commit. This means that egress lists can be updated quickly without the need of a new osd-network-verifier release.

Network-verifier knows which list to pull from by using the [platform interface](./pkg/data/cloud/platform.go). For example, if the AWSClassic platform type is used, network-verifier will pull down the egress list associated with that platform type.

It is also possible to pass in a custom list of egress endpoints by using the `--egress-list-location` flag.

### Probes
Probes within the verifier are responsible for a number of important tasks.
These include the following:
- determining which machine images are to be used
- parsing cloud instance console output
- configuring instructions to the cloud instance

Probes are cloud-platform-agnostic by design,
meaning that their implementations are not specific to any one cloud provider.
All probes must honor the contract defined by the [base probe interface](./pkg/probes/package_probes.go).
By default, the verifier uses the [curl probe](./pkg/probes/curl/curl_json.go).

#### Image Selection

Each probe is responsible for determining its list of approved machine images.
The list of images (RHEL base images) that osd-network-verifier selects
from to run in is maintained in `pkg/probes/<probe_name>/machine_images.go`.
Which image is selected is based on the platform, region and cpu architecture type.
By default, "X86" is used unless manually overridden by the `--cpu-arch` flag.

### IAM Permission Requirement List

Version ID [required for IAM permissions](https://github.com/openshift/osd-network-verifier/blob/main/docs/aws/aws.md#iam-permissions) may need update to match specification in [AWS docs](https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_elements_version.html).

### Terraform Scripts (AWS-only)

The Terraform scripts in this repository's (under `/examples/aws/terraform/`) allow you to quickly deploy temporary AWS VPCs for testing the network verifier against several common network scenarios. See each subdirectory's README for more details and usage instructions:
- [VPC with no firewall](examples/aws/terraform/vpc/README.md)
- [VPC with an egress firewall](examples/aws/terraform/vpc-firewall/README.md)
- [VPC with an explicit proxy server](examples/aws/terraform/vpc-proxied-explicit/README.md)
- [VPC with a transparent proxy server](examples/aws/terraform/vpc-proxied-transparent/README.md)

## Interface ##

### Platform ###
The [platform struct type](./pkg/data/cloud/platform.go) is used to inform network-verifier of the platform type it is running on (AWSClassic, GCPClassic, etc) and can be referred to by supported aliases. For example, "aws" and "aws-classic" are both mapped to "AWSClassic". These platform types are used to determine information such as which egress verification list, machine type, and cpu type to use.
```
type Platform struct {
	// names holds 3 unique lowercase names of the Platform (e.g., "aws"). We use a fixed-
	// size array so that this struct remains comparable. Any of the 3 values can be used to refer
	// to this specific Platform via Platform.ByName(), but only the first (element
	// 0) element will be the "preferred name" returned by Platform.String()
	names [3]string
}
```

Currently network-verifier supports four implementations for Platform types.
- AWSClassic
- AWSHCP
- AWSHCPZeroEgress
- GCPClassic

Network-verifier uses these supported platform types to determine information such as which egress verification list, machine type, and cpu type to use.

## Release Process

See [RELEASE.md](./RELEASE.md)