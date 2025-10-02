---
title: Kubernetes
weight: 1
---

## Overview

The Kubernetes (K8s) resource type allows KubeCloudScaler to manage Kubernetes workload resources based on time periods. This enables cost optimization by automatically scaling workloads up or down according to your schedule, particularly useful for development, staging, and test environments.

## Authentication

KubeCloudScaler supports multiple authentication methods for accessing Kubernetes clusters:

### 1. InCluster (Default)
- **What it does**: KubeCloudScaler manages resources in the local cluster where it's deployed
- **When to use**: Default mode for single-cluster deployments
- **Configuration**: No additional setup required

**Example**: Deploy KubeCloudScaler with appropriate RBAC permissions, and it will automatically use the pod's service account to access cluster resources.

### 2. KUBECONFIG Environment Variable
- **What it does**: Uses a kubeconfig file to connect to remote clusters
- **When to use**: When you have kubeconfig files for remote clusters
- **Configuration**:
  1. Mount a secret containing the kubeconfig as a volume
  2. Set the `KUBECONFIG` environment variable to point to the mounted file

**Example**:
```yaml
# Create a secret with your kubeconfig
apiVersion: v1
kind: Secret
metadata:
  name: remote-cluster-kubeconfig
  namespace: kubecloudscaler-system
type: Opaque
data:
  config: <base64-encoded-kubeconfig-content>
---
# Deploy KubeCloudScaler with the kubeconfig mounted
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kubecloudscaler
  namespace: kubecloudscaler-system
spec:
  template:
    spec:
      containers:
      - name: kubecloudscaler
        env:
        - name: KUBECONFIG
          value: /etc/kubeconfig/config
        volumeMounts:
        - name: kubeconfig-volume
          mountPath: /etc/kubeconfig
          readOnly: true
      volumes:
      - name: kubeconfig-volume
        secret:
          secretName: remote-cluster-kubeconfig
```

### 3. Service Account Token (authSecret)
> [!NOTE]
> This authentication method is currently in development and not fully implemented in v1alpha2.

- **What it does**: Uses a service account token to authenticate with remote clusters
- **When to use**: When you need long-lived authentication to remote clusters
- **Current Status**: The controller recognizes the `authSecret` field but does not yet support remote cluster authentication via secrets

## Supported Resource Types

KubeCloudScaler can manage various types of Kubernetes workload resources. By default, it targets all **deployments** across all namespaces (excluding system namespaces).

### Available Resource Types

- **deployments** (default) - Standard Kubernetes Deployments
- **statefulsets** - StatefulSets for stateful applications
- **cronjobs** - Scheduled job resources

> [!WARNING]
> Application resources (deployments, statefulsets, cronjobs) cannot be managed simultaneously with HPA resources as they serve conflicting purposes. The controller validates this at runtime.

### Resource Selection

Resources can be targeted using multiple methods:

#### By Resource Type

```yaml
spec:
  resources:
    types:
      - deployments
      - statefulsets
```

#### By Name

Target specific resources by name:

```yaml
spec:
  resources:
    types:
      - deployments
    names:
      - api-server
      - web-frontend
      - worker-service
```

#### By Label Selector

