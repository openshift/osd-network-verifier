# Releasing

## Update the machine images (AMIs) used by the verifier
The code we're about to release contains a hardcoded list of IDs referencing public machine images (a.k.a. AMIs) that are used by the verifier to launch VMs/instances into the target subnet. If the machine images referenced by a given verifier release become unavailable, the verifier won't be able to run in all supported regions. Even if still available, old machine images might contain vulnerabilities that pose security/compliance risks. Therefore, it's important that this list is updated with each release. 

**For GCP**, this is relatively straightforward, as the verifier uses the latest official/public global image of RHEL 9. Simply ensure that the image IDs listed near the bottom of [machine_images.go](./pkg/probes/curl/machine_images.go) match what's shown in the [Images tab of the GCE console](https://console.cloud.google.com/compute/images?pageState=(%22images%22:(%22f%22:%22%255B%257B_22k_22_3A_22_22_2C_22t_22_3A10_2C_22v_22_3A_22_5C_22rhel-9-v_5C_22_22%257D_2C%257B_22k_22_3A_22_22_2C_22t_22_3A10_2C_22v_22_3A_22_5C_22OR_5C_22_22_2C_22o_22_3Atrue_2C_22s_22_3Atrue%257D_2C%257B_22k_22_3A_22_22_2C_22t_22_3A10_2C_22v_22_3A_22_5C_22rhel-9-arm64-v_5C_22_22%257D%255D%22))) — look for the images named `rhel-9-v$DATE` for x86 and `rhel-9-arm64-v$DATE` for ARM. If the IDs currently referenced by machine_images.go have been deprecated, please open and thoroughly test a PR updating those IDs before proceeding with a release.

**For AWS**, this process is complicated by the regional nature of AMIs (vs. the global nature of GCP machine images) and the managed policy requirement that we build our own AMIs and host them in our own accounts. Our ["golden AMI" repo](https://gitlab.cee.redhat.com/service/osd-network-verifier-golden-ami) and its associated [Jenkins instance](https://ci.int.devshift.net/job/gl-build-master-osd-network-verifier-golden-ami-packer) is responsible for these builds.

with the latest [most recent stable golden-ami build](https://ci.int.devshift.net/job/gl-build-master-osd-network-verifier-golden-ami-packer/lastStableBuild). 
```bash
AMIS_FROM_JENKINS=$(mktemp)
SRC_MACHINE_IMAGES=pkg/probes/curl/machine_images.go
curl -s https://ci.int.devshift.net/job/gl-build-master-osd-network-verifier-golden-ami-packer/lastStableBuild/consoleText | grep -E -o ami-[[:alnum:]]{17} | sort -u > $AMIS_FROM_JENKINS
test -z $(grep -E -o ami-[[:alnum:]]{17} $SRC_MACHINE_IMAGES | sort -u | comm -23 - $AMIS_FROM_JENKINS | head -n1) && echo "All referenced AMIs are from the latest stable Jenkins build" || echo "WARNING: AMIs referenced in $SRC_MACHINE_IMAGES may be outdated!"
```


## 1. Run unit and integration tests
Each Our CI typically won't allow PRs that fail unit tests to merge
Run our integration test

osd-network-verifier will follow semantic versioning

## Considerations around changes

These changes don't automatically mean a change is a breaking or significant change, but should be taken into consideration:

* The various input verifier structs in [./pkg/verifier/package_verifier.go](./pkg/verifier/package_verifier.go) are exported and consumed downstream. Breaking changes to that input struct should be considered breaking changes for osd-network-verifier.
* New AMIs in [./pkg/probes/curl/machine_images.go](./pkg/probes/curl/machine_images.go), especially as the result of security fixes.
* New cloud IAM requirements/new cloud infrastructure to provision

## Testing changes

For now, this is mostly manual. It's important to validate that these scenarios are working before making a new release:

* The `cloudMachineImageMap` values in [./pkg/probes/curl/machine_images.go](./pkg/probes/curl/machine_images.go) should be updated to match what is in the console output for the [most recent golden-ami build](https://ci.int.devshift.net/job/gl-build-master-osd-network-verifier-golden-ami-packer/).
  * Note - if the above build is broken due to `ResourceLimitExceeded` issues, you will have to clean up the AMI image repository by running the [cleangoldenami module](./cleangoldenami/README.md), and then re-running the Jenkins build.
* Build the `integration` binary by running `go build` from the `/integration` folder. Then use this binary to test both the `aws` and `hostedcluster` configurations as shown below. For more information on setting up integration tests, see the [integration README](./integration/README.md).
  * `./integration --platform aws-classic`
  * `./integration --platform aws-hcp`
* egress test in AWS with a cluster-wide proxy
* ~~egress test on GCP~~ This should be added back when GCP support is functional again

After a new release has been created, please create an MR for the downstream projects to use the latest verifier
version.
The latest version can be fetched with `go get github.com/openshift/osd-network-verifier@<the new tag>`

* Cluster Service (https://gitlab.cee.redhat.com/service/uhc-clusters-service)
* osdctl (https://github.com/openshift/osdctl)
* Configuration Anomaly Detection (https://github.com/openshift/configuration-anomaly-detection)
