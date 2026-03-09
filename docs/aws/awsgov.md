# AWS GovCloud Support

## Overview

AWS GovCloud (US) is a separate AWS partition designed for U.S. government agencies and their partners. The osd-network-verifier tool supports GovCloud environments using **pod mode only**, which runs verification as a Kubernetes Job inside your cluster rather than creating EC2 instances. Pod mode is **automatically enabled** when you specify a GovCloud platform.

## Setup

### Prerequisites

1. **Access to a GovCloud cluster**: You need a running OpenShift cluster in AWS GovCloud
2. **Kubeconfig**: Valid kubeconfig with access to the cluster
   ```shell
   # Using OCM backplane
   ocm backplane login <cluster-name>

   # Or set KUBECONFIG directly
   export KUBECONFIG=/path/to/govcloud-cluster-kubeconfig
   ```

3. **osd-network-verifier binary**: Build or download the binary
   ```shell
   git clone git@github.com:openshift/osd-network-verifier.git
   cd osd-network-verifier
   make build
   ```

## Usage

### Basic GovCloud Verification

```shell
./osd-network-verifier egress \
  --platform aws-govcloud-classic \
  --region us-gov-west-1
```

### Using Platform Aliases

```shell
# Short alias for aws-govcloud-classic
./osd-network-verifier egress \
  --platform govcloud \
  --region us-gov-west-1
```

### GovCloud HCP (Hosted Control Plane)

```shell
./osd-network-verifier egress \
  --platform aws-govcloud-hcp \
  --region us-gov-west-1
```

### Supported Regions

- `us-gov-west-1` (US West)
- `us-gov-east-1` (US East)

## Custom Egress Lists

You may need to use a custom egress list for testing or when endpoints are not yet merged into the main repository.

### Option 1: Local File

Use a local YAML file on your filesystem:

```shell
./osd-network-verifier egress \
  --platform aws-govcloud-classic \
  --region us-gov-west-1 \
  --egress-list-location /path/to/custom-egress-list.yaml
```

Example using the repository's local file:
```shell
./osd-network-verifier egress \
  --platform aws-govcloud-classic \
  --region us-gov-west-1 \
  --egress-list-location pkg/data/egress_lists/aws-govcloud-classic.yaml
```

### Option 2: GitHub Repository (Fork or Branch)

Use a raw GitHub URL from your fork or a specific branch:

```shell
./osd-network-verifier egress \
  --platform aws-govcloud-classic \
  --region us-gov-west-1 \
  --egress-list-location https://raw.githubusercontent.com/YOUR_USERNAME/osd-network-verifier/YOUR_BRANCH/pkg/data/egress_lists/aws-govcloud-classic.yaml
```

Example using a specific branch:
```shell
./osd-network-verifier egress \
  --platform aws-govcloud-classic \
  --region us-gov-west-1 \
  --egress-list-location https://raw.githubusercontent.com/annelson-rh/osd-network-verifier/HCMSEC-1647/pkg/data/egress_lists/aws-govcloud-classic.yaml
```

### Option 3: External URL

Any HTTP(S) accessible YAML file:

```shell
./osd-network-verifier egress \
  --platform aws-govcloud-classic \
  --region us-gov-west-1 \
  --egress-list-location https://example.com/my-custom-egress-list.yaml
```

## Custom Egress List Format

Your custom egress list should follow this YAML format:

```yaml
endpoints:
  - host: registry.redhat.io
    ports:
      - 443
  - host: api.openshift.com
    ports:
      - 443
  - host: sts.${AWS_REGION}.amazonaws.com
    ports:
      - 443
  # Add more endpoints as needed
```

**Variable Substitution**: The `${AWS_REGION}` variable will be automatically replaced with the region specified via `--region` flag.

**TLS Disabled**: For endpoints that don't require TLS verification:
```yaml
  - host: cert-api.access.redhat.com
    ports:
      - 443
    tlsDisabled: true
```

## Additional Options

### Custom Namespace

By default, verification pods run in `openshift-network-diagnostics` namespace. To use a different namespace:

```shell
./osd-network-verifier egress \
  --platform aws-govcloud-classic \
  --region us-gov-west-1 \
  --namespace my-custom-namespace
```

### Debug Mode

Enable verbose logging to see detailed verification output:

```shell
./osd-network-verifier egress \
  --platform aws-govcloud-classic \
  --region us-gov-west-1 \
  --debug
```

### Proxy Configuration

If your cluster uses an HTTP(S) proxy:

```shell
./osd-network-verifier egress \
  --platform aws-govcloud-classic \
  --region us-gov-west-1 \
  --http-proxy http://proxy.example.com:8080 \
  --https-proxy https://proxy.example.com:8443 \
  --no-proxy "localhost,127.0.0.1,.svc,.cluster.local"
```
