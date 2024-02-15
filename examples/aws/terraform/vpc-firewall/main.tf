# AWS provider configuration
provider "aws" {
  profile = var.profile # AWS profile
  region  = var.region  # AWS region
}

# Create a VPC
resource "aws_vpc" "main" {
  cidr_block           = var.vpc_cidr_block
  enable_dns_hostnames = true
}

# Create an Internet Gateway and attach it to the VPC
resource "aws_internet_gateway" "igw" {
  vpc_id = aws_vpc.main.id # ID of the VPC
}

# Create a public subnet within the VPC
resource "aws_subnet" "public" {
  vpc_id            = aws_vpc.main.id              # ID of the VPC
  cidr_block        = var.public_subnet_cidr_block # CIDR block for public subnet
  availability_zone = var.availability_zone        # Availability Zone
}

# Create a private subnet within the VPC
resource "aws_subnet" "private" {
  vpc_id            = aws_vpc.main.id               # ID of the VPC
  cidr_block        = var.private_subnet_cidr_block # CIDR block for private subnet
  availability_zone = var.availability_zone         # Availability Zone
}

# Create a firewall subnet within the VPC
resource "aws_subnet" "firewall" {
  vpc_id            = aws_vpc.main.id                # ID of the VPC
  cidr_block        = var.firewall_subnet_cidr_block # CIDR block for firewall subnet
  availability_zone = var.availability_zone          # Availability Zone
}

# Create a route table for the public subnet
resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id # ID of the VPC
}

# Create a route table for the private subnet
resource "aws_route_table" "private" {
  vpc_id = aws_vpc.main.id # ID of the VPC
}

# Create a route table for the firewall subnet
resource "aws_route_table" "firewall" {
  vpc_id = aws_vpc.main.id # ID of the VPC
}

# Allocate an Elastic IP address for the NAT Gateway
resource "aws_eip" "nat" {
  domain = "vpc"
}

# Create a NAT Gateway and assign it the allocated Elastic IP address
resource "aws_nat_gateway" "nat" {
  subnet_id     = aws_subnet.public.id      # ID of pubic subnet
  allocation_id = aws_eip.nat.allocation_id # ID of the EIP address
}

# Associate the public subnet with its route table
resource "aws_route_table_association" "public" {
  subnet_id      = aws_subnet.public.id      # ID of public subnet
  route_table_id = aws_route_table.public.id # ID of public route table
}

# Associate the private subnet with its route table
resource "aws_route_table_association" "private" {
  subnet_id      = aws_subnet.private.id      # ID of private subnet
  route_table_id = aws_route_table.private.id # ID of private route table
}

# Associate the firewall subnet with its route table
resource "aws_route_table_association" "firewall" {
  subnet_id      = aws_subnet.firewall.id      # ID of firewall subnet
  route_table_id = aws_route_table.firewall.id # ID of firewall route table
}

# Create a network firewall
resource "aws_networkfirewall_firewall" "firewall" {
  name                = var.firewall_name                                       # Name of the firewall
  firewall_policy_arn = aws_networkfirewall_firewall_policy.firewall_policy.arn # ARN of the firewall policy
  vpc_id              = aws_vpc.main.id                                         # ID of the VPC

  # Map the firewall to the firewall subnet
  subnet_mapping {
    subnet_id = aws_subnet.firewall.id # ID of firewall subnet
  }
}

# Create a firewall policy
resource "aws_networkfirewall_firewall_policy" "firewall_policy" {
  name = var.firewall_policy_name # Name of the firewall policy

  # Define the firewall policy
  firewall_policy {
    stateless_default_actions          = ["aws:forward_to_sfe"]
    stateless_fragment_default_actions = ["aws:forward_to_sfe"]
    stateful_rule_group_reference {
      resource_arn = aws_networkfirewall_rule_group.stateful_rule_group.arn # ARN of the rule group
    }
  }
}

# Create a stateful rule group for the firewall
resource "aws_networkfirewall_rule_group" "stateful_rule_group" {
  capacity = 100                 # Capacity of the rule group
  name     = var.rule_group_name # Name of the rule group

  # Define the rule group
  rule_group {
    rules_source {
      rules_source_list {
        generated_rules_type = "DENYLIST"
        target_types         = ["TLS_SNI"]
        targets              = var.block_domains # List of domains to block
      }
    }
  }

  type = "STATEFUL" # Type of the rule group
}

# Define a route for the public subnet
resource "aws_route" "route_public" {
  route_table_id         = aws_route_table.public.id # ID of public route table
  destination_cidr_block = "0.0.0.0/0"               # CIDR block for the destination
  # ID of firewall VPC endpoint 
  # aws_networkfirewall_firewall.firewall.firewall_status[0].sync_states[*].attachment[0].endpoint_id is a set of object                                                                                            
  vpc_endpoint_id = (aws_networkfirewall_firewall.firewall.firewall_status[0].sync_states[*].attachment[0].endpoint_id)[0]
}

# Define a route for the private subnet
resource "aws_route" "route_private" {
  route_table_id         = aws_route_table.private.id # ID of private route table
  destination_cidr_block = "0.0.0.0/0"                # CIDR block for the destination
  nat_gateway_id         = aws_nat_gateway.nat.id     # ID of the NAT Gateway
}

# Define a route for the firewall subnet
resource "aws_route" "route_firewall" {
  route_table_id         = aws_route_table.firewall.id # ID of firewall route table
  destination_cidr_block = "0.0.0.0/0"                 # CIDR block for the destination
  gateway_id             = aws_internet_gateway.igw.id # ID of the Internet Gateway
}

# Create a route table for the Internet Gateway
resource "aws_route_table" "igw" {
  vpc_id = aws_vpc.main.id # ID of the VPC
}

# Define a route for the Internet Gateway
resource "aws_route" "route_igw" {
  route_table_id         = aws_route_table.igw.id       # ID of Internet Gateway route table
  destination_cidr_block = var.public_subnet_cidr_block # CIDR block for public subnet
  # ID of firewall VPC endpoint 
  # aws_networkfirewall_firewall.firewall.firewall_status[0].sync_states[*].attachment[0].endpoint_id is a set of object 
  vpc_endpoint_id = (aws_networkfirewall_firewall.firewall.firewall_status[0].sync_states[*].attachment[0].endpoint_id)[0]
}
# Edge Associate the Internet Gateway with route table 
resource "aws_route_table_association" "igw_association" {
  gateway_id     = aws_internet_gateway.igw.id # ID of the Internet Gateway
  route_table_id = aws_route_table.igw.id      # ID of Internet Gateway route table
}

# Output the ID of the VPC
output "vpc_id" {
  description = "The ID of the VPC"
  value       = aws_vpc.main.id
}

# Output the region of the VPC
data "aws_region" "current" {}

output "region" {
  description = "The region of the VPC"
  value       = data.aws_region.current.name
}

# Output the ID of the public subnet
output "public_subnet_id" {
  description = "The ID of the Public Subnet"
  value       = aws_subnet.public.id
}

# Output the ID of the private subnet
output "private_subnet_id" {
  description = "The ID of the Private Subnet"
  value       = aws_subnet.private.id
}

# Output the ID of the firewall subnet
output "firewall_subnet_id" {
  description = "The ID of the Private Subnet"
  value       = aws_subnet.firewall.id
}
