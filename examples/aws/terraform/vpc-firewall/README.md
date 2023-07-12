# AWS VPC with firewall by Terraform for OSD Network Verifier Test

This repository contains Terraform scripts to set up a VPC in AWS with public, private, and firewall subnets. An Internet Gateway is also set up for the VPC. A network firewall with a firewall policy and a rule group is also set up for the VPC.

Use `block_domains` to add the domain you want to blocked and then test with osd-network-verifier.

## Prerequisites

- Terraform 0.12.x or later
- AWS Account

## Usage

### AWS Credentials

This script uses AWS profiles for authentication. You should configure your AWS credentials in your AWS credentials file. The default location is `~/.aws/credentials` on Unix systems and `C:\Users\USERNAME\.aws\credentials` on Windows. You can specify the profile to use in the `terraform.tfvars` file.

### Configuration

Before running the scripts, you need to configure the variables used by the scripts. A `terraform.tfvars.example` file is provided as a template. Here are the steps to configure the variables:

1. Copy the example file:

    ```bash
    cp terraform.tfvars.example terraform.tfvars
    ```

2. Open the terraform.tfvars file in a text editor.

3. Replace the Variable values with your actual values. Here is an explanation of each variable:

- `profile`: The AWS profile to use. This profile should be configured in your AWS credentials file.
- `region`: The AWS region where resources will be created.
- `availability_zone`: The availability zone within the region where subnets will be created.
- `vpc_cidr_block`: The CIDR block for the VPC.
- `public_subnet_cidr_block`: The CIDR block for the public subnet within the VPC.
- `private_subnet_cidr_block`: The CIDR block for the private subnet within the VPC.
- `firewall_subnet_cidr_block`: The CIDR block for the firewall subnet within the VPC.
- `block_domains`: A list of domains that you want to block.
- `firewall_name`: The name of the network firewall.
- `firewall_policy_name`: The name of the firewall policy.
- `rule_group_name`: The name of the stateful rule group for the firewall.

### Running the scripts

1. Initialize Terraform:

```bash
terraform init
```


2. Check the execution plan:

```bash
terraform plan
```


3. Apply the changes:

```bash
terraform apply
```


4. To destroy the resources:

```bash
terraform destroy
```


## Outputs

The scripts output the IDs of the created VPC and subnets.

- `vpc_id`: The ID of the VPC.
- `region`: The region of the VPC.
- `public_subnet_id`: The ID of the public subnet.
- `private_subnet_id`: The ID of the private subnet.
- `firewall_subnet_id`: The ID of the firewall subnet.

## Test OSD Network Verifier

Use the following command example tp verfifer the block domain failed the verifier.

```bash
osd-network-verifier egress \
    --platform aws \
    --subnet-id $private_subnet_id \
    --security-group-id "" \
    --profile $aws_profile \
    --region $region
```
Replace `$private_subnet_id`, `$aws_profile` and `$region` with the terraform output value.

Example:

```bash
$ ./osd-network-verifier egress --platform aws --subnet-id subnet-080exxxxxxxx6aef1 --security-group-id "" --profile default --region us-east-1                                                                                                                         1 ↵
Using region: us-east-1
Created security group with ID: sg-04f4xxxxxxxx53f29
Created instance with ID: i-0197xxxxxxxx8c4ce
Summary:
printing out failures:
 - egressURL error: api.openshift.com:443
 - egressURL error: quay.io:443
 - egressURL error: registry.redhat.io:443

printing out exceptions preventing the verifier from running the specific test:
printing out errors faced during the execution:
Failure!
```