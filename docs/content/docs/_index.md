---
title: 'Documentation'
breadcrumbs: false
---

# Welcome to KubeCloudScaler Docs

KubeCloudScaler helps you automatically scale your Kubernetes and cloud workloads based on time periods. Save money by powering down unused resources outside business hours, and scale back up when you need them.

## Why KubeCloudScaler?

- **Time-based scaling**: Schedule scaling for nights, weekends, or maintenance windows
- **Cost optimization**: Cut cloud costs by stopping unused resources automatically
- **Multi-resource support**: Manage Kubernetes workloads (Deployments, StatefulSets, CronJobs, HPAs) and GCP resources (Compute Engine VMs)
- **Flow orchestration**: Coordinate scaling across multiple resources with timing delays
- **Declarative configuration**: Define scaling policies as Kubernetes Custom Resources

## Custom Resources

KubeCloudScaler provides three cluster-scoped CRDs:

| CRD | API Version | Purpose |
|-----|-------------|---------|
| **K8s** | `kubecloudscaler.cloud/v1alpha3` | Scale Kubernetes workloads |
| **Gcp** | `kubecloudscaler.cloud/v1alpha3` | Scale GCP Compute Engine VMs |
| **Flow** | `kubecloudscaler.cloud/v1alpha3` | Orchestrate multi-resource scaling workflows |

## What's Next?

- [Installation Guide](install) -- Get up and running in minutes
- [Usage Guide](usage) -- Understand periods, resources, and flows
- [API Reference](api) -- Full CRD API documentation
