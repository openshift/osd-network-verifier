# AWS Profile to use
variable "profile" {
  description = "AWS Profile to use"
  type        = string
  default     = "default"
}

# The AWS Region where resources will be created
variable "region" {
  description = "AWS Region"
  type        = string      # Expect a string input
  default     = "us-east-1" # Default to US East (N. Virginia)
}

# The availability zone where the subnets will be created
variable "availability_zone" {
  description = "The availability zone where the subnets will be created"
  type        = string       # Expect a string input
  default     = "us-east-1a" # Default to US East (N. Virginia) AZ a
}

# CIDR block for the VPC
variable "vpc_cidr_block" {
  description = "CIDR block for the VPC"
  type        = string        # Expect a string input
  default     = "10.0.0.0/16" # Default to a /16 block within the 10.0.0.0 private network
}

# CIDR block for the public subnet
variable "public_subnet_cidr_block" {
  description = "CIDR block for the public subnet"
  type        = string        # Expect a string input
  default     = "10.0.0.0/24" # Default to a /24 block within the 10.0.0.0 private network
}

# CIDR block for the private subnet
variable "private_subnet_cidr_block" {
  description = "CIDR block for the private subnet"
  type        = string        # Expect a string input
  default     = "10.0.1.0/24" # Default to a /24 block within the 10.0.0.0 private network
}