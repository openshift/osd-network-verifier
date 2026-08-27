packer {
  required_plugins {
    amazon = {
      version = "= 1.3.8"
      source  = "github.com/hashicorp/amazon"
    }
  }
}

variable "ami_prefix" {
  type    = string
  default = "verifier"
}

variable "instance_type" {
  type    = string
  default = "t2.micro"
}

variable "source_region" {
  type = string
}

variable "source_ami" {
  type = string
}

variable "ssh_user" {
  type    = string
  default = "ec2-user"
}

variable "dest_regions" {
  type    = list(string)
  default = []
}

variable "version_tag" {
  type    = string
  default = ""
}

locals {
  timestamp = regex_replace(timestamp(), "[- TZ:]", "")
}

source "amazon-ebs" "verifier-image" {
  ami_name      = "${var.ami_prefix}-${var.version_tag}-${local.timestamp}"
  instance_type = var.instance_type
  region        = var.source_region
  source_ami    = var.source_ami
  ami_regions   = var.dest_regions
  ssh_username  = var.ssh_user
  ami_groups    = ["all"]
  tags = {
    red_hat_managed = "true"
    version         = var.version_tag
  }
}

build {
  name    = "verifier-image-build"
  sources = [
    "source.amazon-ebs.verifier-image"
  ]
}