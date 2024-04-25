# osd-network-verifier clean golden ami

This module contains a Go script
that will loop through each region supported by the verifier
and delete the oldest public AMI if the resource quota is at capacity.

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