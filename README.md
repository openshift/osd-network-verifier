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
- BYOVPC config requirements

The recommended workflow of diagnostic use of ONV is shown in the following flow diagram:

![shift](https://user-images.githubusercontent.com/87340776/168323039-ec5269a8-2cf9-44db-ab5f-e490c88d4342.jpg)

## Cloud Provider Specific READMEs
-  [AWS](docs/aws/aws.md)
-  [GCP](docs/gcp/gcp.md)

### Building
`make build`: Builds `osd-network-verifier` executable in base directory

## Contributing and Maintenance
If interested, please fork this repo and create pull requests to the `main` branch.

### Golden AMI
osd-network-verifier depends on these publicly available [AMIs](https://github.com/openshift/osd-network-verifier/blob/612be8c5b0ef8ac01e8018eca15dd143ab31cd1f/pkg/verifier/aws/aws_verifier.go#L25-L45) built from the [osd-network-verifier-golden-ami](https://gitlab.cee.redhat.com/service/osd-network-verifier-golden-ami) repo.

Golden AMI provides the following:
- runtime environment setup (such as container engine, configurations, etc)
- building and embedding the validator binary which performs the individual checks to the endpoints

### Egress List

This list of essential domains for egress verification should be maintained in [gitlab repo](https://gitlab.cee.redhat.com/service/osd-network-verifier-golden-ami/-/blob/master/build/config/config.yaml).

### IAM Permission Requirement List

Version ID [required for IAM support role](docs/aws/aws.md#iam-support-role) may need update to match specification in [AWS docs](https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_elements_version.html).

## Release Process

See [RELEASE.md](./RELEASE.md)