# AWS GovCloud Support

## Overview

AWS GovCloud (US) is a separate AWS partition designed for U.S. government agencies and their partners. The osd-network-verifier tool supports GovCloud environments using **pod mode only**, which runs verification as a Kubernetes Job inside your cluster rather than creating EC2 instances. Pod mode is **automatically enabled** when you specify a GovCloud platform.

## Why Pod Mode for GovCloud?

- **FedRAMP Compliance**: Avoids restrictions on EC2 instance creation in GovCloud environments
- **Simplified Testing**: No need to manage AMIs, security groups, or other infrastructure
- **Network Context**: Runs verification from inside the cluster's actual network context
- **Recommended Approach**: Already used in production for GovCloud clusters

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

**Note**: The `--pod-mode` flag is optional for GovCloud platforms as it's automatically enabled.

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

## Troubleshooting

### Kubeconfig Required

GovCloud platforms automatically enable pod mode, which requires a valid kubeconfig to access your cluster.

**Solution**: Ensure you have a valid kubeconfig:
- Use `ocm backplane login <cluster-name>` to get access
- Or set `KUBECONFIG` environment variable to point to your kubeconfig file
- Or use `--kubeconfig` flag to specify the path

### Permission Denied Creating Pods

**Solution**: Ensure your kubeconfig has sufficient permissions to create Jobs/Pods in the target namespace.

### Egress List Not Found

```
failed to fetch egress URL list from https://...
```

**Solution**:
1. Verify the URL is accessible
2. Check that the file exists at the specified location
3. For GitHub URLs, ensure you're using the raw content URL

### Pod Failures

**Solution**: Use `--debug` flag to see detailed pod logs and identify which endpoints are failing.

## GovCloud vs Commercial AWS Differences

### Endpoints

GovCloud uses the same DNS suffixes as commercial AWS (`.amazonaws.com`), but with different region names:
- Commercial: `us-east-1`, `us-west-2`, etc.
- GovCloud: `us-gov-west-1`, `us-gov-east-1`

### Service Availability

Some AWS services may not be available in GovCloud. The GovCloud egress lists are tailored to reflect these differences.

### FIPS Compliance

GovCloud environments may require FIPS 140-2 validated cryptographic modules. Ensure your cluster configuration meets these requirements.

## Examples

### Full Example with Custom List

```shell
# 1. Get kubeconfig
ocm backplane login my-govcloud-cluster

# 2. Run verification with custom egress list
./osd-network-verifier egress \
  --platform aws-govcloud-classic \
  --region us-gov-west-1 \
  --egress-list-location ./my-custom-govcloud-egress.yaml \
  --debug
```

### Testing Multiple Regions

```shell
# Test us-gov-west-1
./osd-network-verifier egress \
  --platform govcloud \
  --region us-gov-west-1

# Test us-gov-east-1
./osd-network-verifier egress \
  --platform govcloud \
  --region us-gov-east-1
```

## Supported Platforms

- `aws-govcloud-classic` (or alias: `govcloud`, `aws-govcloud`)
- `aws-govcloud-hcp` (or alias: `aws-govcloud-hosted-cp`)

**Note**: GovCloud platforms automatically enable pod mode. EC2 mode is not available for GovCloud.

## See Also

- [AWS Setup Documentation](./aws.md)
- [Pod Mode Details](../../README.md#aws-govcloud-support)
- [Egress Lists](../../pkg/data/egress_lists/)
