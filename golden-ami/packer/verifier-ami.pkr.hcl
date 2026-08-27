packer {
  required_plugins {
    amazon = {
      version = "= 1.3.8"
      source  = "github.com/hashicorp/amazon"
    }
  }
}

variable "subnet_id" {
  type    = string
  default = ""
}

variable "ami_prefix" {
  type    = string
  default = "verifier"
}

variable "instance_type" {
  type    = string
  default = "t2.micro"
}

variable "build_region" {
  type    = string
  default = "us-east-1"
}

variable "other_regions" {
  type    = list(string)
  default = ["us-east-1"]
}

variable "image_owner" {
  type    = string
  default = "309956199498"
}

variable "base_image_filter" {
  type    = string
  default = "RHEL-8.4*-x86_64-*"
}

variable "ssh_user" {
  type    = string
  default = "ec2-user"
}

variable "image_uri" {
  type    = string
  default = ""
}

locals {
  timestamp = regex_replace(timestamp(), "[- TZ:]", "")
}

source "amazon-ebs" "bake-verifier-image" {
  ami_name      = "${var.ami_prefix}-legacy-x86_64-${local.timestamp}"
  instance_type = var.instance_type
  region        = var.build_region
  subnet_id     = var.subnet_id
  source_ami_filter {
    filters = {
      name                = var.base_image_filter
      root-device-type    = "ebs"
      virtualization-type = "hvm"
    }
    most_recent = true
    owners      = [var.image_owner]
  }
  ami_block_device_mappings {
    device_name = "/dev/sda1"
    volume_size = "10"
    volume_type = "gp3"
    delete_on_termination = true
  }
  ami_regions  = var.other_regions
  ssh_username = var.ssh_user
  ami_groups = ["all"]
  tags = {
    red_hat_managed = "true"
    version         = "legacy-x86_64"
  }
}

build {
  name = "bake-verifier-image-build"
  sources = [
    "source.amazon-ebs.bake-verifier-image"
  ]

  post-processor "manifest" {
    output     = "manifest.json"
    strip_path = true
  }

  provisioner "ansible" {
    use_proxy       = false
    playbook_file   = "${path.root}/provisioner.yaml"
  }

  provisioner "shell" {
    inline = [
      "sudo yum upgrade -y",
      "sudo cloud-init clean",
      "sudo sed -i 's/GRUB_CMDLINE_LINUX=\"console=ttyS0,115200n8 console=tty0/GRUB_CMDLINE_LINUX=\"console=tty1 console=ttyS0/' /etc/default/grub",
      "sudo grub2-mkconfig -o /boot/grub2/grub.cfg",
      "sudo sysctl -w kernel.printk=\"3 4 1 3\"",
      "sudo docker pull ${var.image_uri}"
    ]
  }
}
