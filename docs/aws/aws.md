### Table of Contents ###

- [Setup](#setup)
  - [AWS Environment](#aws-environment)
  - [VPC](#vpc)
  - [IAM Permissions](#iam-permissions)
- [Available tools](#available-tools)
  - [1. Egress Verification](#1-egress-verification)
    - [1.1 Usage](#11-usage)
      - [1.1.1 CLI Executable](#111-cli-executable)
      - [1.1.2 Go Implementation Examples](#112-go-implementation-examples)
    - [1.2 Interpreting Output](#12-interpreting-output)
    - [1.3 Workflow](#13-workflow)
  - [2. VPC DNS Verification](#2-vpc-dns-verification)
    - [2.1 Usage](#21-usage)
      - [2.1.1 CLI Executable](#211-cli-executable)
      - [2.1.2 Golang API](#212-golang-api)
  - [3. BYOVPC Configurations Verification](#3-byovpc-configurations-verification)

## Setup ##
### AWS Environment ###
Set up your environment to use the correct credentials for the AWS account for the target cluster. 
- Obtain a valid set of AWS secret and key for the target account.
- Export these AWS credentials:
   ```shell
   export AWS_ACCESS_KEY_ID=<YOUR_AWS_ACCESS_KEY_ID)>
   export AWS_SECRET_ACCESS_KEY=<YOUR_AWS_SECRET_ACCESS_KEY>
   ```
  For STS credentials, also:
    ```shell 
    export AWS_SESSION_TOKEN=<YOUR_SESSION_TOKEN_STRING> 
    ```
  Export any other AWS environment vars:
    ```shell
    export AWS_REGION=<VPC_AWS_REGION>
    ````
  
### VPC ###
- Any VPC for a ROSA/OSD CCS cluster can be tested using this tool.
- You should get the VPC set up by the customer.  
- To set up your own VPC and firewall for testing and development, [check out this example](firewall.md).
- Apart from the AWS credentials, you will need to know the following information about the VPC to be verified.
    - Subnet IDs
    - AWS region
    - VPC ID (if verifying DNS)
  
### IAM permissions ###
Ensure that the AWS credentials being used have the following permissions.
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:RunInstances",
        "ec2:DescribeInstanceStatus",
        "ec2:DescribeInstanceTypes",
        "ec2:GetConsoleOutput",
        "ec2:TerminateInstances",
        "ec2:DescribeVpcAttribute" 
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

   2. Clone the source:
      ```shell
      git clone git@github.com:openshift/osd-network-verifier.git
      ``` 
   3. Build the cli:
      ```shell
      make build
      ```
      This generates `osd-network-verifier` executable in project root directory. 

   4. Obtain params:
      1. subnet_id: Obtain the subnet id to be verified. 
      2. image_id: Select an optional image id parameter (ami-xxxxxxxxxxxx) to run on ec2 instance. 
      
         You may use the following public image-id:
         ```bash
          resolve:ssm:/aws/service/ami-amazon-linux-latest/amzn2-ami-hvm-x86_64-gp2`
         ```
          If the image id is not provided, it is defaulted to an image id from [AWS account olm-artifacts-template.yaml](https://github.com/openshift/aws-account-operator/blob/17be7a41036e252d59ab19cc2ad1dcaf265758a2/hack/olm-registry/olm-artifacts-template.yaml#L75),
   for the same region where your subnet is.

   5. Execute:
    
       ```shell        
       ./osd-network-verifier egress \
           --subnet-id <subnet_id> 
        ```
   
        Additional optional flags for overriding defaults:
      ```shell
      --cloud-tags stringToString   (optional) comma-seperated list of tags to assign to cloud resources (default [osd-network-verifier=owned,red-hat-managed=true,Name=osd-network-verifier])
      --debug                       (optional) if true, enable additional debug-level logging
      --image-id string             (optional) specify cloud image for the compute instance
      --instance-type string        compute instance type (default "(optional) t3.micro")
      --kms-key-id string           (optional) ID of KMS key used to encrypt root volumes of compute instances. Defaults to cloud account default key
      --region string               (optional) compute instance region. If absent, environment var AWS_DEFAULT_REGION will be used, if set (default "us-east-1")
      --subnet-id string            source subnet ID
      --timeout duration            (optional) timeout for individual egress verification requests (default 1s)
       ```
       Get cli help:
    
        ```shell
        ./osd-network-verifier egress --help
        ```

##### 1.1.2 Go implementation Examples #####
- [AWS Go SDK v1](../../examples/aws/verify_egressv1.go)  
- [AWS Go SDK v2](../../examples/aws/verify_egressv2.go)
 
#### 1.2 Interpreting Output ###
(TODO: add errors)

#### 1.3 Workflow ####
Pictorial representation of workflow of the egress test tool:

 ![egress](https://user-images.githubusercontent.com/87340776/168323176-af0c8a37-2bdc-4747-82f0-f464970d5373.jpg)


Description:

1. AWS client creates a test ec2 instance in the target vpc/subnet and wait till the instance gets ready
2. The actual network verification is automated by using the `USERDATA` param [available for ec2 instances](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/user-data.html) which is run by ec2 on the instance on creation. 
3. The [`USERDATA`](../../pkg/helpers/config/userdata.yaml) script is in the form of base64-encoded text, and does the following -

   1. installs docker
   2. runs the [osd-network-verifier docker image](https://github.com/openshift/osd-network-verifier/tree/main/build) included with this source.
      (The image is also published at: https://quay.io/repository/app-sre/osd-network-verifier)
   3. The entry point of the osd-network-verifier docker image then executes the main egress verification script
      ```shell
      network-validator --timeout=1s --config=config/config.yaml
       ```
      - **This entrypoint is where the actual egress endpoint verification is performed.** `build/bin/network-validator.go` makes `curl` requests to each other endpoint in the [egress list](../../README.md#egress-list) (i.e. list of all essential domains for OSD clusters).
      - During development, the verifier docker image can be tested locally as:
         ```shell
         docker run --env "AWS_REGION=us-east-1" quay.io/app-sre/osd-network-verifier:latest --timeout=2s
         ```
   
4. `USERDATA` script then redirects the instance's console output to the AWS cloud client SDK. The end of this output message is signified with a special End Verification string.
5. If debug logging is enabled, this output is printed in full, otherwise only errors are printed, if any.

### 2. VPC DNS Verification ###
#### 2.1 Usage ####
Verifying that a given VPC's DNS configuration is correct is fairly straightforward: we
just need to ensure that the VPC attributes `enableDnsHostnames` and `enableDnsSupport`
are both set to `true`. This tool automates that process

##### 2.1.1 CLI Executable #####
Build the `osd-network-verifier` executable as shown the egress documentation above.
Then run:
```shell
AWS_ACCESS_KEY_ID=$(YOUR_AWS_ACCESS_KEY_ID) AWS_SECRET_ACCESS_KEY=$(YOUR_AWS_SECRET_ACCESS_KEY) ./osd-network-verifier dns --vpc-id=vpc-0123456789deadbeef
```

##### 2.1.2 Golang API #####
See the egress golang examples above, and replace the line starting with `out := cli.ValidateEgress(...` with:
```go
out := cli.VerifyDns(context.TODO(), "vpcID")
```

### 3. BYOVPC Configurations Verification ###
(TODO: add doc)
