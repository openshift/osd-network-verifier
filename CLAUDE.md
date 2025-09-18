# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Development Commands

- `make build`: Build the `osd-network-verifier` executable in the base directory
- `make test`: Run all tests using `go test ./...`
- `make fmt`: Format Go code and tidy modules
- `make check-fmt`: Check formatting and ensure clean git status
- `make boilerplate-update`: Update boilerplate files

## Architecture Overview

osd-network-verifier is a CLI tool for validating network configurations for ROSA and OSD CCS clusters on AWS and GCP. The tool consists of several key architectural components:

### Core Components

**CLI Structure** (`cmd/`):
- `root.go`: Main cobra command with `egress` and `dns` subcommands
- `cmd/egress/`: Egress validation subcommand
- `cmd/dns/`: DNS validation subcommand

**Verifier Service** (`pkg/verifier/`):
- Defines the `verifierService` interface with `ValidateEgress()` and `VerifyDns()` methods
- Cloud-agnostic verification orchestration
- Structured input types for validation operations

**Cloud Clients** (`pkg/clients/`):
- `aws/`: AWS-specific client implementation
- `gcp/`: GCP-specific client implementation
- `kube/`: Kubernetes client for pod-mode operations

**Probe System** (`pkg/probes/`):
- `package_probes.go`: Defines the `Probe` interface for cloud-agnostic instance behavior
- `curl/`: Default curl-based probe implementation
- `legacy/`: Legacy probe for pre-1.0 behavior compatibility
- `dummy/`: Test probe implementation
- Each probe manages machine images, user data generation, and output parsing

**Platform Abstraction** (`pkg/data/cloud/`):
- Platform type definitions (AWSClassic, AWSHCP, AWSHCPZeroEgress, GCPClassic)
- Platform-specific configuration and behavior

**Data Management** (`pkg/data/`):
- `egress_lists/`: Network endpoint lists fetched dynamically from latest commit
- `cpu/`: CPU architecture definitions
- `curlgen/`: Curl command generation utilities

### Key Design Patterns

1. **Cloud Agnostic Design**: Probes and core verification logic work across AWS/GCP
2. **Interface-Based Architecture**: `verifierService` and `Probe` interfaces enable pluggable implementations
3. **Dynamic Configuration**: Egress lists are pulled from the repository dynamically rather than hardcoded
4. **Platform Selection**: Platform types determine egress lists, machine types, and CPU architectures

### Testing and Machine Images

- Machine images are maintained per probe in `machine_images.go` files
- Image selection based on platform, region, and CPU architecture
- Default X86 architecture unless overridden with `--cpu-arch` flag
- Terraform examples available in `examples/aws/terraform/` for testing different network scenarios

### Important Files for Extension

- `pkg/verifier/package_verifier.go`: Core verification interface and input structures
- `pkg/probes/package_probes.go`: Probe interface definition
- `pkg/data/cloud/platform.go`: Platform type system
- `cmd/root.go`: Main CLI command structure