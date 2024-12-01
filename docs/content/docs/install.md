---
title: 'Install'
weight: 1
---

{{< tabs items="Manual,Helm,Kustomize" >}}

  {{< tab >}}
{{% steps %}}
### Build the Docker Image

  ```shell
  make docker-build docker-push IMG=ghcr.io/k8scloudscaler/k8scloudscaler:latest
  ```

### Install CRDs

  ```shell
  make install
  ```

### Deploy the Operator

  ```shell
  make deploy IMG=ghcr.io/k8scloudscaler/k8scloudscaler:latest
  ```

### Apply Sample Configurations

  ```shell
  kubectl apply -k config/samples/
  ```

### (optional) Uninstall
  ```shell
  kubectl delete -k config/samples/
  make uninstall
  ```
{{% /steps %}}
  {{< /tab >}}

  {{< tab >}}
{{% steps %}}

### Install the Chart

  ```shell
  helm install k8scloudscaler oci://ghcr.io/k8scloudscaler/k8scloudscaler/k8scloudscaler --namespace k8scloudscaler-system
  ```

### Create a Scaler Custom Resource (CR)

  ```yaml
  # Example: Downscales all deployments (excluding kube-system) to 0 from 19:00 to 21:00 (Paris time) daily.
  apiVersion: k8scloudscaler/v1alpha1
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

### Apply the Configuration

  ```shell
  kubectl apply -f <scaler-CR-file.yaml>
  ```

{{% /steps %}}

  {{< /tab >}}
  {{< tab >}}
{{% steps %}}
### Clone the Repository

  ```shell
  git clone https://github.com/k8scloudscaler/k8scloudscaler.git
  cd cloudscaler
  ```

### Apply Kustomize Configuration

  In the repository root directory, run:
  ```shell
  kubectl apply -k config/default
  ```

### Uninstall with Kustomize

  ```shell
  kubectl delete -k config/default
  ```
{{% /steps %}}
  {{< /tab >}}

{{< /tabs >}}
