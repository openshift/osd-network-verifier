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
  vpc_id                  = aws_vpc.main.id              # ID of the VPC
  cidr_block              = var.public_subnet_cidr_block # CIDR block for public subnet
  availability_zone       = var.availability_zone        # Availability Zone
  map_public_ip_on_launch = true
}

# Create a private subnet within the VPC
resource "aws_subnet" "private" {
  vpc_id            = aws_vpc.main.id               # ID of the VPC
  cidr_block        = var.private_subnet_cidr_block # CIDR block for private subnet
  availability_zone = var.availability_zone         # Availability Zone
}

# Create a route table for the public subnet
resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id # ID of the VPC

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.igw.id
  }
}

# Create a route table for the private subnet
resource "aws_route_table" "private" {
  vpc_id = aws_vpc.main.id # ID of the VPC
  route {
    cidr_block     = "0.0.0.0/0"
    nat_gateway_id = aws_nat_gateway.nat.id
  }
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

