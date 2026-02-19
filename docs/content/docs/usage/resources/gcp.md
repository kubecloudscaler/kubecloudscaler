---
title: Google Cloud Platform
weight: 2
---

## Overview

The Gcp resource type allows KubeCloudScaler to manage Google Cloud Platform compute resources based on time periods. This enables cost optimization by automatically starting and stopping GCP resources according to your schedule.

## Spec Structure

```yaml
apiVersion: kubecloudscaler.cloud/v1alpha3
kind: Gcp
metadata:
  name: my-gcp-scaler       # Cluster-scoped, no namespace needed
spec:
  dryRun: false              # Optional: preview mode
  periods: [...]             # Required: time-based scaling rules
  resources:                 # Required: what to scale
    types: [...]
    names: [...]
    labelSelector: { ... }
  config:                    # GCP-specific settings
    projectId: ""            # Required: GCP project ID
    region: ""               # Optional: GCP region
    authSecret: null         # Optional: secret for GCP credentials
    restoreOnDelete: true
    waitForOperation: false
    defaultPeriodType: "down"
```

## Authentication

KubeCloudScaler supports two authentication methods for accessing GCP resources:

### 1. Application Default Credentials (Default)
- **What it does**: Uses the default GCP authentication available in the environment
- **When to use**: When running in a GCP environment (GKE, GCE) with appropriate service account permissions
- **Configuration**: No additional setup required

**Example**: Deploy KubeCloudScaler in GKE with Workload Identity or a node service account that has compute instance permissions, and it will automatically authenticate.

### 2. Service Account Key (authSecret)
- **What it does**: Uses a service account JSON key file for authentication
- **When to use**: When you need explicit authentication or running outside GCP
- **Configuration**:
  1. Create a GCP service account with appropriate permissions
  2. Download the JSON key file
  3. Create a Kubernetes secret containing the key file
  4. Reference the secret in the `config.authSecret` field

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
apiVersion: kubecloudscaler.cloud/v1alpha3
kind: Gcp
metadata:
  name: gcp-scaler-example
spec:
  periods: [...]
  resources:
    types:
      - vm-instances
  config:
    projectId: my-gcp-project
    region: us-central1
    authSecret: gcp-credentials
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

#### Resource Selection

- **types**: Specify resource types to manage (default: `vm-instances`)
- **names**: List specific instance names to target
- **labelSelector**: Filter instances using GCP labels

**Example targeting specific instances**:
```yaml
spec:
  resources:
    types:
      - vm-instances
    names:
      - dev-instance-1
      - dev-instance-2
  config:
    projectId: my-project
    region: us-central1
```

**Example using label selectors**:
```yaml
spec:
  resources:
    types:
      - vm-instances
    labelSelector:
      matchLabels:
        environment: development
        team: backend
  config:
    projectId: my-project
    region: us-central1
```

## Configuration Options

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `periods` | `[]ScalerPeriod` | Time periods defining when to scale resources |
| `resources` | `Resources` | Resource types and filters to target |
| `config.projectId` | `string` | The GCP project ID containing the resources |

### Optional Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `dryRun` | `bool` | `false` | Preview actions without executing them |
| `config.region` | `string` | all regions | GCP region to target |
| `config.authSecret` | `string` | none | Name of the Kubernetes secret containing GCP credentials |
| `config.restoreOnDelete` | `bool` | `true` | Restore resources to their original state when the scaler is deleted |
| `config.waitForOperation` | `bool` | `false` | Wait for GCP operations to complete before proceeding |
| `config.defaultPeriodType` | `string` | `down` | Default state for resources outside defined periods (`up` or `down`) |

## Complete Configuration Examples

### Example 1: Scale Down Non-Business Hours

Stop development instances outside business hours to save costs:

```yaml
apiVersion: kubecloudscaler.cloud/v1alpha3
kind: Gcp
metadata:
  name: dev-instances-scaler
spec:
  resources:
    types:
      - vm-instances
    labelSelector:
      matchLabels:
        environment: development
  config:
    projectId: my-dev-project
    region: us-central1
    restoreOnDelete: true
    defaultPeriodType: down
  periods:
    - type: "down"
      name: "outside-business-hours"
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
apiVersion: kubecloudscaler.cloud/v1alpha3
kind: Gcp
metadata:
  name: test-weekend-scaler
spec:
  dryRun: false
  resources:
    types:
      - vm-instances
    names:
      - test-vm-1
      - test-vm-2
      - test-vm-3
  config:
    projectId: my-test-project
    authSecret: gcp-sa-key
    restoreOnDelete: true
    waitForOperation: true
  periods:
    - type: "down"
      name: "friday-evening"
      time:
        recurring:
          days:
            - friday
          startTime: "18:00"
          endTime: "23:59"
          timezone: "Europe/Paris"
          gracePeriod: "30s"
    - type: "down"
      name: "weekend"
      time:
        recurring:
          days:
            - saturday
            - sunday
          startTime: "00:00"
          endTime: "23:59"
          timezone: "Europe/Paris"
    - type: "up"
      name: "monday-start"
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
apiVersion: kubecloudscaler.cloud/v1alpha3
kind: Gcp
metadata:
  name: holiday-scaler
spec:
  resources:
    types:
      - vm-instances
    labelSelector:
      matchLabels:
        auto-scale: "true"
  config:
    projectId: my-project
    region: europe-west1
    restoreOnDelete: true
  periods:
    - type: "down"
      name: "christmas"
      time:
        fixed:
          startTime: "2026-12-24 18:00:00"
          endTime: "2027-01-02 08:00:00"
          timezone: "Europe/Paris"
          gracePeriod: "120s"
```

## Status Monitoring

The Gcp scaler reports its status including successful and failed operations:

```yaml
status:
  currentPeriod:
    type: down
    name: "outside-business-hours"
    spec:
      days:
        - monday
        - friday
      startTime: "08:00"
      endTime: "18:00"
      timezone: "America/New_York"
    specSHA: abc123def456
    success:
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
4. **Enable RestoreOnDelete**: Keep `config.restoreOnDelete: true` to avoid leaving resources in unexpected states
5. **Monitor Status**: Regularly check the status field for failed operations
6. **Regional Scope**: Specify `config.region` when possible to improve performance
7. **Use Default Period Type**: Set `config.defaultPeriodType` to define behavior outside configured periods
8. **Workload Identity**: Prefer GKE Workload Identity over service account keys for authentication
