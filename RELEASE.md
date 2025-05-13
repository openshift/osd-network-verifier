# Releasing osd-network-verifier
In practice, "the verifier" is a system comprised of three pieces of software: the machine image used by the probe VM/instance (a.k.a. "golden AMI"), the Golang module that handles launching that probe into the subnet-under-test and parsing its output (this repo), and the applications that use this Golang mod like osdctl, uhc-clusters-service/OCM, and CAD ("downstream consumers"). This doc details the release process for all three parts of this system.

## 0. Determine release version number
osd-network-verifier follows [semantic versioning](https://semver.org/spec/v2.0.0.html) (mostly). Most releases should be backwards-compatible and therefore will only bump the minor or patch versions. Releases including breaking changes to the API used by downstream consumers (e.g., the structs in [package_verifier.go](./pkg/verifier/package_verifier.go)) must bump the major version and should be preceded by at least 1 backwards-compatible transitional minor version. Changes that expand the cloud IAM permission set required for existing functionality are considered breaking and warrant a major version bump.

## 1. Update the machine images used by the verifier
Each osd-network-verifier release's Golang module contains hardcoded lists of IDs referencing public machine images (a.k.a. AMIs) that are used by the verifier to launch VMs/instances into the subnet-under-test. If the machine images referenced by a given verifier release become unavailable, the verifier won't be able to run in all supported regions. Even if available, old machine images might contain OS vulnerabilities that pose security/compliance risks. Therefore, it's important that every cloud/architecture combination's AMI list is updated with every verifier release, even if the release contains no probe- or AMI-related changes.

**For GCP**, this is relatively straightforward, as the verifier uses the latest official/public global image of RHEL 9. Simply ensure that the image IDs listed near the bottom of [machine_images.go](./pkg/probes/curl/machine_images.go) match what's shown in the [Images tab of the GCE console](https://console.cloud.google.com/compute/images?pageState=(%22images%22:(%22f%22:%22%255B%257B_22k_22_3A_22_22_2C_22t_22_3A10_2C_22v_22_3A_22_5C_22rhel-9-v_5C_22_22%257D_2C%257B_22k_22_3A_22_22_2C_22t_22_3A10_2C_22v_22_3A_22_5C_22OR_5C_22_22_2C_22o_22_3Atrue_2C_22s_22_3Atrue%257D_2C%257B_22k_22_3A_22_22_2C_22t_22_3A10_2C_22v_22_3A_22_5C_22rhel-9-arm64-v_5C_22_22%257D%255D%22))) — look for the images named `rhel-9-v$DATE` for x86 and `rhel-9-arm64-v$DATE` for ARM. If the IDs currently referenced by machine_images.go have been deprecated, please open and thoroughly test a PR updating those IDs before proceeding with a release.

**For AWS**, this process is complicated by the regional nature of AMIs (vs. the global nature of GCP machine images) and the managed policy requirement that we build our own AMIs and host them in our own accounts. Our ["golden AMI" repo](https://gitlab.cee.redhat.com/service/osd-network-verifier-golden-ami) and its associated [Jenkins instance](https://ci.int.devshift.net/job/gl-build-master-osd-network-verifier-golden-ami-packer) is responsible for these builds. The following instructions show you how to ensure the regional AMI IDs hardcoded into [machine_images.go](./pkg/probes/curl/machine_images.go) are up to date.

First, log into the RH VPN, `cd` into your clone of this repo, and ensure you've pulled the latest `main`. Then check if the AMI IDs referenced by your local copy of [machine_images.go](./pkg/probes/curl/machine_images.go) match the [most recent successful Jenkins build](https://ci.int.devshift.net/job/gl-build-master-osd-network-verifier-golden-ami-packer/lastStableBuild) using the following lines of BASH.
```bash
SRC_MACHINE_IMAGES=pkg/probes/curl/machine_images.go
BUILD_LOG_AMIS=$(mktemp)
[[ "$(curl -s https://ci.int.devshift.net/job/gl-build-master-osd-network-verifier-golden-ami-packer/lastBuild/api/json | jq -r .result)" = "SUCCESS" ]] || echo "WARNING: most recent Jenkins build failed! Falling back to last successful build. Is the build pipeline broken (e.g., due to AMI quota issues)?"
curl -s https://ci.int.devshift.net/job/gl-build-master-osd-network-verifier-golden-ami-packer/lastSuccessfulBuild/consoleText | grep -E -o ami-[[:alnum:]]{17} | sort -u > $BUILD_LOG_AMIS
[[ -z $(grep -E -o ami-[[:alnum:]]{17} $SRC_MACHINE_IMAGES | sort -u | comm -23 - $BUILD_LOG_AMIS | head -n1) ]] && echo "All referenced AMIs are from the latest successful Jenkins build" || echo "WARNING: AMIs referenced in $SRC_MACHINE_IMAGES may be outdated!"
```
If you get any curl errors, make sure you're logged into the RH VPN. If you get an outdated AMI warning, proceed with the steps below. If you get a warning about the last build failing, [investigate it](https://ci.int.devshift.net/job/gl-build-master-osd-network-verifier-golden-ami-packer/lastBuild/), run the [cleangoldenami module](./cleangoldenami/README.md) if necessary, and re-run the Jenkins build. Otherwise, skip to [step 2](#2-perform-pre-release-testing) if all is well.

Obtain credentials for the AWS account that hosts the verifier's AMIs (see [golden AMI README](https://gitlab.cee.redhat.com/service/osd-network-verifier-golden-ami#aws-account)) and set your AWS_PROFILE accordingly. If you're not using a profile, remove `-p $AWS_PROFILE` from the commands below before running them. Run the following BASH to generate Golang snippets containing the latest AMI IDs and to enable [deregistration protection](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/deregister-ami.html#ami-deregistration-protection) for each AMI.
```bash
export AWS_PROFILE=my_example_profile AMI_ACCOUNT_NUM=0123456789 # Replace with real values
export $(osdctl account cli -i $AMI_ACCOUNT_NUM -p $AWS_PROFILE -oenv | xargs)
OUT_DIR=$(mktemp -d) AWS_LOG=$(mktemp)
for REG_AMI_PAIR in $(curl -s https://ci.int.devshift.net/job/gl-build-master-osd-network-verifier-golden-ami-packer/lastSuccessfulBuild/consoleText | grep -E -o "[[:alpha:]]+-[[:alpha:]]+-[[:digit:]]: ami-[[:alnum:]]{17}" | sort -u | tr -d " "); do
  REG=$(echo $REG_AMI_PAIR | cut -d: -f1) AMI=$(echo $REG_AMI_PAIR | cut -d: -f2)
  ARCH=$(aws ec2 describe-images --image-ids=$AMI --region=$REG --query='Images[0].Architecture' --output text)
  echo "\"$REG\": \"$AMI\"," >> $OUT_DIR/$ARCH
  aws ec2 enable-image-deregistration-protection --image-id=$AMI --region=$REG >> $AWS_LOG \
    && echo "$AMI ($ARCH, $REG) protected from deregistration" \
    || echo "ERROR: Failed to protect $AMI ($ARCH, $REG) from deregistration. Do not proceed with release! See $AWS_LOG"
done
echo "Overwrite the region:AMI mappings in machine_images.go with the following Golang snippets..."
for AF in $OUT_DIR/*; do
 echo "...for the slice corresponding to $(basename $AF):"
 cat $AF
done
```
If any errors arise, `Ctrl-C` repeatedly until execution stops, double-check your AWS credentials and VPN connection, and try again.

If the process completes without error, create a new branch and copy-paste the output snippets into machine_images.go, taking care to match up the CPU architectures. Test thoroughly and merge the resulting PR before proceeding with releasing.

## 2. Perform pre-release testing
The CI tests run before PRs are merged are fairly basic and can be bypassed in some situations, so it's important to manually run our unit tests (via `make test`) and our automated [integration test](./integration/README.md) before each release _at the very least_. Please also perform the testing described below if applicable.

If the changes you're releasing...
* apply to HCP clusters, be sure to run the integration test with the `--platform aws-hcp` flag
* enable new regions, perform your own testing in those regions using our [firewalled VPC Terraform examples](./examples/aws/terraform/vpc-firewall)
* could behave differently in proxied networks, perform testing against our proxy Terraform examples ([explicit](./examples/aws/terraform/vpc-proxied-explicit), [implicit](./examples/aws/terraform/vpc-proxied-implicit))
* have anything to do with GCP, perform your own thorough testing on that platform (none of our automated tests/examples cover GCP yet)

# 3. Cut a new release of the Golang module
Ask someone with write permissions on this repo (e.g., team leads) to [create a new release](https://github.com/openshift/osd-network-verifier/releases/new) on GitHub. The auto-generated release notes are usually sufficient.

# 4. Integrate the new Golang module downstream
Once a new release has been tested and published, create a PR/MR updating the osd-network-verifier dependency for each downstream consumer listed below. Usually, this can be as simple as bumping the version number specified in go.mod and then running `go mod tidy && git commit -a -m "Bumped osd-network-verifier" && make test && git push`
* [uhc-clusters-service (OCM)](https://gitlab.cee.redhat.com/service/uhc-clusters-service)
* [osdctl](https://github.com/openshift/osdctl)
* [CAD](https://github.com/openshift/configuration-anomaly-detection)
