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

It is also possible to pass in a custom list of egress endpoints by using the `--egress-list-location` flag.

Newly-added lists should be registered as "platform types" in [`helpers.go`](pkg/helpers/helpers.go#L94) using the list file's extensionless name as the value (e.g., abc.yaml should be registered as `PlatformABC     string = "abc"`). Finally, the `--platform` help message and value handling logic in [`cmd.go`](cmd/egress/cmd.go) should also be updated.

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

## Release Process

See [RELEASE.md](./RELEASE.md)