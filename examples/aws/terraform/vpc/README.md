# AWS VPC with no firewall by Terraform for OSD Network Verifier Test

This repository contains Terraform scripts to set up a VPC in AWS with public, private subnets, and aslo an Internet Gateway. 

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
$ ./osd-network-verifier egress --platform aws --subnet-id subnet-0654xxxxxxxxfd95b --security-group-id "" --profile default --region us-east-1                                                                                              1 ↵
Using region: us-east-1
Created security group with ID: sg-069exxxxxxxx200ee
Created instance with ID: i-08e1xxxxxxxx768d9
Summary:
All tests passed!
Success
```