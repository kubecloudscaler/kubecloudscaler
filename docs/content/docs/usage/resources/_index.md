---
title: 'Resources'
---

## Overview

KubeCloudScaler provides three custom resource types to manage scaling. Each targets different infrastructure: Kubernetes workloads, GCP cloud resources, or orchestrated multi-resource workflows.

All resources are **cluster-scoped** and share a common [period](../period) configuration for time-based rules.

## Resource Selection

K8s and Gcp resources use the same `resources` field to define what to scale:

```yaml
resources:
  types:           # Resource types to target
    - deployments
  names:           # Optional: specific resource names
    - my-app
  labelSelector:   # Optional: Kubernetes label selector
    matchLabels:
      env: dev
```

- **types**: The kind of resources to manage (e.g., `deployments`, `statefulsets`, `vm-instances`)
- **names**: Target specific resources by name (optional, targets all if omitted)
- **labelSelector**: Filter resources using standard Kubernetes [label selectors](https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/label-selector/#LabelSelector) (optional)
