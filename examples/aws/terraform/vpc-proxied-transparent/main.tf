# AWS provider configuration
provider "aws" {
  profile = var.profile # AWS profile
  region  = var.region  # AWS region
}

## RESOURCES
# Create a VPC
resource "aws_vpc" "main" {
  cidr_block           = var.vpc_cidr_block
  enable_dns_support   = true
  enable_dns_hostnames = true
  tags                 = { Name = "${var.name_prefix}-vpc" }
}

# Create an Internet Gateway and attach it to the VPC
resource "aws_internet_gateway" "igw" {
  vpc_id = aws_vpc.main.id
  tags   = { Name = "${var.name_prefix}-igw" }
}

# Create a public subnet within the VPC (where the proxy machine will live)
resource "aws_subnet" "public" {
  vpc_id                  = aws_vpc.main.id
  cidr_block              = var.public_subnet_cidr_block # CIDR block for public subnet
  availability_zone       = var.availability_zone
  map_public_ip_on_launch = true
  tags                    = { Name = "${var.name_prefix}-public" }
}

# Create a route table for the public subnet
resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id
  tags   = { Name = "${var.name_prefix}-public-rtb" }

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.igw.id
  }
}

# Associate the public subnet with its route table (skipping this because default RTB is fine for public sub)
resource "aws_route_table_association" "public" {
  subnet_id      = aws_subnet.public.id
  route_table_id = aws_route_table.public.id
}

# Create a security group for the proxy machine (rules below)
resource "aws_security_group" "proxy_machine_sg" {
  name_prefix = var.name_prefix
  description = "Allow all outbound traffic and inbound traffic from proxied subnet or developer SSH client"
  vpc_id      = aws_vpc.main.id
}
# proxy_machine_sg: Allow all ingress traffic from the proxied subnet
resource "aws_vpc_security_group_ingress_rule" "allow_proxied_traffic" {
  security_group_id = aws_security_group.proxy_machine_sg.id
  cidr_ipv4         = aws_subnet.proxied.cidr_block
  ip_protocol       = "-1" # semantically equivalent to all ports
}
# proxy_machine_sg: Allow SSH traffic from the developer's IP
resource "aws_vpc_security_group_ingress_rule" "allow_developer_ssh" {
  security_group_id = aws_security_group.proxy_machine_sg.id
  cidr_ipv4         = var.developer_cidr_block
  from_port         = 22
  to_port           = 22
  ip_protocol       = "tcp"
}
# proxy_machine_sg: Allow webUI traffic from the developer's IP
resource "aws_vpc_security_group_ingress_rule" "allow_developer_webui" {
  security_group_id = aws_security_group.proxy_machine_sg.id
  cidr_ipv4         = var.developer_cidr_block
  from_port         = 8443
  to_port           = 8443
  ip_protocol       = "tcp"
}
# proxy_machine_sg: Allow all IPv4 egress traffic from the proxy
resource "aws_vpc_security_group_egress_rule" "allow_all_traffic_ipv4" {
  security_group_id = aws_security_group.proxy_machine_sg.id
  cidr_ipv4         = "0.0.0.0/0"
  ip_protocol       = "-1" # semantically equivalent to all ports
}
# proxy_machine_sg: Allow all IPv6 egress traffic from the proxy
resource "aws_vpc_security_group_egress_rule" "allow_all_traffic_ipv6" {
  security_group_id = aws_security_group.proxy_machine_sg.id
  cidr_ipv6         = "::/0"
  ip_protocol       = "-1" # semantically equivalent to all ports
}
# End proxy_machine_sg rules

# Create an SSH keypair that the user can use for debugging the proxy_machine
resource "aws_key_pair" "proxy_machine_key" {
  key_name_prefix = var.name_prefix
  public_key      = var.proxy_machine_ssh_pubkey
}

# Generate a random password for the proxy web UI
resource "random_password" "proxy_webui_password" {
  length  = 16
  special = false
}

