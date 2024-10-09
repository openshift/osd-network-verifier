### Table of Contents ###

<!-- TOC -->
    * [Table of Contents](#table-of-contents-)
  * [Setup](#setup-)
    * [AWS Environment](#aws-environment-)
    * [VPC](#vpc-)
    * [IAM permissions](#iam-permissions-)
  * [Available Tools](#available-tools-)
    * [1. Egress Verification](#1-egress-verification-)
      * [1.1 Usage](#11-usage-)
        * [1.1.1 CLI Executable](#111-cli-executable-)
        * [Egress Validations Under Proxy](#egress-validations-under-proxy-)
        * [Force Temporary Security Group Creation](#force-temporary-security-group-creation-)
        * [1.1.2 Go implementation Examples](#112-go-implementation-examples-)
      * [1.2 Interpreting Output](#12-interpreting-output-)
      * [1.3 Workflow](#13-workflow-)
    * [2. VPC DNS Verification](#2-vpc-dns-verification-)
      * [2.1 Usage](#21-usage-)
        * [2.1.1 CLI Executable](#211-cli-executable-)
        * [2.1.2 Golang API](#212-golang-api-)
<!-- TOC -->

## Setup ##
### AWS Environment ###
Set up your environment to use the correct credentials for the AWS account for the target cluster. 
- Obtain a valid set of AWS secret and key for the target account and use them in one of the following ways:
  - Set them as an AWS profile in you ~/.aws/credentials file as prescribed in [this AWS doc.](https://docs.aws.amazon.com/sdk-for-php/v3/developer-guide/guide_credentials_profiles.html)
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
Ensure that the AWS credentials being used have the following permissions. (This list is a subset of permissions documented in the Support role and Support policy sections [in this doc.](https://docs.openshift.com/rosa/rosa_architecture/rosa-sts-about-iam-resources.html#rosa-sts-account-wide-roles-and-policies_rosa-sts-about-iam-resources))
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:CreateTags",
        "ec2:RunInstances",
        "ec2:DescribeInstances",
        "ec2:DescribeInstanceTypes",
        "ec2:GetConsoleOutput",
        "ec2:TerminateInstances",
        "ec2:DescribeVpcAttribute",
        "ec2:CreateSecurityGroup",
        "ec2:DeleteSecurityGroup",
        "ec2:DescribeSecurityGroup",
        "ec2:AuthorizeSecurityGroupEgress",
        "ec2:RevokeSecurityGroupEgress",
        "ec2:DescribeSubnets"
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
To verify the entire VPC, repeat the verification process for each subnet ID.

##### 1.1.1 CLI Executable #####
   1. Ensure correct [environment setup](#setup-).

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
      1. subnet-id: the subnet id to be verified.
      2. platform: This parameter dictates for which set of endpoints the verifier should test. If testing a subnet that hosts (or will host) a traditional OSD/ROSA cluster, set this to `aws` (or leave blank). If you're instead testing a subnet hosting a HyperShift Hosted Cluster (*not* a hosted control plane/management cluster) on AWS, set this to `hostedcluster`.

   5. Execute:

       ```shell        
      # using AWS profile on an OSD/ROSA cluster
      ./osd-network-verifier egress --platform aws-classic --subnet-id $SUBNET_ID --profile $AWS_PROFILE
      
      # using AWS secret on a HyperShift hosted cluster
        AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY  \
      ./osd-network-verifier egress --platform aws-hcp --subnet-id $SUBNET_ID  
        ```
   
        Additional optional flags for overriding defaults can be found with:
        ```shell
        ./osd-network-verifier egress --help
        ```

##### Egress Validations Under Proxy #####

* Follow the same flow shown above, until execution
* Pass the proxy config to the egress subcommand

```shell
./osd-network-verifier egress \
    --subnet-id <subnet_id>  \
    --http-proxy http://sre:123@18.18.18.18:8888 \
    --https-proxy https://sre:123@18.18.18.18:8888 \
    --cacert path-to-ca.pem \
    --no-tls # optional, used to bypass ca.pem validation (https)
```

##### Force Temporary Security Group Creation #####

* Follow the similar flow above, till execute
* Use the `--force-temp-security-group` flag

```shell
./osd-network-verifier egress \
    --subnet-id <subnet_id>  \
    --force-temp-security-group \
    --security-group-ids=<securityGroupID-1, ..., securityGroupID-N> # To add extra security Groups in addtion to the temporary one.
```

##### 1.1.2 Go implementation Examples #####
- [Verify Egress Example](../../examples/aws/verify_egress.go)
 
#### 1.2 Interpreting Output ###
(TODO: add errors)

#### 1.3 Workflow ####
Pictorial representation of the egress test tool workflow:

 ![egress](https://user-images.githubusercontent.com/87340776/168323176-af0c8a37-2bdc-4747-82f0-f464970d5373.jpg)


Description:
The AWS client creates a test EC2 instance in the target VPC/subnet and waits until the instance is ready.
The actual network verification is automated
by using the `USERDATA` param [available for ec2 instances](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/user-data.html)
which is run by ec2 on the instance on creation.

The instance's console output is redirected to the AWS cloud client SDK.
The active probe then parses this output before the verifier prints it to the user's terminal.
If debug logging is enabled, the verifier prints this output is printed in full; otherwise it only prints errors.

---
**NOTE**
For more information on probes, see [the README](../../README.md#probes).

---

### 2. VPC DNS Verification ###
#### 2.1 Usage ####
Verifying that a given VPC's DNS configuration is correct is fairly straightforward: we
just need to ensure that the VPC attributes `enableDnsHostnames` and `enableDnsSupport`
are both set to `true`. This tool automates that process

##### 2.1.1 CLI Executable #####
Build the `osd-network-verifier` executable as shown the egress documentation above.
Then run:

```shell
 # using AWS profile    
  ./osd-network-verifier dns --vpc-id=$VPC_ID --profile $AWS_PROFILE
  
 # using AWS secret
  AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY  \
  ./osd-network-verifier dns --vpc-id=$VPC_ID 
```

##### 2.1.2 Golang API #####
See the egress golang examples above, and replace the line starting with `out := cli.ValidateEgress(...` with:
```go
out := cli.VerifyDns(context.TODO(), "vpcID")
```