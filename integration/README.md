# osd-network-verifier integration test

This houses an integration test that will:

1. Create a BYOVPC with an AWS Network Firewall blocking quay.io following this architecture: 
https://docs.aws.amazon.com/network-firewall/latest/developerguide/arch-igw-ngw.html.
2. Run osd-network-verifier's egress check. It should run using the pre-pulled AMI and be able to validate that quay.io is blocked even though it cannot pull the desired container image hosted on quay.io.
3. Delete the BYOVPC and AWS Network Firewall created in step 1

You can consider this test "passed" if no runtime errors occur and the output contains something along the lines of the following blurb.
```
printing out failures:
 - egressURL error: cdn03.quay.io:443
 - egressURL error: cdn01.quay.io:443
 - egressURL error: cdn02.quay.io:443
 - egressURL error: quay.io:443
```

## Usage

It can be run with AWS credentials setup as environment variables beforehand, e.g.

```bash
DEV_ACCOUNT_ID=""
REGION=""
PROBE="curl" # or "legacy"
export $(osdctl account cli -i ${DEV_ACCOUNT_ID}-p osd-staging-2 -r ${REGION} -oenv | xargs)
go build .
./integration --region=$REGION --probe=$PROBE
```

Or with a profile:

```bash
go build .
./integration --region=$REGION --probe=$PROBE --profile=$PROFILE
```

Other flags include:
* `--probe`: select which Probe to use (e.g., "curl" or "legacy") (mandatory)
* `--debug`: enable verbose logging
* `--platform`: set the value to `Platform` that's passed into `ValidateEgress()` (default: "aws")
* `--create-only`: only create infrastructure without deleting it or running the egress test
* `--delete-only`: delete infrastructure left behind by a failed test (or by the above option) without running the egress test

TODO: Create a minimal IAM policy. The required permissions are laid out in:
`pkg/aws/ec2.go`, `pkg/aws/networkfirewall.go`, and `pkg/aws/resourcegroupstaggingapi.go`.

## Maintenance

If this hasn't been run a while, you will probably want to bump the dependency on ONV via

```bash
go get -u github.com/openshift/osd-network-verifier
```