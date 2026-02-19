---
title: Flow
weight: 3
---

## Overview

The Flow resource type allows you to orchestrate scaling across multiple K8s and Gcp resources with timing delays. Instead of managing individual scalers independently, a Flow groups them together and coordinates their scaling with optional start and end time offsets per resource.

This is useful for scenarios where resources need to be scaled in a specific order, such as shutting down frontend services before backend databases, or starting databases before application servers.

## Spec Structure

```yaml
apiVersion: kubecloudscaler.cloud/v1alpha3
kind: Flow
metadata:
  name: my-flow              # Cluster-scoped, no namespace needed
spec:
  periods:                    # Required: time-based scaling rules (with names)
    - type: "down"
      name: "night"           # Name is used to reference periods in flows
      time: { ... }
  resources:                  # Required: define the K8s and Gcp resources
    k8s:
      - name: "my-k8s-resource"
        resources: { ... }
        config: { ... }
    gcp:
      - name: "my-gcp-resource"
        resources: { ... }
        config: { ... }
  flows:                      # Optional: define per-period resource orchestration
    - periodName: "night"
      resources:
        - name: "my-k8s-resource"
          startTimeDelay: "0m"
          endTimeDelay: "0m"
```

## How Flows Work

1. **Periods** define the time windows (same as K8s and Gcp resources)
2. **Resources** define the K8s and Gcp resources to manage, each with their own configuration
3. **Flows** map periods to resources with optional timing delays

When a period becomes active, KubeCloudScaler creates the corresponding K8s and/or Gcp scaler resources automatically. The `startTimeDelay` and `endTimeDelay` fields allow staggering the scaling of different resources.

### Timing Delays

- **`startTimeDelay`**: Delays the start of the period for this specific resource. For example, `"10m"` means this resource starts scaling 10 minutes after the period begins.
- **`endTimeDelay`**: Delays the end of the period for this specific resource. For example, `"5m"` means this resource stops scaling 5 minutes after the period ends.

Both fields accept a duration in minutes (e.g., `"0m"`, `"5m"`, `"30m"`).

## Resource Definitions

### K8s Resources in a Flow

Each K8s resource entry defines a named Kubernetes scaler with its own resource selection and configuration:

```yaml
resources:
  k8s:
    - name: "frontend-apps"        # Unique name for this resource group
      resources:
        types:
          - deployments
        labelSelector:
          matchLabels:
            tier: frontend
      config:
        namespaces:
          - production
        restoreOnDelete: true
```

The `resources` and `config` fields follow the same structure as a standalone [K8s resource](k8s).

### Gcp Resources in a Flow

Each Gcp resource entry defines a named GCP scaler with its own resource selection and configuration:

```yaml
resources:
  gcp:
    - name: "dev-vms"              # Unique name for this resource group
      resources:
        types:
          - vm-instances
        labelSelector:
          matchLabels:
            environment: development
      config:
        projectId: my-project
        region: us-central1
        restoreOnDelete: true
```

The `resources` and `config` fields follow the same structure as a standalone [Gcp resource](gcp).

## Complete Configuration Examples

### Example 1: Coordinated Dev Environment Shutdown

Shut down frontend services first, then backend, then stop GCP VMs:

```yaml
apiVersion: kubecloudscaler.cloud/v1alpha3
kind: Flow
metadata:
  name: dev-environment-flow
spec:
  periods:
    - type: "down"
      name: "night-shutdown"
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
  resources:
    k8s:
      - name: "frontend"
        resources:
          types:
            - deployments
          labelSelector:
            matchLabels:
              tier: frontend
        config:
          namespaces:
            - development
          restoreOnDelete: true
      - name: "backend"
        resources:
          types:
            - deployments
            - statefulsets
          labelSelector:
            matchLabels:
              tier: backend
        config:
          namespaces:
            - development
          restoreOnDelete: true
    gcp:
      - name: "dev-vms"
        resources:
          types:
            - vm-instances
          labelSelector:
            matchLabels:
              env: dev
        config:
          projectId: my-dev-project
          region: europe-west1
          restoreOnDelete: true
  flows:
    - periodName: "night-shutdown"
      resources:
        - name: "frontend"
          startTimeDelay: "0m"      # Frontend scales down immediately
          endTimeDelay: "10m"       # Frontend scales up 10min after period ends
        - name: "backend"
          startTimeDelay: "5m"      # Backend scales down 5min after frontend
          endTimeDelay: "5m"        # Backend scales up 5min after period ends
        - name: "dev-vms"
          startTimeDelay: "10m"     # VMs stop 10min after backend
          endTimeDelay: "0m"        # VMs start immediately when period ends
```

### Example 2: Weekend Shutdown with Staggered Start

Coordinate a weekend shutdown with staggered startup on Monday:

```yaml
apiVersion: kubecloudscaler.cloud/v1alpha3
kind: Flow
metadata:
  name: weekend-flow
spec:
  periods:
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
  resources:
    k8s:
      - name: "apps"
        resources:
          types:
            - deployments
        config:
          namespaces:
            - staging
          restoreOnDelete: true
    gcp:
      - name: "staging-vms"
        resources:
          types:
            - vm-instances
          labelSelector:
            matchLabels:
              env: staging
        config:
          projectId: my-project
          region: europe-west1
          restoreOnDelete: true
  flows:
    - periodName: "weekend"
      resources:
        - name: "staging-vms"
          startTimeDelay: "0m"
          endTimeDelay: "0m"        # VMs start first
        - name: "apps"
          startTimeDelay: "0m"
          endTimeDelay: "5m"        # Apps start 5min after VMs
```

## Status Monitoring

The Flow resource uses standard Kubernetes conditions to report its state:

```yaml
status:
  conditions:
    - type: Available
      status: "True"
      lastTransitionTime: "2026-02-19T10:00:00Z"
      reason: ReconcileSuccess
      message: "Flow resources are in sync"
    - type: Progressing
      status: "False"
      lastTransitionTime: "2026-02-19T10:00:00Z"
      reason: ReconcileComplete
      message: "All resources reconciled"
```

To check the status of individual K8s and Gcp resources created by the Flow, inspect the child resources directly:

```shell
# List all K8s scalers (includes those created by flows)
kubectl get k8s

# List all Gcp scalers (includes those created by flows)
kubectl get gcp
```

## Best Practices

1. **Name your periods**: Flow resources reference periods by name in the `flows` section, so always set the `name` field on periods
2. **Use timing delays wisely**: Start databases before applications, stop frontends before backends
3. **Test with dry-run**: Use `dryRun: true` on individual resource configs before deploying flows
4. **Monitor child resources**: The Flow creates K8s and Gcp resources -- monitor their status for errors
5. **Keep delays reasonable**: Large delays can cause resources to be out of sync for extended periods
6. **Use RestoreOnDelete**: Enable `restoreOnDelete` on all resource configs to ensure clean cleanup
