# Releasing osd-network-verifier
In practice, "the verifier" is a system comprised of three pieces of software: the machine image used by the probe VM/instance (a.k.a. "golden AMI"), the Golang module that handles launching that probe into the subnet-under-test and parsing its output (this repo), and the applications that use this Golang mod like osdctl, uhc-clusters-service/OCM, and CAD ("downstream consumers"). This doc details our release process, which touches all three parts.

## 0. Versioning
These changes don't automatically mean a change is a breaking or significant change, but should be taken into consideration:

* The various input verifier structs in [./pkg/verifier/package_verifier.go](./pkg/verifier/package_verifier.go) are exported and consumed downstream. Breaking changes to that input struct should be considered breaking changes for osd-network-verifier.
* New AMIs in [./pkg/probes/curl/machine_images.go](./pkg/probes/curl/machine_images.go), especially as the result of security fixes.
* New cloud IAM requirements/new cloud infrastructure to provision

## 1. Update the machine images used by the verifier
The code we're about to release contains hardcoded lists of IDs referencing public machine images (a.k.a. AMIs) that are used by the verifier to launch VMs/instances into the subnet-under-test. If the machine images referenced by a given verifier release become unavailable, the verifier won't be able to run in all supported regions. Even if available, old machine images might contain OS vulnerabilities that pose security/compliance risks. Therefore, it's important that every cloud/architecture combination's AMI list is updated with every verifier release, even if the release contains no probe- or AMI-related changes.

**For GCP**, this is relatively straightforward, as the verifier uses the latest official/public global image of RHEL 9. Simply ensure that the image IDs listed near the bottom of [machine_images.go](./pkg/probes/curl/machine_images.go) match what's shown in the [Images tab of the GCE console](https://console.cloud.google.com/compute/images?pageState=(%22images%22:(%22f%22:%22%255B%257B_22k_22_3A_22_22_2C_22t_22_3A10_2C_22v_22_3A_22_5C_22rhel-9-v_5C_22_22%257D_2C%257B_22k_22_3A_22_22_2C_22t_22_3A10_2C_22v_22_3A_22_5C_22OR_5C_22_22_2C_22o_22_3Atrue_2C_22s_22_3Atrue%257D_2C%257B_22k_22_3A_22_22_2C_22t_22_3A10_2C_22v_22_3A_22_5C_22rhel-9-arm64-v_5C_22_22%257D%255D%22))) — look for the images named `rhel-9-v$DATE` for x86 and `rhel-9-arm64-v$DATE` for ARM. If the IDs currently referenced by machine_images.go have been deprecated, please open and thoroughly test a PR updating those IDs before proceeding with a release.

**For AWS**, this process is complicated by the regional nature of AMIs (vs. the global nature of GCP machine images) and the managed policy requirement that we build our own AMIs and host them in our own accounts. Our ["golden AMI" repo](https://gitlab.cee.redhat.com/service/osd-network-verifier-golden-ami) and its associated [Jenkins instance](https://ci.int.devshift.net/job/gl-build-master-osd-network-verifier-golden-ami-packer) is responsible for these builds. The following instructions show you how to ensure the regional AMI IDs hardcoded into [machine_images.go](./pkg/probes/curl/machine_images.go) are up to date.

First, log into the RH VPN, `cd` into your clone of this repo, and ensure you've pulled the latest `main`. Then check if the AMI IDs referenced by your local copy of [machine_images.go](./pkg/probes/curl/machine_images.go) match the [most recent successful Jenkins build](https://ci.int.devshift.net/job/gl-build-master-osd-network-verifier-golden-ami-packer/lastStableBuild) using the following lines of BASH.
```bash
SRC_MACHINE_IMAGES=pkg/probes/curl/machine_images.go
BUILD_LOG_AMIS=$(mktemp)
curl -s https://ci.int.devshift.net/job/gl-build-master-osd-network-verifier-golden-ami-packer/lastSuccessfulBuild/consoleText | grep -E -o ami-[[:alnum:]]{17} | sort -u > $BUILD_LOG_AMIS
test -z $(grep -E -o ami-[[:alnum:]]{17} $SRC_MACHINE_IMAGES | sort -u | comm -23 - $BUILD_LOG_AMIS | head -n1) && echo "All referenced AMIs are from the latest successful Jenkins build" || echo "WARNING: AMIs referenced in $SRC_MACHINE_IMAGES may be outdated!"
```
If you get any curl errors, make sure you're logged into the RH VPN. If you get an outdated AMI warning, proceed with the steps below. Otherwise, skip to the next section.

Run the following BASH to generate Golang snippets containing the latest AMI IDs. It will take 1-3 minutes, and you'll need to set an `AWS_PROFILE` corresponding to an AWS account that's enabled for all the same regions as the verifier (`rhcontrol` works well for this, as shown below).
```bash
AWS_PROFILE=rhcontrol
OUT_DIR=$(mktemp -d)
for REG_AMI_PAIR in $(curl -s https://ci.int.devshift.net/job/gl-build-master-osd-network-verifier-golden-ami-packer/lastSuccessfulBuild/consoleText | grep -E -o "[[:alpha:]]+-[[:alpha:]]+-[[:digit:]]: ami-[[:alnum:]]{17}" | sort -u | tr -d " "); do
  REG=$(echo $REG_AMI_PAIR | cut -d: -f1) AMI=$(echo $REG_AMI_PAIR | cut -d: -f2)
  ARCH=$(aws ec2 describe-images --image-ids=$AMI_ID --region=$REGION --query='Images[0].Architecture' --output text --profile=$AWS_PROFILE)
  echo "\"$REG\": \"$AMI\"," >> $OUT_DIR/$ARCH
done
echo "Overwrite the region:AMI mappings in machine_images.go with the following Golang snippets..."
for AF in $OUT_DIR/*; do
 echo "...for the slice corresponding to $(basename $AF):"
 cat $AF
done
```
Create a new branch and copy-paste the snippets into machine_images.go, taking care to match up the CPU architectures. Test thoroughly and merge the resulting PR before proceeding with releasing.
## 2. Cut a release of the Golang module
### Run unit and integration tests
Each Our CI typically won't allow PRs that fail unit tests to merge
Run our integration test

osd-network-verifier will follow semantic versioning


### Testing changes

For now, this is mostly manual. It's important to validate that these scenarios are working before making a new release:

* The `cloudMachineImageMap` values in [./pkg/probes/curl/machine_images.go](./pkg/probes/curl/machine_images.go) should be updated to match what is in the console output for the [most recent golden-ami build](https://ci.int.devshift.net/job/gl-build-master-osd-network-verifier-golden-ami-packer/).
  * Note - if the above build is broken due to `ResourceLimitExceeded` issues, you will have to clean up the AMI image repository by running the [cleangoldenami module](./cleangoldenami/README.md), and then re-running the Jenkins build.
* Build the `integration` binary by running `go build` from the `/integration` folder. Then use this binary to test both the `aws` and `hostedcluster` configurations as shown below. For more information on setting up integration tests, see the [integration README](./integration/README.md).
  * `./integration --platform aws-classic`
  * `./integration --platform aws-hcp`
* egress test in AWS with a cluster-wide proxy
* egress test on GCP

After a new release has been created, please create an MR for the downstream projects to use the latest verifier
version.
The latest version can be fetched with `go get github.com/openshift/osd-network-verifier@<the new tag>`

* Cluster Service (https://gitlab.cee.redhat.com/service/uhc-clusters-service)
* osdctl (https://github.com/openshift/osdctl)
* Configuration Anomaly Detection (https://github.com/openshift/configuration-anomaly-detection)
