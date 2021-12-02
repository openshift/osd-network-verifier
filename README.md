# osd-network-verifier

A cli and set of libraries that validates the preconfigured networking components for some osd options.

## Overview

osd-network-verifier can be used prior to the installation of osd/rosa clusters to ensure the pre-requirements are valid for various network options.

### Workflow for egress

* it creates an instance in the target vpc/subnet and wait till the instance gets ready
* when the instance is ready, an `userdata` script is run in the instance. The `userdata` mainly performs 2 steps, it
    * installs appropriate packages, primarily docker
    * runs the validation image against the vpc/subnet as containerized form of https://github.com/openshift/osd-network-verifier/tree/main/build
* the output is collected via the SDK from the EC2 console output, which only includes the userdata script output because of a special line we added to the userdata to redirect the output.

### Subcommands

Take a look at https://github.com/openshift/osd-network-verifier/tree/main/cmd

### Examples

```shel
AWS_ACCESS_KEY_ID=<redacted> AWS_SECRET_ACCESS_KEY=<redacted> ./osd-network-verifier egress --subnet-id subnet-0ccetestsubnet1864 --image-id=ami-0df9a9ade3c65a1c7
```

### Dev

```shell
make build
```