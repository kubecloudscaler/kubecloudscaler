---
title: 'Usage'
---

## Overview

KubeCloudScaler allows you to automatically scale your Kubernetes and cloud resources based on time-based rules called **periods**. This guide explains how to configure and use the three custom resources provided by KubeCloudScaler.

## Custom Resources

All KubeCloudScaler CRDs are **cluster-scoped** (no namespace required) and use the `kubecloudscaler.cloud/v1alpha3` API version.

| CRD | Purpose | Use case |
|-----|---------|----------|
| [**K8s**](resources/k8s) | Scale Kubernetes workloads | Scale Deployments, StatefulSets, CronJobs, HPAs, GitHub ARS based on time |
| [**Gcp**](resources/gcp) | Scale GCP resources | Start/stop Compute Engine VM instances based on time |
| [**Flow**](resources/flow) | Orchestrate multi-resource scaling | Coordinate scaling across multiple K8s and Gcp resources with timing delays |

## Common Structure

All scalers share a common structure with **periods** defining when to scale and **resources** defining what to scale:

```yaml
apiVersion: kubecloudscaler.cloud/v1alpha3
kind: K8s  # or Gcp, Flow
metadata:
  name: my-scaler
spec:
  periods:
    - type: "down"          # "down" or "up"
      name: "night-scale"   # optional period name
      time:
        recurring:           # or fixed
          days: [all]
          startTime: "20:00"
          endTime: "07:00"
          timezone: "Europe/Paris"
      minReplicas: 0
  resources:
    types:
      - deployments
  config:
    # Resource-specific configuration
```

## Key Concepts

- **[Periods](period)**: Define when resources should be scaled (recurring or fixed time windows)
- **Resources**: Define what types of resources to target (by type, name, or label selector)
- **Config**: Resource-specific configuration (namespaces, authentication, etc.)
- **Dry-run mode**: Preview scaling actions without executing them (`dryRun: true`)
- **Restore on delete**: Automatically restore resources to their original state when the scaler CR is deleted
