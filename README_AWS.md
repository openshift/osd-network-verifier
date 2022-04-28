### Table of Contents ###

- [Setup](#setup)
  - [VPC](#vpc)
  - [AWS Environment](#aws-environment)
  - [IAM Support Role](#iam-support-role)
- [Available tools](#available-tools)
  - [1. Egress Verification](#1-egress-verification)
    - [1.1 Usage](#11-usage)
      - [1.1.1 CLI Executable](#111-cli-executable)
      - [1.1.2 AWS Go SDK Implementaiton](#112-aws-go-sdk-implementation)
    - [1.2 Interpreting Output](#12-interpreting-output)
    - [1.3 Workflow](#13-workflow)
  - [2. BYOVPC Configurations Verification](#2-byovpc-configurations-verification)

## Setup ##
### VPC ###
- Any VPC for a non-STS CCS cluster can be tested using this tool.
- You will need to know the following information about the VPC to be verified.
    - subnet IDs
    - AWS region
  
### AWS Environment ###
Set up your environment to use the correct credentials for the AWS account for the target cluster. 
- If this is an existing cluster, use [this SOP](https://github.com/openshift/ops-sop/blob/master/v4/howto/aws/aws.md#via-ocm-the-quickest-way-1) to get AWS credentials.
- If this cluster is not installed yet:
  - If cluster is STS, customer should provide credentials for support role.
  - If cluster is non-STS, creds are creaed by AWS account operator on hive shard. Obtain them using [this SOP](https://github.com/openshift/ops-sop/blob/master/v4/howto/aws/aws.md#on-hive-shard).
- Export these AWS credentials and any other AWS defaults:
   ```shell
   export AWS_ACCESS_KEY_ID=<YOUR_AWS_ACCESS_KEY_ID)>
   export AWS_SECRET_ACCESS_KEY=<YOUR-AWS_SECRET_ACCESS_KEY>
   ```

    ```shell
    export AWS_DEFAULT_REGION=<VPC_AWS_REGION>
    ````
  
### IAM Support Role ###
Ensure that the IAM support role policy (default: ManagedOpenShift-Support-Role-Policy) includes the following permissions.
```json
{
  "Version": "<version>",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:RunInstances",
        "ec2:DescribeInstanceStatus",
        "ec2:DescribeInstanceTypes",
        "ec2:GetConsoleOutput",
        "ec2:TerminateInstances"
      ],
      "Resource": "*"
    }
  ]
}
```
 
## Available Tools ##

### 1. Egress Verification ###
#### 1.1 Usage ####
The processes below describe different ways of using egress verifier on a single subnet. 
In order to verify entire VPC, 
repeat the verification process for each subnet ID.

##### 1.1.1 CLI Executable #####
   1. Ensure correct [environment setup](#setup).

   2. Clone this [repo.](https://github.com/openshift/osd-network-verifier)
   3. Build the cli.
      ```shell
      make build
      ```
      This generates `osd-network-verifier` executable in project root directory. 
   4. From AWS, obtain the subnet id to be verified and export it.
       ```shell 
       export SUBNET_ID=<subnet_id>
        ```
   5. Set the optional image id parameter (in the form ami-xxxxxxxxxxxx) to run on ec2 instance. You may use the following public image-id
       ```shell 
       export IMAGE_ID=resolve:ssm:/aws/service/ami-amazon-linux-latest/amzn2-ami-hvm-x86_64-gp2 
       ```
        If the image id is not passed, it is defaulted to an image id from [AWS account olm-artifacts-template.yaml](https://github.com/openshift/aws-account-operator/blob/17be7a41036e252d59ab19cc2ad1dcaf265758a2/hack/olm-registry/olm-artifacts-template.yaml#L75),
   for the same region where your subnet is.

   6. Execute:
    
        ```shell        
        ./osd-network-verifier egress --subnet-id $(SUBNET_ID) --image-id=$(IMAGE_ID)
        ```

        Optionally, provide a list of tags to apply to the test instance:
       
         ```shell
        ./osd-network-verifier egress --subnet-id=$(SUBNET_ID) \
         --image-id=$(IMAGE_ID) \
         --cloud-tags osd-network-verifier=owned,key1=value1,key2=value2
        ```
       Get more help:
    
        ```shell
        ./osd-network-verifier egress --help
        ```

##### 1.1.2 AWS Go SDK implementation #####
##### aws-sdk-go-v2 #####
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

##### aws-sdk-go-v1 #####
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
#### 1.2 Interpreting Output ###
(TODO: add errors)

#### 1.3 Workflow ####
1. AWS client creates a test ec2 instance in the target vpc/subnet and wait till the instance gets ready
2. The actual network verification is automated by using the `USERDATA` param [available for ec2 instances](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/user-data.html) which is run by ec2 on the instance on creation. 
The [`USERDATA`](pkg/helpers/config/userdata.yaml) script is in the form of base64-encoded text, and does the following -
   1. passes default cloud configurations
   2. installs dependencies
   3. runs the [ONV docker image](https://github.com/openshift/osd-network-verifier/tree/main/build) included with this source.
      (The image is also published at: https://quay.io/repository/app-sre/osd-network-verifier)
3. The entry point of the ONV docker image then executes the main egress verification script
   ```shell
   network-validator --timeout=1s --config=config/config.yaml
    ```
   - **This entrypoint is where the actual egress endpoint verification is performed.** `build/bin/network-validator.go` makes `curl` requests to each other endpoint in the [egress list](README.md#egress-list) (i.e. list of all essential domains for OSD clusters).
   - During development, the verifier docker image can be tested locally as:
      ```shell
      docker run --env "AWS_REGION=us-east-1" quay.io/app-sre/osd-network-verifier:latest --timeout=2s
      ```
   
4. `USERDATA` script then redirects the instance's console output to the AWS cloud client SDK. The end of this output message is signified with a special End Verification string.
5. If debug logging is enabled, this output is printed in full, otherwise only errors are printed, if any.

### 2. BYOVPC Configurations Verification ###
(TODO: add doc)