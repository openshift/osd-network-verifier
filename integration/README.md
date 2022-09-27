# osd-network-verifier integration test

This houses an integration test that will:

1. Create a BYOVPC with an AWS Network Firewall blocking quay.io following this architecture: 
https://docs.aws.amazon.com/network-firewall/latest/developerguide/arch-igw-ngw.html.
2. Run osd-network-verifier's egress check with the following expected output:

    In essence, it should run using the pre-pulled AMI and be able to validate that quay.io
    is blocked even though it cannot pull the desired container image hosted on quay.io.

    ```
   EC2 Instance: i-0e6ed3c2e89834405 Running
   Terminating ec2 instance with id i-0e6ed3c2e89834405
   Summary:
   printing out failures:
   egressURL error: Unable to reach quay.io:443
   ```
   
3. Delete the BYOVPC and AWS Network Firewall created in step 1

## Usage

It can be run with AWS credentials setup as environment variables beforehand, e.g.

```bash
DEV_ACCOUNT_ID=""
REGION=""
export $(osdctl account cli -i ${DEV_ACCOUNT_ID}-p osd-staging-2 -r ${REGION} -oenv | xargs)
go build .
./integration --region "${REGION}"
```

Or with a profile:

```bash
go build .
./integration --region "${REGION}" --profile "${PROFILE}"
```

TODO: Create a minimal IAM policy. The required permissions are laid out in:
`pkg/aws/ec2.go`, `pkg/aws/networkfirewall.go`, and `pkg/aws/resourcegroupstaggingapi.go`.

## Maintenance

If this hasn't been run a while, you will probably want to bump the dependency on ONV via

```bash
go get -u github.com/openshift/osd-network-verifier
```