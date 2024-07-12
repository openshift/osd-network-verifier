# osd-network-verifier clean golden ami

This module includes a Go script that loops through each region supported by the verifier. 
It determines the public image quota limit for each region and ensures there is space for at least three new public images: one for the legacy binary on AMD architecture, one for the curl binary (base RHEL9) on AMD architecture, and another for the curl binary on ARM architecture. 
The script prioritizes deleting the oldest image from the image type with the most available images until the public image count for that region has enough space for three new images.

## Usage

It can be run with AWS credentials setup as environment variables beforehand, e.g.

```bash
AMI_ACCOUNT_ID=...
export $(osdctl account cli -i ${AMI_ACCOUNT_ID} -p rhcontrol -oenv | xargs)
go build .
./cleangoldenami
```

The executable also supports the following flags:

* `--verbose` - When specified, explicitly states which regions are not at their quota.
* `--dry-run` - When specified, show which AMIs would be deregistered without actually deregistering them.