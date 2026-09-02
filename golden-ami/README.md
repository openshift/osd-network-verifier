# osd-network-verifier Golden AMI

Packer-based solution that bakes golden AMI images for
[osd-network-verifier](https://github.com/openshift/osd-network-verifier)
across all supported AWS regions.

## Overview

This directory contains two Packer workflows:

1. **Legacy AMI** (`packer/`) - Builds an AMI from a RHEL 8 base image with
   Docker installed and the network-validator container pre-pulled. Used by the
   legacy probe.
2. **RHEL repackage AMI** (`packer_repackage/`) - Copies the latest RHEL 9 AMI
   from Red Hat's official account into all supported regions. Used by the curl
   probe.

## Prerequisites

- [Packer](https://www.packer.io/) (with the HashiCorp Amazon plugin v1.3.8)
- AWS credentials with permissions to create/copy/publish AMIs
- AWS CLI (for `copy-rhel-ami.sh`)

## Usage

### Build RHEL repackage AMIs (primary workflow)

```bash
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...

# Build for x86_64
./copy-rhel-ami.sh x86_64

# Build for arm64
./copy-rhel-ami.sh arm64
```

### Build legacy AMI

```bash
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
export PKR_VAR_image_uri=quay.io/app-sre/osd-network-verifier:<tag>

./run.sh
```

Or pass the image URI as an argument:

```bash
./run.sh quay.io/app-sre/osd-network-verifier:<tag>
```

### Makefile targets

```bash
make copy-rhel-ami-x86_64    # RHEL 9 repackage for x86_64
make copy-rhel-ami-arm64     # RHEL 9 repackage for arm64
make build-all-amis-ci       # Both RHEL repackage AMIs (used by CI)
```

## AMI Quota Management

The quotas for public AMIs in a given AWS account are limited per region (typically 20).
When the quota is reached, CI builds will fail until cleanup is performed.

Use the [cleangoldenami](../cleangoldenami/) tool in this repository to perform cleanup:

```bash
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
cd ../cleangoldenami
go build .
./cleangoldenami
```

See [cleangoldenami/README.md](../cleangoldenami/README.md) for detailed instructions.

## Adding a New Region

When adding a new AWS region:

1. Enable the region in the AMI build AWS account
2. Disable public AMI block for the new region:
   ```bash
   aws ec2 disable-image-block-public-access --region <REGION>
   ```
3. Add the region to the `PKR_VAR_dest_regions` list in `copy-rhel-ami.sh`
4. Add the region to the `PKR_VAR_other_regions` list in `run.sh` (if applicable)

## Testing

Packer configuration can be validated with:

```bash
packer validate packer/
packer validate packer_repackage/
```
