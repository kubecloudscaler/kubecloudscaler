---
title: Usage
weight: 2
---

## Overview

KubeCloudScaler allows you to automatically scale your Kubernetes resources based on time-based rules called **periods**. This guide explains how to configure and use these periods effectively.

## Understanding Periods

Periods define when and how your resources should be scaled. Each scaler definition can contain multiple period definitions that control scaling behavior based on time patterns.

### Key Concepts

- **Sequential Evaluation**: Periods are evaluated in order, with the first matching period taking precedence
- **Reverse Mode**: Use the `reverse` field to invert period logic - making it inactive during the specified time range and active outside of it
- **One-time Scaling**: Set `once: true` to apply scaling only when entering or leaving a time range, preventing interference with manual scaling
- **Inclusive End Time**: The `endTime` is inclusive, meaning a period remains active until the last second before the specified end time (e.g., `endTime: "00:00"` stays active until `23:59:59`)

> [!NOTE]
> When `once` is enabled, KubeCloudScaler will only scale resources when transitioning into or out of the specified time range. Manual scaling operations will not be overridden.

## Period Types

### Recurring Periods

Recurring periods repeat on a daily basis according to specified days and times.

**Time Format**: `HH:MM` (24-hour format)

**Example**: `"08:30"` represents 8:30 AM

### Fixed Periods

Fixed periods occur at specific dates and times, useful for one-time events or maintenance windows.

**Time Format**: `YYYY-MM-DD HH:MM:SS`

**Example**: `"2024-12-25 09:00:00"` represents December 25th, 2024 at 9:00 AM

## Configuration Examples

{{< tabs items="Basic Scaling,Multiple Periods,Scheduled Maintenance" >}}

  {{< tab >}}
**Scenario**: Scale down resources during off-hours

```yaml
periods:
  - time:
      recurring:
        days:
          - all
        startTime: "01:00"
        endTime: "22:50"
        timezone: "Europe/Paris"
    minReplicas: 0
    maxReplicas: 10
    type: "down"
```
> [!NOTE]
> Resources are scaled down to 0 replicas daily from 1:00 AM to 10:50 PM (Paris time).

  {{< /tab >}}

  {{< tab >}}
**Scenario**: Different scaling rules for different times of day

```yaml
periods:
  - time:
      recurring:
        days:
          - all
        startTime: "01:00"
        endTime: "07:00"
        timezone: "Europe/Paris"
    minReplicas: 0
    maxReplicas: 10
    type: "down"
  - time:
      recurring:
        days:
          - all
        startTime: "12:00"
        endTime: "20:00"
        timezone: "Europe/Paris"
    minReplicas: 0
    maxReplicas: 10
    type: "up"
```
> [!NOTE]
> Resources are scaled down to 0 replicas from 1:00-7:00 AM and scaled up to 10 replicas from 12:00-8:00 PM (Paris time).

  {{< /tab >}}

  {{< tab >}}
**Scenario**: Planned maintenance window

```yaml
periods:
  - time:
      fixed:
        startTime: "2024-11-15 20:00:00"
        endTime: "2024-11-17 08:00:00"
        timezone: "Europe/Paris"
    minReplicas: 0
    maxReplicas: 10
    type: "down"
```
> [!NOTE]
> Resources are scaled down to 0 replicas during a specific maintenance window from November 15th 8:00 PM to November 17th 8:00 AM (Paris time).

  {{< /tab >}}
{{< /tabs >}}

## Resource Management

### Kubernetes cluster authentication

KubeCloudScaler can manage resources in both local and remote Kubernetes clusters. There are three authentication methods available:

#### 1. InCluster (Default)
- **What it does**: KubeCloudScaler manages resources only in the local cluster where it's deployed
- **When to use**: Default mode for single-cluster deployments
- **Configuration**: No additional setup required

**Example**: This is the simplest setup - just deploy KubeCloudScaler and it will automatically detect and use the current cluster's service account.

#### 2. KUBECONFIG Environment Variable
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
type: Opaque
data:
  kubeconfig: <base64-encoded-kubeconfig-content>
