#!/bin/bash

# Copies the latest RHEL 9 AMI from the Red Hat account into all supported
# regions. This is the primary CI entry point for producing golden AMIs.
#
# Usage: ./copy-rhel-ami.sh [x86_64|arm64]
#
# Requirements:
#   AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY must be exported.
#   Optionally set CI_AWS_ACCESS_KEY_ID / CI_AWS_SECRET_ACCESS_KEY for CI
#   credential injection (falls back when primary creds are absent).

SOURCE_REGION=us-east-2
# Red Hat's official AMI owner account
OWNER_ACCOUNT=309956199498

if [[ $# -eq 1 ]]; then
    export IMAGE_ARCHITECTURE=$1
else
    export IMAGE_ARCHITECTURE="x86_64"
fi

if [[ ${IMAGE_ARCHITECTURE} != "x86_64" && ${IMAGE_ARCHITECTURE} != "arm64" ]]; then
    echo "\"${IMAGE_ARCHITECTURE}\" is not a supported architecture. Use either \"x86_64\" or \"arm64\"."
    exit 1
fi

if [[ ${IMAGE_ARCHITECTURE} == "x86_64" ]]; then
    export PKR_VAR_instance_type=t2.micro
elif [[ ${IMAGE_ARCHITECTURE} == "arm64" ]]; then
    export PKR_VAR_instance_type=m7g.medium
fi

export PKR_VAR_version_tag="rhel-$IMAGE_ARCHITECTURE"

if [[ -z "${AWS_ACCESS_KEY_ID}" || -z "${AWS_SECRET_ACCESS_KEY}" ]]; then
    unset AWS_ACCESS_KEY_ID
    unset AWS_SECRET_ACCESS_KEY
    unset AWS_PROFILE
    unset AWS_SESSION_TOKEN

    if [[ -n "${CI_AWS_ACCESS_KEY_ID}" && -n "${CI_AWS_SECRET_ACCESS_KEY}" ]]; then
        echo "Setting variables from CI environment"
        export AWS_ACCESS_KEY_ID="${CI_AWS_ACCESS_KEY_ID}"
        export AWS_SECRET_ACCESS_KEY="${CI_AWS_SECRET_ACCESS_KEY}"
    else
        echo "AWS credentials not properly set and could not be determined from the environment"
        exit 2
    fi
fi

if [[ -z "${PKR_VAR_source_ami}" ]]; then
    PKR_VAR_source_ami=$(aws ec2 describe-images \
        --owners $OWNER_ACCOUNT \
        --filters "Name=platform-details,Values='Red Hat Enterprise Linux'" "Name=architecture,Values=${IMAGE_ARCHITECTURE}" "Name=root-device-type,Values=ebs" "Name=manifest-location,Values=amazon/RHEL-9.*_HVM-*-${IMAGE_ARCHITECTURE}-*-Hourly2-GP2" \
        --region=$SOURCE_REGION \
        --output text \
        --query 'sort_by(Images, &CreationDate)[-1].ImageId')
    export PKR_VAR_source_ami
fi

echo -e "source account:\t$OWNER_ACCOUNT"
echo -e "source image:\t$PKR_VAR_source_ami"
echo -e "source region:\t$SOURCE_REGION"
echo "Copying source to specified regions in account $(aws sts get-caller-identity --output text --query Account)..."

export PKR_VAR_source_region=$SOURCE_REGION

if [[ -z "${PKR_VAR_dest_regions}" ]]; then
    export PKR_VAR_dest_regions='["af-south-1", "ap-east-1", "ap-northeast-1", "ap-northeast-2", "ap-northeast-3", "ap-south-1", "ap-south-2", "ap-southeast-1", "ap-southeast-2", "ap-southeast-3", "ap-southeast-4", "ap-southeast-5", "ap-southeast-6", "ap-southeast-7", "ca-central-1", "ca-west-1", "eu-central-1", "eu-central-2", "eu-north-1", "eu-south-1", "eu-south-2", "eu-west-1", "eu-west-2", "eu-west-3", "il-central-1", "me-central-1", "me-south-1", "mx-central-1", "sa-east-1", "us-east-1", "us-east-2", "us-west-1", "us-west-2"]'
fi

packer init packer_repackage/
packer build packer_repackage/
