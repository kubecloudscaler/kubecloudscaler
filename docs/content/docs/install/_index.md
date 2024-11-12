---
title: 'Install'
---

## Helm

1. **Install the Chart**

    ```bash
    helm install cloudscaler oci://ghcr.io/cloudscalerio/cloudscaler/cloudscaler --namespace cloudscaler-system
    ```

2. **Create a Scaler Custom Resource (CR)**

    ```yaml
    # Example: Downscales all deployments (excluding kube-system) to 0 from 19:00 to 21:00 (Paris time) daily.
    apiVersion: cloudscaler.io/v1alpha1
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
