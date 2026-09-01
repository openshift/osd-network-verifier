#!/bin/bash

# Orchestration script that calls packer with suitable parameters to build the
# legacy golden AMI (RHEL 8 + Docker + network-validator container).
#
# Requirements:
#   AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY must be exported.
#   Optionally set CI_AWS_ACCESS_KEY_ID / CI_AWS_SECRET_ACCESS_KEY for CI
#   credential injection (falls back when primary creds are absent).

if [[ $# -eq 1 ]] ; then
  if [[ ! "$1" =~ ^[a-zA-Z0-9._/:@-]+$ ]]; then
    echo "Invalid image URI: $1"
    exit 1
  fi
  export PKR_VAR_image_uri=$1
fi

if [[ -z "${AWS_ACCESS_KEY_ID}" || -z "${AWS_SECRET_ACCESS_KEY}" ]]; then
  unset AWS_ACCESS_KEY_ID
  unset AWS_SECRET_ACCESS_KEY
  unset AWS_PROFILE
  unset AWS_SESSION_TOKEN

  if [[ -n "${CI_AWS_ACCESS_KEY_ID}" && -n "${CI_AWS_SECRET_ACCESS_KEY}" ]]  ; then
    echo "Setting variables from CI environment"
    export AWS_ACCESS_KEY_ID="${CI_AWS_ACCESS_KEY_ID}"
    export AWS_SECRET_ACCESS_KEY="${CI_AWS_SECRET_ACCESS_KEY}"
    if [[ -n "${CI_AWS_SESSION_TOKEN:-}" ]]; then
      export AWS_SESSION_TOKEN="${CI_AWS_SESSION_TOKEN}"
    fi
  else
    echo "AWS credentials not properly set and could not be determined from the environment"
    exit 2
  fi
fi

if [[ -z "${PKR_VAR_subnet_id}" ]]; then
  export PKR_VAR_subnet_id=""
fi

if [[ -z "${PKR_VAR_other_regions}" ]]; then
  export PKR_VAR_other_regions='["af-south-1", "ap-east-1", "ap-northeast-1", "ap-northeast-2", "ap-northeast-3", "ap-south-1", "ap-south-2", "ap-southeast-1", "ap-southeast-2", "ap-southeast-3", "ap-southeast-4", "ca-central-1", "eu-central-1", "eu-central-2", "eu-north-1", "eu-south-1", "eu-south-2", "eu-west-1", "eu-west-2", "eu-west-3", "me-central-1", "me-south-1", "sa-east-1", "us-east-1", "us-east-2", "us-west-1", "us-west-2"]'
fi

packer init packer/

packer build -var-file=packer/parameters.pkrvars.hcl packer/
