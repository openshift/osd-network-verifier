# AWS Profile to use
variable "profile" {
  description = "AWS Profile to use"
  type        = string
  default     = "default"
}

# The AWS Region where resources will be created
variable "region" {
  description = "AWS Region"
  type        = string
  default     = "us-east-1" # Default to US East (N. Virginia)
}

# The availability zone where the subnets will be created
variable "availability_zone" {
  description = "The availability zone where the subnets will be created"
  type        = string
  default     = "us-east-1a" # Default to US East (N. Virginia) AZ a
}

# CIDR block for the VPC
variable "vpc_cidr_block" {
  description = "CIDR block for the VPC"
  type        = string
  default     = "10.0.0.0/16" # Default to a /16 block within the 10.0.0.0 private network
}

# CIDR block for the public subnet
variable "public_subnet_cidr_block" {
  description = "CIDR block for the public subnet"
  type        = string
  default     = "10.0.0.0/24" # Default to a /24 block within the 10.0.0.0 private network
}

# CIDR block for the private subnet
variable "proxied_subnet_cidr_block" {
  description = "CIDR block for the private subnet"
  type        = string
  default     = "10.0.1.0/24" # Default to a /24 block within the 10.0.0.0 private network
}

# User/developer's CIDR block for SSH/webUI access to the proxy machine for debugging
variable "developer_cidr_block" {
  description = "A CIDR block containing your workstation's IP, for SSH/webUI access to the proxy machine for debugging. The output of 'echo $(curl -s ipv4.icanhazip.com)/32' is a sane default"
  type        = string
}

# Username to use for the proxy machine web UI
variable "proxy_webui_username" {
  description = "Username to use for the proxy machine web UI"
  type        = string
  default     = "developer"
}

# Prefix to add to name tags
variable "name_prefix" {
  description = "prefix to add to the Name tag associated with most of the resources created by these scripts"
  type        = string
  default     = "transparent-proxy-"
}

# SSH public key to use for ec2-user@proxy-machine
variable "proxy_machine_ssh_pubkey" {
  description = "SSH public key to use for ec2-user@proxy-machine"
  type        = string
}