---
# Deploy KubeCloudScaler with the kubeconfig mounted
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kubecloudscaler
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
      volumes:
      - name: kubeconfig-volume
        secret:
          secretName: remote-cluster-kubeconfig
```

#### 3. Service Account Token (authSecret)
- **What it does**: Uses a service account token to authenticate with remote clusters
- **When to use**: When you need long-lived authentication to remote clusters
- **Configuration**:
  1. Create a secret containing:
     - The service account token (copy from the automatically generated secret in the remote cluster)
     - The CA of the remote cluster
     - An `insecure` field if the CA is not trusted
     - A `URL` field with the remote cluster's API server URL
  2. Reference this secret name in the `authSecret` field of your KubeCloudScaler resource

**Example**:
```yaml
# 1. Create a secret with the service account token and cluster URL
apiVersion: v1
kind: Secret
metadata:
  name: remote-cluster-auth
type: Opaque
data:
  token: <base64-encoded-service-account-token>
  ca.crt: <base-64-encoded-service-account-ca>
  URL: <base64-encoded-cluster-api-url>
  insecure: <true|false>
---
# 2. Reference the secret in your KubeCloudScaler resource
apiVersion: kubecloudscaler.example.com/v1alpha1
kind: KubeCloudScaler
metadata:
  name: example-scaler
spec:
  authSecret: remote-cluster-auth
  # ... other configuration
```

**Note**: For more details on creating long-lived tokens, see the [Kubernetes documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#manually-create-a-long-lived-api-token-for-a-serviceaccount)

### Supported Kubernetes Resources

KubeCloudScaler can manage various types of Kubernetes resources. By default, it targets all **deployments** across all namespaces (excluding `kube-system`).

#### Available Resource Types

- **deployments** - Standard Kubernetes Deployments
- **statefulsets** - StatefulSets for stateful applications
- **cronjobs** - Scheduled job resources
- **horizontalPodAutoscalers** - HPA resources for automatic scaling

> [!WARNING]
> Deployments and HorizontalPodAutoscalers cannot be managed simultaneously as they serve conflicting purposes.


### Namespace Selection

**Default Behavior**: All namespaces except `kube-system` are included

**Custom Configuration Options**:
- `namespaces`: Specify exact namespaces to include
- `excludeNamespaces`: Exclude specific namespaces from selection
- `forceExcludeSystemNamespaces`: Ensure system namespaces are always excluded

### Resource Filtering

Use Kubernetes [labelSelector](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/label-selector/#LabelSelector) to filter resources based on labels, allowing precise control over which resources are managed.

### Integration with ArgoCD

When using [Argo-CD](https://argo-cd.readthedocs.io/en/stable/user-guide/diffing/) for GitOps workflows, you may encounter out-of-sync issues due to KubeCloudScaler's resource modifications. To resolve this, configure ArgoCD to ignore differences in `managedFields`:

```yaml
resource.customizations.ignoreDifferences.all: |
  managedFieldsManagers:
    - kubecloudscaler
```

## Complete Configuration Example

This example demonstrates a comprehensive scaler configuration with multiple periods and resource targeting:

```yaml
spec:
  restoreOnDelete: true
  resources:
    - deployments
    - statefulsets
  namespaces:
    - default
  excludeNamespaces: []
  periods:
    # Scale down during lunch break
    - time:
        recurring:
          days:
            - all
          startTime: "12:00"
          endTime: "14:00"
          timezone: "Europe/Paris"
          once: false
          reverse: false
          gracePeriod: 5s
      minReplicas: 0
      type: "down"
    # Keep resources down during night hours (using reverse mode)
    - time:
        recurring:
          days:
            - all
          startTime: "07:00"
          endTime: "23:00"
          timezone: "Europe/Paris"
          once: false
          reverse: true
          gracePeriod: 5s
      minReplicas: 0
      type: "down"
  labelSelector:
    matchLabels:
      app.kubernetes.io/name: my-preferred-app
```