Use Kubernetes [labelSelector](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/label-selector/#LabelSelector) for flexible filtering:

```yaml
spec:
  resources:
    types:
      - deployments
    labelSelector:
      matchLabels:
        environment: development
        auto-scale: "true"
      matchExpressions:
        - key: tier
          operator: In
          values:
            - backend
            - frontend
```

## Namespace Selection

Control which namespaces are included or excluded from scaling operations.

### Default Behavior

By default, all namespaces are included except system namespaces (`kube-system`, `kube-public`, `kube-node-lease`).

### Configuration Options

- **namespaces**: Specify exact namespaces to include (when set, only these namespaces are targeted)
- **excludeNamespaces**: Exclude specific namespaces from selection
- **forceExcludeSystemNamespaces**: Ensure system namespaces are always excluded (default: `false`)

**Example - Target Specific Namespaces**:
```yaml
spec:
  namespaces:
    - development
    - staging
  resources:
    types:
      - deployments
```

**Example - Exclude Specific Namespaces**:
```yaml
spec:
  excludeNamespaces:
    - production
    - critical-services
  forceExcludeSystemNamespaces: true
  resources:
    types:
      - deployments
```

## Configuration Options

### Required Fields

- **periods** (array): Time periods defining when to scale resources

### Optional Fields

- **resources**: Resource types and filters (default: deployments in all non-system namespaces)
  - **types** ([]string): Resource types to manage
  - **names** ([]string): Specific resource names to target
  - **labelSelector**: Label-based resource filtering
- **namespaces** ([]string): Specific namespaces to target
- **excludeNamespaces** ([]string): Namespaces to exclude
- **forceExcludeSystemNamespaces** (bool): Always exclude system namespaces (default: `false`)
- **dryRun** (bool): Enable dry-run mode to preview actions (default: `false`)
- **restoreOnDelete** (bool): Restore resources to original state when scaler is deleted (default: `true`)
- **disableEvents** (bool): Disable Kubernetes event generation (default: `false`)
- **deploymentTimeAnnotation** (string): Custom annotation for tracking deployment time

## Time Periods

Periods define when resources should be scaled up or down. KubeCloudScaler supports both recurring and fixed time periods.

### Period Configuration

Each period consists of:
- **type** (string): Action to perform - `up` or `down`
- **time**: Time configuration (recurring or fixed)
- **minReplicas** (int32): Minimum replica count when scaling down
- **maxReplicas** (int32): Maximum replica count when scaling up

### Recurring Periods

For schedules that repeat regularly:

```yaml
periods:
  - type: down
    minReplicas: 0
    time:
      recurring:
        days:
          - monday
          - tuesday
          - wednesday
          - thursday
          - friday
        startTime: "19:00"
        endTime: "07:00"
        timezone: "Europe/Paris"
        reverse: false
        once: false
        gracePeriod: "30s"
```

**Fields**:
- **days**: Days when the period applies (`monday`, `tuesday`, `wednesday`, `thursday`, `friday`, `saturday`, `sunday`, or `all`)
- **startTime**: Start time in HH:MM format (24-hour)
- **endTime**: End time in HH:MM format (24-hour)
- **timezone**: Timezone name (e.g., `America/New_York`, `Europe/Paris`, `Asia/Tokyo`)
- **reverse**: Invert the period (scale during times outside the range)
- **once**: Run only once at startTime, then wait until endTime
- **gracePeriod**: Wait period before scaling (e.g., `30s`, `5m`, `1h`)

### Fixed Periods

For one-time schedules with specific dates:

```yaml
periods:
  - type: down
    minReplicas: 0
    time:
      fixed:
        startTime: "2025-12-24 18:00:00"
        endTime: "2025-12-26 08:00:00"
        timezone: "Europe/Paris"
        once: false
        gracePeriod: "60s"
```

**Fields**:
- **startTime**: Start date and time in `YYYY-MM-DD HH:MM:SS` format
- **endTime**: End date and time in `YYYY-MM-DD HH:MM:SS` format
- **timezone**: Timezone name
- **once**: Run only once at startTime
- **gracePeriod**: Wait period before scaling

## Integration with ArgoCD

When using [Argo-CD](https://argo-cd.readthedocs.io/en/stable/user-guide/diffing/) for GitOps workflows, you may encounter out-of-sync issues due to KubeCloudScaler's resource modifications. To resolve this, configure ArgoCD to ignore differences in `managedFields`:

```yaml
resource.customizations.ignoreDifferences.all: |
  managedFieldsManagers:
    - kubecloudscaler
```

## Complete Configuration Examples

### Example 1: Scale Down Development Environment After Hours

Scale down all development deployments outside business hours:

```yaml
apiVersion: kubecloudscaler.cloud/v1alpha2
kind: K8s
metadata:
  name: dev-environment-scaler
spec:
  restoreOnDelete: true
  namespaces:
    - development
  resources:
    types:
      - deployments
      - statefulsets
  periods:
    # Scale down after business hours (reverse mode keeps things up during the day)
    - type: down
      minReplicas: 0
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

### Example 2: Weekend Shutdown for Test Services

Completely shut down test services during weekends:

```yaml
apiVersion: kubecloudscaler.cloud/v1alpha2
kind: K8s
metadata:
  name: test-weekend-shutdown
spec:
  restoreOnDelete: true
  dryRun: false
  namespaces:
    - test
    - qa
  excludeNamespaces:
    - test-production
  resources:
    types:
      - deployments
    labelSelector:
      matchLabels:
        auto-scale: "enabled"
  periods:
    # Shutdown Friday evening
    - type: down
      minReplicas: 0
      time:
        recurring:
          days:
            - friday
          startTime: "18:00"
          endTime: "23:59"
          timezone: "UTC"
          gracePeriod: "120s"
    # Keep down all weekend
    - type: down
      minReplicas: 0
      time:
        recurring:
          days:
            - saturday
            - sunday
          startTime: "00:00"
          endTime: "23:59"
          timezone: "UTC"
    # Start up Monday morning
    - type: up
      time:
        recurring:
          days:
            - monday
          startTime: "07:00"
          endTime: "07:05"
          timezone: "UTC"
          once: true
```

### Example 3: Lunch Break Scaling

Scale down non-critical services during lunch hours:

```yaml
apiVersion: kubecloudscaler.cloud/v1alpha2
kind: K8s
metadata:
  name: lunch-break-scaler
spec:
  restoreOnDelete: true
  resources:
    types:
      - deployments
    labelSelector:
      matchLabels:
        priority: low
        environment: staging
  forceExcludeSystemNamespaces: true
  periods:
    - type: down
      minReplicas: 1
      time:
        recurring:
          days:
            - all
          startTime: "12:00"
          endTime: "14:00"
          timezone: "Europe/Paris"
          gracePeriod: "30s"
```

### Example 4: Holiday Shutdown

Scale down for a specific holiday period:

```yaml
apiVersion: kubecloudscaler.cloud/v1alpha2
kind: K8s
metadata:
  name: christmas-holiday-scaler
spec:
  restoreOnDelete: true
  namespaces:
    - development
    - staging
  resources:
    types:
      - deployments
      - statefulsets
      - cronjobs
  periods:
    - type: down
      minReplicas: 0
      time:
        fixed:
          startTime: "2025-12-24 18:00:00"
          endTime: "2026-01-02 08:00:00"
          timezone: "America/Los_Angeles"
          gracePeriod: "300s"
```

### Example 5: Multi-Period Complex Schedule

Combine multiple periods for complex scheduling:

```yaml
apiVersion: kubecloudscaler.cloud/v1alpha2
kind: K8s
metadata:
  name: complex-schedule-scaler
spec:
  restoreOnDelete: true
  disableEvents: false
  resources:
    types:
      - deployments
    names:
      - api-service
      - worker-queue
      - batch-processor
  periods:
    # Night time scale down (weekdays)
    - type: down
      minReplicas: 1
      time:
        recurring:
          days:
            - monday
            - tuesday
            - wednesday
            - thursday
            - friday
          startTime: "22:00"
          endTime: "06:00"
          timezone: "Europe/London"
          gracePeriod: "60s"
    # Lunch time minimal scaling
    - type: down
      minReplicas: 2
      time:
        recurring:
          days:
            - monday
            - tuesday
            - wednesday
            - thursday
            - friday
          startTime: "12:30"
          endTime: "13:30"
          timezone: "Europe/London"
          gracePeriod: "30s"
    # Weekend complete shutdown
    - type: down
      minReplicas: 0
      time:
        recurring:
          days:
            - saturday
            - sunday
          startTime: "00:00"
          endTime: "23:59"
          timezone: "Europe/London"
```

## Status Monitoring

The K8s scaler reports its status including successful and failed operations:

```yaml
status:
  currentPeriod:
    type: down
    spec:
      days:
        - monday
        - tuesday
        - wednesday
        - thursday
        - friday
      startTime: "19:00"
      endTime: "07:00"
      timezone: "Europe/Paris"
    specSHA: abc123def456
    successful:
      - kind: deployments
        name: api-server
        comment: Scaled to 0 replicas
      - kind: deployments
        name: web-frontend
        comment: Scaled to 0 replicas
    failed:
      - kind: deployments
        name: worker-service
        reason: "Deployment not found in namespace"
  comments: "time period processed"
```

## Best Practices

1. **Start with Dry-Run**: Test your configuration with `dryRun: true` before applying changes to production
2. **Use Label Selectors**: Organize resources with labels like `auto-scale: enabled` for easier management
3. **Set Minimum Replicas**: Use `minReplicas: 1` instead of `0` for critical services to maintain availability
4. **Configure Grace Periods**: Allow time for graceful shutdowns with appropriate `gracePeriod` values
5. **Enable RestoreOnDelete**: Keep `restoreOnDelete: true` (default) to avoid leaving resources in unexpected states
6. **Namespace Isolation**: Use specific namespaces to avoid accidentally scaling production resources
7. **Monitor Status**: Regularly check the status field for failed operations and adjust configuration
8. **Test Period Logic**: Verify your time periods work as expected, especially when using `reverse` mode
9. **Consider Time Zones**: Always specify the correct timezone for your schedule
10. **Resource Exclusions**: Use `excludeNamespaces` or label selectors to protect critical resources

## Troubleshooting

### Resources Not Scaling

- Verify the scaler has appropriate RBAC permissions
- Check that resource names and namespaces are correct
- Ensure label selectors match your resources
- Review the status field for error messages

### Incorrect Timing

- Verify the timezone is set correctly
- Check that period times don't overlap unexpectedly
- Ensure `reverse` mode is used correctly

### ArgoCD Conflicts

- Configure ArgoCD to ignore `managedFields` as shown above
- Consider using ArgoCD's `ignoreDifferences` for replica counts
