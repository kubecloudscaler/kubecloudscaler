---
title: 'Install'
weight: 1
---

{{< tabs items="Manual,Helm,Kustomize" >}}

  {{< tab >}}
{{% steps %}}
### Build the Docker Image

  ```shell
  make docker-build docker-push IMG=ghcr.io/kubecloudscaler/kubecloudscaler:latest
  ```

### Install CRDs

  ```shell
  make install
  ```

### Deploy the Operator

  ```shell
  make deploy IMG=ghcr.io/kubecloudscaler/kubecloudscaler:latest
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
  helm install kubecloudscaler oci://ghcr.io/kubecloudscaler/kubecloudscaler/kubecloudscaler --namespace kubecloudscaler-system
  ```

### Create a Scaler Custom Resource (CR)

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
  git clone https://github.com/kubecloudscaler/kubecloudscaler.git
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
