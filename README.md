![Release Status](https://github.com/kubecloudscaler/kubecloudscaler/actions/workflows/release.yml/badge.svg) ![Documentation Status](https://github.com/kubecloudscaler/kubecloudscaler/actions/workflows/doc.yml/badge.svg) [![Apache 2.0 License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://choosealicense.com/licenses/apache-2.0/) [![GoLang](https://img.shields.io/badge/1.22.0-blue.svg?logo=go)]()

# Cloudscaler

**Cloudscaler** is a Kubernetes operator that scales cloud resources up or down using custom CRDs. It supports Kubernetes resources like [deployments](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/) and [cron jobs](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/) and aims to extend support to cloud resources like [Compute Engine](https://cloud.google.com/compute/docs/instances) and [Cloud SQL](https://cloud.google.com/sql/docs) on GCP.

This project is inspired by [kube-downscaler](https://codeberg.org/hjacobs/kube-downscaler).

## Getting Started

### Installation via Helm

1. **Install the Chart**

    ```bash
    helm install cloudscaler oci://ghcr.io/kubecloudscaler/kubecloudscaler/kubecloudscaler --namespace cloudscaler-system
    ```

2. **Create a Scaler Custom Resource (CR)**

    ```yaml
    # Example: Downscales all deployments (excluding kube-system) to 0 from 19:00 to 21:00 (Paris time) daily.
    apiVersion: kubecloudscaler/v1alpha1
    kind: K8s
    metadata:
      name: k8s-sample
    spec:
      periods:
        - time:
            recurring: true
            days:
              - all
            startTime: "19:00"
            endTime: "21:00"
            timezone: "Europe/Paris"
          minReplicas: 0
          maxReplicas: 10
          type: "down"
    ```

3. **Apply the Configuration**

    ```bash
    kubectl apply -f <scaler-CR-file.yaml>
    ```

## Documentation

Full documentation is available [here](https://kubecloudscaler.cloud).

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](http://www.apache.org/licenses/LICENSE-2.0) for details.