# Create the proxy EC2 instance inside the public subnet
resource "aws_instance" "proxy_machine" {
  ami               = data.aws_ami.rhel9.id
  instance_type     = "t3.micro"
  key_name          = aws_key_pair.proxy_machine_key.key_name # SSH key for debugging
  availability_zone = var.availability_zone
  tags              = { Name = "${var.name_prefix}-proxy-machine" }

  user_data = templatefile(
    "assets/userdata.yaml.tpl",
    {
      mitmproxy_sysctl_b64  = filebase64("assets/mitmproxy-sysctl.conf")
      mitmproxy_service_b64 = filebase64("assets/mitmproxy.service")
      caddyfile_b64 = base64encode(templatefile(
        "assets/Caddyfile.tpl",
        {
          proxy_webui_password_hash = random_password.proxy_webui_password.bcrypt_hash
          proxy_webui_username      = var.proxy_webui_username
        }
      ))
    }
  )
  user_data_replace_on_change = true # Destroy and re-create this instance if user-data.yaml changes

  subnet_id                   = aws_subnet.public.id
  vpc_security_group_ids      = [aws_security_group.proxy_machine_sg.id]
  associate_public_ip_address = true  # Necessary b/c we're not using a NAT gateway
  source_dest_check           = false # Critical for correct routing
}

# Create a proxied subnet (where the test/"captive" machines will live)
resource "aws_subnet" "proxied" {
  vpc_id            = aws_vpc.main.id
  availability_zone = var.availability_zone
  cidr_block        = var.proxied_subnet_cidr_block
  tags              = { Name = "${var.name_prefix}-proxied" }
}

# Create a route table for the proxied subnet that routes all traffic into the proxy_machine
resource "aws_route_table" "proxied" {
  vpc_id = aws_vpc.main.id
  tags   = { Name = "${var.name_prefix}-proxied-rtb" }

  route {
    cidr_block           = "0.0.0.0/0"
    network_interface_id = aws_instance.proxy_machine.primary_network_interface_id
  }
}

# Associate the proxied subnet with its route table
resource "aws_route_table_association" "proxied" {
  subnet_id      = aws_subnet.proxied.id      # ID of proxied subnet
  route_table_id = aws_route_table.proxied.id # ID of proxied route table
}

## OUTPUTS
output "region" {
  description = "VPC region"
  value       = data.aws_region.current.name
}
output "proxy_machine_instance_id" {
  description = "Proxy machine instance ID"
  value       = aws_instance.proxy_machine.id
}
output "proxy_machine_ssh_url" {
  description = "SSH URL for logging into the proxy machine"
  value       = "ssh://ec2-user@${aws_instance.proxy_machine.public_ip}"
}
output "proxy_webui_url" {
  description = "URL for accessing the proxy webUI (available in 2-5 minutes)"
  value       = "https://${aws_instance.proxy_machine.public_ip}:8443/"
}
output "proxy_webui_username" {
  description = "Proxy webUI username"
  value       = var.proxy_webui_username
}
output "proxy_webui_password" {
  description = "Proxy webUI password"
  value       = random_password.proxy_webui_password.result
  sensitive   = true
}
output "proxy_machine_cert_url" {
  description = "Credentialed URL for downloading the proxy CA cert (available in 2-5 minutes)"
  value       = "https://${var.proxy_webui_username}:${random_password.proxy_webui_password.result}@${aws_instance.proxy_machine.public_ip}:8443/mitmproxy-ca-cert.pem"
  sensitive   = true
}
output "proxied_subnet_id" {
  description = "Proxied subnet ID (launch your test/'captive' instances here)"
  value       = aws_subnet.proxied.id
}
# output "public_subnet_id" {
#   description = "The ID of the Public Subnet"
#   value       = aws_subnet.public.id
# }

## DATA
# Get the current AWS region
data "aws_region" "current" {}

# Automatic lookup of the latest official RHEL 9 AMI
data "aws_ami" "rhel9" {
  most_recent = true

  filter {
    name   = "platform-details"
    values = ["Red Hat Enterprise Linux"]
  }

  filter {
    name   = "architecture"
    values = ["x86_64"]
  }

  filter {
    name   = "root-device-type"
    values = ["ebs"]
  }

  filter {
    name   = "manifest-location"
    values = ["amazon/RHEL-9.*_HVM-*-x86_64-*-Hourly2-GP2"]
  }

  owners = ["309956199498"] # Amazon's "Official Red Hat" account
}
