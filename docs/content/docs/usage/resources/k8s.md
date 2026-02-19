---
title: Kubernetes
weight: 1
---

## Overview

The K8s resource type allows KubeCloudScaler to manage Kubernetes workload resources based on time periods. This enables cost optimization by automatically scaling workloads up or down according to your schedule, particularly useful for development, staging, and test environments.

## Spec Structure

```yaml
apiVersion: kubecloudscaler.cloud/v1alpha3
kind: K8s
metadata:
  name: my-k8s-scaler      # Cluster-scoped, no namespace needed
spec:
  dryRun: false             # Optional: preview mode
  periods: [...]            # Required: time-based scaling rules
  resources:                # Required: what to scale
    types: [...]
    names: [...]
    labelSelector: { ... }
  config:                   # Optional: K8s-specific settings
    namespaces: [...]
    excludeNamespaces: [...]
    forceExcludeSystemNamespaces: true
    restoreOnDelete: true
    disableEvents: false
    deploymentTimeAnnotation: ""
    authSecret: null
```

## Authentication

KubeCloudScaler supports multiple authentication methods for accessing Kubernetes clusters:

### 1. InCluster (Default)
- **What it does**: KubeCloudScaler manages resources in the local cluster where it's deployed
- **When to use**: Default mode for single-cluster deployments
- **Configuration**: No additional setup required

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
> This authentication method allows connecting to remote clusters via a service account token stored in a Kubernetes secret.

- **What it does**: Uses a service account token to authenticate with remote clusters
- **When to use**: When you need long-lived authentication to remote clusters
- **Configuration**: Set the `config.authSecret` field to the name of a Kubernetes secret containing the token

> [!IMPORTANT]
> Since `K8s` is a cluster-scoped CRD, the secret must be created in the **operator namespace** (`kubecloudscaler-system` by default, or the value of the `POD_NAMESPACE` environment variable).

## Supported Resource Types

KubeCloudScaler can manage various types of Kubernetes workload resources. By default, it targets all **deployments**.

### Available Resource Types

| Type | Description |
|------|-------------|
| `deployments` | Standard Kubernetes Deployments (default) |
| `statefulsets` | StatefulSets for stateful applications |
| `cronjobs` | Scheduled job resources |
| `hpas` | Horizontal Pod Autoscalers |
| `github-ars` | GitHub AutoScalingRunnerSets |

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

Control which namespaces are included or excluded from scaling operations via the `config` section.

### Default Behavior

By default, all namespaces are included except system namespaces (`kube-system`, `kube-public`, `kube-node-lease`) when `forceExcludeSystemNamespaces` is `true` (the default).

### Configuration Options

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `config.namespaces` | `[]string` | all | Specific namespaces to include (when set, only these are targeted) |
| `config.excludeNamespaces` | `[]string` | none | Namespaces to exclude. Ignored if `namespaces` is set |
| `config.forceExcludeSystemNamespaces` | `bool` | `true` | Always exclude system namespaces |

**Example -- Target Specific Namespaces**:
```yaml
spec:
  resources:
    types:
      - deployments
  config:
    namespaces:
      - development
      - staging
```

**Example -- Exclude Specific Namespaces**:
```yaml
spec:
  resources:
    types:
      - deployments
  config:
    excludeNamespaces:
      - production
      - critical-services
    forceExcludeSystemNamespaces: true
```

## Configuration Options

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `periods` | `[]ScalerPeriod` | Time periods defining when to scale resources |
| `resources` | `Resources` | Resource types and filters to target |

### Optional Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `dryRun` | `bool` | `false` | Preview actions without executing them |
| `config.namespaces` | `[]string` | all | Specific namespaces to target |
| `config.excludeNamespaces` | `[]string` | none | Namespaces to exclude |
| `config.forceExcludeSystemNamespaces` | `bool` | `true` | Always exclude system namespaces |
| `config.restoreOnDelete` | `bool` | `true` | Restore resources to original state when scaler is deleted |
| `config.disableEvents` | `bool` | `false` | Disable Kubernetes event generation |
| `config.deploymentTimeAnnotation` | `string` | none | Custom annotation for tracking deployment time |
| `config.authSecret` | `string` | none | Name of Kubernetes secret for remote cluster authentication |

