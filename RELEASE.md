# Release

osd-network-verifier will follow semantic versioning

## Considerations around changes

These changes don't automatically mean a change is a breaking or significant change, but should be taken into consideration:

* The various input verifier structs in [./pkg/verifier/package_verifier.go](./pkg/verifier/package_verifier.go) are exported and consumed downstream. Breaking changes to that input struct should be considered breaking changes for osd-network-verifier.
* New AMIs in [./pkg/verifier/aws/aws_verifier.go](./pkg/verifier/aws/aws_verifier.go), especially as the result of security fixes.
* New cloud IAM requirements/new cloud infrastructure to provision

## Testing changes

For now, this is mostly manual. It's important to validate that these scenarios are working before making a new release:

* The `defaultAmi` mapping values in [/pkg/verifier/aws/aws_verifier.go](./pkg/verifier/aws/aws_verifier.go) should be updated to match what is in the console output for the [most recent golden-ami build](https://ci.int.devshift.net/job/gl-build-master-osd-network-verifier-golden-ami-packer/).
  * Note - if the above build is broken due to `ResourceLimitExceeded` issues, you will have to manually clean up the AMI image repository by following [this SOP](https://github.com/openshift/ops-sop/blob/master/v4/howto/network-verifier/clean-golden-ami.md), and then re-running the Jenkins build.
* The `networkValidatorImage` in [./pkg/verifier/aws/aws_verifier.go](./pkg/verifier/aws/aws_verifier.go) is the same image that is pre-baked on the `defaultAMI`'s. This can be found by looking at the latest tagged image in the [osd-network-verifier quay repository](https://quay.io/repository/app-sre/osd-network-verifier?tab=tags&tag=latest).
* Build the `integration` binary by running `go build` from the `/integration` folder. Then use this binary to test both the `aws` and `hostedcluster` configurations as shown below. For more information on setting up integration tests, see the [integration README](./integration/README.md).
  * `./integration --platform aws`
  * `./integration --platform hostedcluster`
* egress test in AWS with a cluster-wide proxy
* ~~egress test on GCP~~ This should be added back when GCP support is functional again

After a new release has been created, please create an MR for the downstream projects to use the latest verifier version:

* Cluster Service (https://gitlab.cee.redhat.com/service/uhc-clusters-service): After cloning the repo, do `go get github.com/openshift/osd-network-verifier@<the new tag>`
* osdctl (https://github.com/openshift/osdctl)
