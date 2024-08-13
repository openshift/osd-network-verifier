## terraform variables
# configure GCP provider
variable "project" {}
variable "credentials_file" {}
variable "region" {
  type = string
  default = "us-east1"
}
variable "zone" {
  type = string
  default = "us-east1-a-a"
}

# create VPC
variable "vpc_name" {
  type = string
  default = "my-vpc"
}
variable "auto_create_subnetworks" {
  type = string
  default = "false"
}
variable "routing_mode" {
  type = string
  default = "GLOBAL"
}
variable "mtu" {
  type = number
  default = 1460
}

# create public subnet
variable "public_subnet_name" {
  type = string
  default = "my-public-subnet"
}
variable "public_ip_cidr_range" {
  type = string
  default = "10.0.1.0/24"
}

# create private subnet
variable "private_subnet_name" {
  type = string
  default = "my-private-subnet"
}
variable "private_ip_cidr_range" {
  type = string
  default = "10.0.2.0/24"
}
variable "private_ip_google_access" {
  type = string
  default = "true"
}

# create cloud NAT
variable "router_name" {
  type = string
  default = "my-router"
}
variable "asn" {
  type = number
  default = 64514
}
variable "cloud_nat_name" {
  type = string
  default = "my-cloud-nat"
}
variable "nat_ip_allocate_option" {
  type = string
  default = "AUTO_ONLY"
}
variable "source_subnetwork_ip_ranges_to_nat" {
  type = string
  default = "ALL_SUBNETWORKS_ALL_IP_RANGES"
}

# create firewall policy
variable "firewall_policy_name" {
  type = string
  default = "my-firewall-policy"
}
variable "priority" {
  type = string
  default = "600"
}
variable "direction" {
  type = string
  default = "EGRESS"
}
variable "action" {
  type = string
  default = "deny"
}
variable "rule_name" {
  type = string
  default = "deny-egress-domains"
}
variable "dest_fqdns" {
  type = list(string)
  default = ["quay.io", "cdn01.quay.io"]
}
variable "ip_protocol" {
  type = string
  default = "all"
}