## Integration with ArgoCD

When using [Argo CD](https://argo-cd.readthedocs.io/en/stable/user-guide/diffing/) for GitOps workflows, you may encounter out-of-sync issues due to KubeCloudScaler's resource modifications. To resolve this, configure Argo CD to ignore differences in `managedFields`:

```yaml
resource.customizations.ignoreDifferences.all: |
  managedFieldsManagers:
    - kubecloudscaler
```

## Complete Configuration Examples

### Example 1: Scale Down Development Environment After Hours

Scale down all development deployments outside business hours:

```yaml
apiVersion: kubecloudscaler.cloud/v1alpha3
kind: K8s
metadata:
  name: dev-environment-scaler
spec:
  resources:
    types:
      - deployments
      - statefulsets
  config:
    restoreOnDelete: true
    namespaces:
      - development
  periods:
    - type: "down"
      name: "outside-business-hours"
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
apiVersion: kubecloudscaler.cloud/v1alpha3
kind: K8s
metadata:
  name: test-weekend-shutdown
spec:
  dryRun: false
  resources:
    types:
      - deployments
    labelSelector:
      matchLabels:
        auto-scale: "enabled"
  config:
    restoreOnDelete: true
    namespaces:
      - test
      - qa
    excludeNamespaces:
      - test-production
  periods:
    - type: "down"
      name: "friday-evening"
      minReplicas: 0
      time:
        recurring:
          days:
            - friday
          startTime: "18:00"
          endTime: "23:59"
          timezone: "UTC"
          gracePeriod: "120s"
    - type: "down"
      name: "weekend"
      minReplicas: 0
      time:
        recurring:
          days:
            - saturday
            - sunday
          startTime: "00:00"
          endTime: "23:59"
          timezone: "UTC"
    - type: "up"
      name: "monday-start"
      time:
        recurring:
          days:
            - monday
          startTime: "07:00"
          endTime: "07:05"
          timezone: "UTC"
          once: true
```

### Example 3: Holiday Shutdown

Scale down for a specific holiday period:

```yaml
apiVersion: kubecloudscaler.cloud/v1alpha3
kind: K8s
metadata:
  name: christmas-holiday-scaler
spec:
  resources:
    types:
      - deployments
      - statefulsets
      - cronjobs
  config:
    restoreOnDelete: true
    namespaces:
      - development
      - staging
  periods:
    - type: "down"
      name: "christmas"
      minReplicas: 0
      time:
        fixed:
          startTime: "2026-12-24 18:00:00"
          endTime: "2027-01-02 08:00:00"
          timezone: "America/Los_Angeles"
          gracePeriod: "300s"
```

### Example 4: Multi-Period Complex Schedule

Combine multiple periods for complex scheduling:

```yaml
apiVersion: kubecloudscaler.cloud/v1alpha3
kind: K8s
metadata:
  name: complex-schedule-scaler
spec:
  resources:
    types:
      - deployments
    names:
      - api-service
      - worker-queue
      - batch-processor
  config:
    restoreOnDelete: true
    disableEvents: false
  periods:
    - type: "down"
      name: "night"
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
    - type: "down"
      name: "lunch"
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
    - type: "down"
      name: "weekend"
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
    name: "night"
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
    success:
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
5. **Enable RestoreOnDelete**: Keep `config.restoreOnDelete: true` (default) to avoid leaving resources in unexpected states
6. **Namespace Isolation**: Use specific namespaces to avoid accidentally scaling production resources
7. **Monitor Status**: Regularly check the status field for failed operations and adjust configuration
8. **Test Period Logic**: Verify your time periods work as expected, especially when using `reverse` mode
9. **Consider Time Zones**: Always specify the correct timezone for your schedule
10. **Resource Exclusions**: Use `config.excludeNamespaces` or label selectors to protect critical resources

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

- Configure Argo CD to ignore `managedFields` as shown above
- Consider using Argo CD's `ignoreDifferences` for replica counts
