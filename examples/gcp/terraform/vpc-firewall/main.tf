# configure GCP provider
provider "google" {
  credentials = file(var.credentials_file)
  project     = var.project
  region      = var.region
  zone        = var.zone
}

# create a VPC
resource "google_compute_network" "vpc_network" {
  name                    = var.vpc_name
  auto_create_subnetworks = var.auto_create_subnetworks
  routing_mode = var.routing_mode 
  # maximum transmission unit in bytes
  # default is 1460 but ranges from 1300 to 8896
  mtu = var.mtu
}

# create a public subnet
resource "google_compute_subnetwork" "public_subnet" {
  name = var.public_subnet_name
  region = var.region
  network = google_compute_network.vpc_network.self_link
  ip_cidr_range =  var.public_ip_cidr_range
  depends_on = [google_compute_network.vpc_network]
}

# create a private subnet
resource "google_compute_subnetwork" "private_subnet" {
  name = var.private_subnet_name
  region = var.region
  network = google_compute_network.vpc_network.self_link
  ip_cidr_range =  var.private_ip_cidr_range
  private_ip_google_access = var.private_ip_google_access
  depends_on = [google_compute_network.vpc_network]
}

# create cloud NAT
resource "google_compute_router" "router" {
  name = var.router_name
  network = google_compute_network.vpc_network.name
  region = var.region
}
resource "google_compute_router_nat" "nat" {
  name = var.cloud_nat_name
  router = google_compute_router.router.name
  region = google_compute_router.router.region
  # how NAT should be configured per Subnetwork
  source_subnetwork_ip_ranges_to_nat = var.source_subnetwork_ip_ranges_to_nat
}
# create firewall policy
resource "google_compute_network_firewall_policy" "fw-policy" {
  name = var.firewall_policy_name
  project = var.project
}
resource "google_compute_network_firewall_policy_rule" "rules" {
  project     = var.project
  priority       = var.priority
  direction      = var.direction
  action         = var.action
  rule_name      = var.rule_name
  firewall_policy = google_compute_network_firewall_policy.fw-policy.name
  match {
    dest_fqdns                = var.dest_fqdns
      layer4_configs {
          ip_protocol = var.ip_protocol
      }
  }
}
resource "google_compute_network_firewall_policy_association" "primary" {
  name = "my-association"
  attachment_target = google_compute_network.vpc_network.id
  firewall_policy = google_compute_network_firewall_policy.fw-policy.name
  project     = var.project
}

# outputs for network verifier
output "vpc_name" {
  value = google_compute_network.vpc_network.name
}
output "public_subnet_id" {
  value = google_compute_subnetwork.public_subnet.name
}
output "private_subnet_id" {
  value = google_compute_subnetwork.private_subnet.name
}
