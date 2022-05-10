# osd-network-verifier

A cli and set of libraries that validates the pre-configured networking components for some osd options.

## Overview

osd-network-verifier can be used prior to the installation of osd/rosa clusters to ensure the pre-requirements are valid for various network options.

## Required Permissions

### AWS
- ec2:RunInstances
- ec2:DescribeInstanceStatus
- ec2:DescribeInstanceTypes
- ec2:GetConsoleOutput
- ec2:TerminateInstances
- ec2:DescribeVpcAttribute

## Egress validation

### Workflow for egress

* it creates an instance in the target vpc/subnet and wait till the instance gets ready
* when the instance is ready, an `userdata` script is run in the instance. The `userdata` mainly performs 2 steps, it
    * installs appropriate packages, primarily docker
    * runs the validation image against the vpc/subnet as containerized form of <https://github.com/openshift/osd-network-verifier/tree/main/build>
        * the image is available at: https://quay.io/repository/app-sre/osd-network-verifier?tag=latest&tab=tags
        * it can be run locally or on an instance on the target vpc via:
        ```shell
        docker run --env "AWS_REGION=us-east-1" quay.io/app-sre/osd-network-verifier:latest --timeout=2s
        ```
* the output is collected via the SDK from the EC2 console output, which only includes the userdata script output because of a special line we added to the userdata to redirect the output.

### Validate egress using go library

#### using aws-sdk-go-v2

```go
// validate aws VPC egress access
import (
    "github.com/aws/aws-sdk-go-v2/credentials"
    "github.com/openshift/osd-network-verifier/pkg/cloudclient"
)
// build the credentials provider
creds := credentials.NewStaticCredentialsProvider("AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN")
region := "us-east-1"

// init a cloudclient
cli, err := cloudclient.NewClient(creds, region)
// ... error checking

// call the validation function and check if it was successful
out := cli.ValidateEgress(context.TODO(), "vpcSubnetID", "cloudImageID", "kmsKeyID", 600)
if !out.IsSuccessful() {
    // Failure
    failures, exceptions, errors := out.Parse()
}
```

#### using aws-sdk-go-v1

```go
import (
    "github.com/aws/aws-sdk-go/aws/credentials"
    "github.com/openshift/osd-network-verifier/pkg/cloudclient"
)
// build the credentials provider
creds := credentials.NewStaticCredentials("AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN")
region := "us-east-1"

// init a cloudclient
cli, err := cloudclient.NewClient(*creds, region)
// ... error checking

out := cli.ValidateEgress(context.TODO(), "vpcSubnetID", "cloudImageID", "kmsKeyID", 600)
if !out.IsSuccessful() {
    // Failure
    failures, exceptions, errors := out.Parse()
}
```

### Validate egress using command line

Build the cli executable 
```shell
make build
```
Execute 
```shell
AWS_ACCESS_KEY_ID=$(YOUR_AWS_ACCESS_KEY_ID) AWS_SECRET_ACCESS_KEY=$(YOUR_AWS_SECRET_ACCESS_KEY) ./osd-network-verifier egress --subnet-id <subnet-id> --image-id=<image-id>
```
* For `<image-id>`, use either:
    - the following public image-id: `resolve:ssm:/aws/service/ami-amazon-linux-latest/amzn2-ami-hvm-x86_64-gp2 `
    - or select one from this list, for the region where your subnet is: [AWS account olm-artifacts-template.yaml](https://github.com/openshift/aws-account-operator/blob/17be7a41036e252d59ab19cc2ad1dcaf265758a2/hack/olm-registry/olm-artifacts-template.yaml#L75) 


Optionally provide a list of tags to use outside of the default:

```shell
AWS_ACCESS_KEY_ID=$(YOUR_AWS_ACCESS_KEY_ID) AWS_SECRET_ACCESS_KEY=$(YOUR_AWS_SECRET_ACCESS_KEY) ./osd-network-verifier egress --subnet-id subnet-0ccetestsubnet1864 --image-id=ami-0df9a9ade3c65a1c7 --cloud-tags key=value,osd-network-verifier=owned
```

## VPC DNS verification
Verifying that a given VPC's DNS configuration is correct is fairly straightforward: we
just need to ensure that the VPC attributes `enableDnsHostnames` and `enableDnsSupport`
are both set to `true`.

### Verify DNS using our go API
See the egress documentation above, and replace the line starting with `out := cli.ValidateEgress(...` with:
```go
out := cli.VerifyDns(context.TODO(), "vpcID")
```

### Verify DNS using command line
Build the `osd-network-verifier` executable as shown the egress documentation above.
Then run:
```shell
AWS_ACCESS_KEY_ID=$(YOUR_AWS_ACCESS_KEY_ID) AWS_SECRET_ACCESS_KEY=$(YOUR_AWS_SECRET_ACCESS_KEY) ./osd-network-verifier dns --vpc-id=vpc-0123456789deadbeef
```


## Other Subcommands

Take a look at <https://github.com/openshift/osd-network-verifier/tree/main/cmd>
