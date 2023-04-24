# Release

osd-network-verifier will follow semantic versioning

## Considerations around changes

These changes don't automatically mean a change is a breaking or significant change, but should be taken into consideration:

* The various input verifier structs in [./pkg/verifier/package_verifier.go](./pkg/verifier/package_verifier.go) are exported and consumed downstream. Breaking changes to that input struct should be considered breaking changes for osd-network-verifier.
* New AMIs in [./pkg/verifier/aws/aws_verifier.go](./pkg/verifier/aws/aws_verifier.go), especially as the result of security fixes.
* New cloud IAM requirements/new cloud infrastructure to provision

## Testing changes

For now, this is mostly manual. It's important to validate that these scenarios are working before making a new release:

* The `networkValidatorImage` in [./pkg/verifier/aws/aws_verifier.go](./pkg/verifier/aws/aws_verifier.go) is the same image that is pre-baked on the `defaultAMI`'s.
* [./integration](./integration/) has steps to run the egress test against an AWS account to test that the osd-network-verifier is able to run using the pre-baked image on the default AMI.
* egress test in AWS with a cluster-wide proxy
* ~~egress test on GCP~~ This should be added back when GCP support is functional again

After a new release has been created, please create an MR for the downstream projects to use the latest verifier version:

* Cluster Service (https://gitlab.cee.redhat.com/service/uhc-clusters-service): After cloning the repo, do `go get github.com/openshift/osd-network-verifier@<the new tag>`
* osdctl (https://github.com/openshift/osdctl)
