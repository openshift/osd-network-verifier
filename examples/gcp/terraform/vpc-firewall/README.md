# GCP VPC with firewall by Terraform for OSD Network Verifier Test
This repository contains Terrafom scripts to set up a VPC in GCP with a public and private subnet. It also creates a cloud NAT for the private subnet and a firewall policy to block egress from certain domains.

Use `dest_fqdns` to add the domains you want blocked to test with the osd-network-verifier.

## Prerequisites
- Terraform 
- GCP account

## Usage
### GCP Credentials
Generate a GCP credentials file by running:
```
gcloud auth application-default login
```
Note the location of the credentials file. The default location and file is `$HOME/.config/gcloud/application_default_credentials.json`.

### Configuration
Before running the script, you need to configure script variables. A `terraform.tfvars.example` file is provided as a template. Here are the steps to configure the variables:
1. Copy the example file:
```
cp terraform.tfvars.example terraform.tfvars
```
2. Use a text editor to set `project` and `credentials_file` in `terraform.tfvars`
- `project`: name of your GCP project
- `credentials_file`: path to your GCP credentials file you generated
3. Set and uncomment any other variables you wish to configure
- `region`: GCP region where resources will be created
- `zone`: GCP zone where resources will be created
- `public_ip_cidr_range`: CIDR block for public subnet within VPC
- `private_ip_cidr_range`: CIDR block for private subnet within VPC
- `dest_fqdns`: list of domains you wish to block

Note: The default value for these variables are defined in `variables.tf`

### Running the scripts
1. Initialize Terraform
```
terraform init
```
2. Check execution plan
```
terraform plan
```
3. Apply changes
```
terraform apply
```
4. Destroy Terraform resources after running verifier
```
terraform destroy
```
## Outputs
The script outputs the IDs of the created VPC and subnets.
```
private_subnet_id = "my-private-subnet"
public_subnet_id = "my-public-subnet"
vpc_name  = "my-vpc"
```

## Test OSD Network Verifier
```
osd-network-verifier/osd-network-verifier egress --platform gcp-classic --subnet-id $subnet_id --vpc-name $vpc_name
```
Replace `$subnet_id` with `private_subnet_id` or `public_subnet_id` and `$vpc_name` with `vpc_name` from the terraform output value.

Example:
```
./osd-network-verifier/osd-network-verifier egress --platform gcp-classic --subnet-id my-private-subnet --vpc-name my-vpc
Using Project ID emhammon-test
Created instance with ID: verifier-4624
Applying labels
Successfully applied labels 
ComputeService Instance: verifier-4624 RUNNING
Gathering and parsing console log output...
Summary:
printing out failures:
 - egressURL error: https://cdn01.quay.io:443 (Failed to connect to cdn01.quay.io port 443: Connection timed out)
 - egressURL error: https://quay.io:443 (Failed to connect to quay.io port 443: Connection timed out)

printing out exceptions preventing the verifier from running the specific test:
printing out errors faced during the execution:
Failure!
```
Because we have configured the firewall rules to block `cdn01.quay.io` and `quay.io`, we can determine the network verifier is working correctly by identifying these domains have been blocked.




