---
title: ""
layout: index
---

# Welcome to KubeCloudScaler

KubeCloudScaler is a Kubernetes operator for **automated, time-based scaling** of your cloud and cluster resources. Whether you want to save costs at night, boost performance during the day, or schedule maintenance windows, KubeCloudScaler has you covered.

## What can it do?

- **Scale up** or **scale down** your resources automatically based on time periods
- Manage **Kubernetes workloads**: [Deployments](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/), [StatefulSets](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/), [CronJobs](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/), HPAs, [GitHub AutoScalingRunnerSets](https://docs.github.com/en/actions/hosting-your-own-runners/managing-self-hosted-runners-with-actions-runner-controller/deploying-runner-scale-sets-with-actions-runner-controller)
- Manage **GCP resources**: [Compute Engine VM instances](https://cloud.google.com/compute/docs/instances)
- **Orchestrate** complex multi-resource scaling workflows with Flows

> Inspired by [kube-downscaler](https://codeberg.org/hjacobs/kube-downscaler)

## Three Custom Resources

| CRD | Purpose |
|-----|---------|
| **K8s** | Scale Kubernetes workloads (Deployments, StatefulSets, CronJobs, HPAs) |
| **Gcp** | Scale GCP resources (Compute Engine VM instances) |
| **Flow** | Orchestrate scaling across multiple K8s and Gcp resources with timing delays |

## Get Started

Ready to optimize your cloud? Dive into the docs to learn how to install, configure, and get the most out of KubeCloudScaler.

{{< cards >}}
  {{< card link="docs" title="Documentation" icon="book-open" >}}
{{< /cards >}}
