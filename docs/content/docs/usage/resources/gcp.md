---
title: Google Cloud Platform
weight: 2
---

## Overview

The GCP resource type allows KubeCloudScaler to manage Google Cloud Platform compute resources based on time periods. This enables cost optimization by automatically scaling GCP resources up or down according to your schedule.

## Authentication

KubeCloudScaler supports two authentication methods for accessing GCP resources:

### 1. Application Default Credentials (Default)
- **What it does**: Uses the default GCP authentication available in the environment
- **When to use**: When running in a GCP environment (GKE, GCE) with appropriate service account permissions
- **Configuration**: No additional setup required

**Example**: Deploy KubeCloudScaler in GKE with a service account that has compute instance permissions, and it will automatically authenticate.

### 2. Service Account Key (authSecret)
- **What it does**: Uses a service account JSON key file for authentication
- **When to use**: When you need explicit authentication or running outside GCP
- **Configuration**:
  1. Create a GCP service account with appropriate permissions
  2. Download the JSON key file
  3. Create a Kubernetes secret containing the key file
  4. Reference the secret in the `authSecret` field

**Example**:
```yaml
# 1. Create a secret with the GCP service account key
apiVersion: v1
kind: Secret
metadata:
  name: gcp-credentials
  namespace: kubecloudscaler-system
type: Opaque
data:
  service-account-key.json: <base64-encoded-json-key-file>
---
# 2. Reference the secret in your GCP scaler resource
apiVersion: kubecloudscaler.cloud/v1alpha2
kind: Gcp
metadata:
  name: gcp-scaler-example
spec:
  projectId: my-gcp-project
  region: us-central1
  authSecret: gcp-credentials
  # ... other configuration
```

**Required GCP Permissions**:
- `compute.instances.get`
- `compute.instances.list`
- `compute.instances.start`
- `compute.instances.stop`
- `compute.regions.get`
- `compute.zoneOperations.get` (if `waitForOperation` is enabled)

## Supported Resource Types

### VM Instances (Default)

KubeCloudScaler can manage GCP Compute Engine VM instances. By default, it targets all VM instances in the specified project and region.

#### Configuration Options

- **types**: Specify resource types to manage (default: `vm-instances`)
- **names**: List specific instance names to target
- **labelSelector**: Filter instances using GCP labels

**Example targeting specific instances**:
```yaml
spec:
  projectId: my-project
  region: us-central1
  resources:
    types:
      - vm-instances
    names:
      - dev-instance-1
      - dev-instance-2
```

**Example using label selectors**:
```yaml
spec:
  projectId: my-project
  region: us-central1
  resources:
    types:
      - vm-instances
    labelSelector:
      matchLabels:
        environment: development
        team: backend
```

## Configuration Options

### Required Fields

- **projectId** (string): The GCP project ID containing the resources
- **periods** (array): Time periods defining when to scale resources

### Optional Fields

- **region** (string): GCP region to target (if not specified, searches all regions)
- **authSecret** (string): Name of the Kubernetes secret containing GCP credentials
- **dryRun** (bool): Enable dry-run mode to preview actions without executing them (default: `false`)
- **restoreOnDelete** (bool): Restore resources to their original state when the scaler is deleted (default: `true`)
- **waitForOperation** (bool): Wait for GCP operations to complete before proceeding (default: `false`)
- **defaultPeriodType** (string): Default state for resources outside defined periods - `up` or `down` (default: `down`)

## Complete Configuration Examples

### Example 1: Scale Down Non-Business Hours

Stop development instances outside business hours to save costs:

```yaml
apiVersion: kubecloudscaler.cloud/v1alpha2
kind: Gcp
metadata:
  name: dev-instances-scaler
spec:
  projectId: my-dev-project
  region: us-central1
  restoreOnDelete: true
  defaultPeriodType: down
  resources:
    types:
      - vm-instances
    labelSelector:
      matchLabels:
        environment: development
  periods:
    # Keep instances running during business hours (reverse mode)
    - type: down
      time:
        recurring:
          days:
            - monday
            - tuesday
            - wednesday
            - thursday
            - friday
          startTime: "08:00"
          endTime: "18:00"
          timezone: "America/New_York"
          reverse: true
          gracePeriod: "60s"
```

### Example 2: Weekend Shutdown

Stop test instances during weekends:

```yaml
apiVersion: kubecloudscaler.cloud/v1alpha2
kind: Gcp
metadata:
  name: test-weekend-scaler
spec:
  projectId: my-test-project
  authSecret: gcp-sa-key
  dryRun: false
  restoreOnDelete: true
  waitForOperation: true
  resources:
    types:
      - vm-instances
    names:
      - test-vm-1
      - test-vm-2
      - test-vm-3
  periods:
    # Stop instances on Friday evening
    - type: down
      time:
        recurring:
          days:
            - friday
          startTime: "18:00"
          endTime: "23:59"
          timezone: "Europe/Paris"
          gracePeriod: "30s"
    # Keep stopped during weekend
    - type: down
      time:
        recurring:
          days:
            - saturday
            - sunday
          startTime: "00:00"
          endTime: "23:59"
          timezone: "Europe/Paris"
    # Start instances on Monday morning
    - type: up
      time:
        recurring:
          days:
            - monday
          startTime: "07:00"
          endTime: "07:05"
          timezone: "Europe/Paris"
          once: true
```

### Example 3: Holiday Shutdown

Stop instances during a holiday period:

```yaml
apiVersion: kubecloudscaler.cloud/v1alpha2
kind: Gcp
metadata:
  name: holiday-scaler
spec:
  projectId: my-project
  region: europe-west1
  restoreOnDelete: true
  resources:
    types:
      - vm-instances
    labelSelector:
      matchLabels:
        auto-scale: "true"
  periods:
    # Shutdown for Christmas holidays
    - type: down
      time:
        fixed:
          startTime: "2025-12-24 18:00:00"
          endTime: "2026-01-02 08:00:00"
          timezone: "Europe/Paris"
          gracePeriod: "120s"
```

## Status Monitoring

The GCP scaler reports its status including successful and failed operations:

```yaml
status:
  currentPeriod:
    type: down
    successful:
      - kind: vm-instances
        name: dev-instance-1
        comment: Successfully stopped
      - kind: vm-instances
        name: dev-instance-2
        comment: Successfully stopped
    failed:
      - kind: vm-instances
        name: dev-instance-3
        reason: Instance not found
  comments: "time period processed"
```

## Best Practices

1. **Start with Dry-Run**: Test your configuration with `dryRun: true` before applying changes
2. **Use Label Selectors**: Organize instances with labels for easier management
3. **Set Grace Periods**: Allow time for graceful shutdowns with `gracePeriod`
4. **Enable RestoreOnDelete**: Keep `restoreOnDelete: true` to avoid leaving resources in unexpected states
5. **Monitor Status**: Regularly check the status field for failed operations
6. **Regional Scope**: Specify `region` when possible to improve performance
7. **Use Default Period Type**: Set `defaultPeriodType` to define behavior outside configured periods
