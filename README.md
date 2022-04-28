# osd-network-verifier

A cli and set of libraries that validates the pre-configured networking components for some osd options.

## Overview

osd-network-verifier can be used prior to the installation of osd/rosa clusters to ensure the pre-requirements are valid for various network options.

## Makefile Targets
- `make build`: Builds executable
- `make test`: `go test $(GOFLAGS)`
- `make build-push`: Builds and pushes image from build/ to ` quay.io/app-sre/osd-network-verifier:$(IMAGE_URI_VERSION)`
- `make skopeo-push`: (TODO add)  

## Cloud Provider Specific Readme
-  [AWS](README_AWS.md)
-  [GCP](README_GCP.md)

### Contributing and Maintenance ####
##### Egress List #####
This list of essential domains for egress verification should be maintained in `build/config/config.yaml`.
##### To Contribute #####
Fork the main repository and create pull requests against the `main` branch.

## Other Subcommands
Take a look at <https://github.com/openshift/osd-network-verifier/tree/main/cmd>
