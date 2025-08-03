---
title: 'Installation'
weight: 1
---

## Overview

KubeCloudScaler can be installed using three different methods. Choose the one that best fits your deployment strategy:

- **Helm** (Recommended): Best for production deployments with easy configuration management
- **Kustomize**: Ideal for GitOps workflows and customized deployments
- **Manual**: For development, testing, or when you need full control over the build process

{{< tabs items="Helm,Kustomize,Manual" >}}

{{< tab >}}
**Recommended for production environments**

{{% steps %}}

### Install KubeCloudScaler

Install the operator using Helm with the official OCI registry:

```shell
helm upgrade --install kubecloudscaler \
  oci://ghcr.io/kubecloudscaler/charts/kubecloudscaler \
  --namespace kubecloudscaler-system \
  --create-namespace
```

### Create Your First Scaler Configuration

Create a YAML file (e.g., `my-scaler.yaml`) with your scaling configuration:

```yaml
# Example: Scale down all deployments in 'default' namespace during evening hours
apiVersion: kubecloudscaler.cloud/v1alpha1
kind: K8s
metadata:
  name: evening-scaledown
  namespace: kubecloudscaler-system
spec:
  periods:
    - time:
        recurring:
          days:
            - all
          startTime: "19:00"
          endTime: "21:00"
          timezone: "Europe/Paris"
      minReplicas: 0
      maxReplicas: 10
      type: "down"
  namespaces:
    - default
  forceExcludeSystemNamespaces: true
```

### Apply the Configuration

Deploy your scaler configuration:

```shell
kubectl apply -f my-scaler.yaml
```

### Verify Installation

Check that the operator is running:

```shell
kubectl get pods -n kubecloudscaler-system
kubectl get k8s -A
```

### Uninstall (Optional)

To remove KubeCloudScaler completely:

```shell
# Remove your scaler configurations first
kubectl delete k8s --all -A

# Uninstall the Helm chart
helm uninstall kubecloudscaler -n kubecloudscaler-system

# Remove the namespace if desired
kubectl delete namespace kubecloudscaler-system
```

{{% /steps %}}
  {{< /tab >}}

{{< tab >}}
**Ideal for GitOps workflows and infrastructure-as-code**

{{% steps %}}

### Clone the Repository

```shell
git clone https://github.com/kubecloudscaler/kubecloudscaler.git
cd kubecloudscaler
```

### Deploy Using Kustomize

Apply the default configuration directly from the repository:

```shell
kubectl apply -k config/default
```

This will:
- Install the Custom Resource Definitions (CRDs)
- Deploy the operator in the `kubecloudscaler-system` namespace
- Set up necessary RBAC permissions

### Customize Your Deployment (Optional)

Create your own kustomization file to customize the deployment:

```yaml
# kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- https://github.com/kubecloudscaler/kubecloudscaler/config/default

# Add your customizations here
patchesStrategicMerge:
- custom-config.yaml

namespace: my-custom-namespace
```

### Create Scaler Resources

Create your scaler configurations in separate YAML files and apply them:

```shell
kubectl apply -f my-scaler-configs/
```

### Verify Installation

```shell
kubectl get pods -n kubecloudscaler-system
kubectl get crd | grep kubecloudscaler
```

### Uninstall

Remove the deployment:

```shell
kubectl delete -k config/default
```

{{% /steps %}}
  {{< /tab >}}

  {{< tab >}}
**For development, testing, and custom builds**

{{% steps %}}

### Prerequisites

Ensure you have the following tools installed:
- Docker or Podman
- kubectl configured for your cluster
- make
- Go 1.21+ (for local development)

### Clone and Build

```shell
git clone https://github.com/kubecloudscaler/kubecloudscaler.git
cd kubecloudscaler
```

### Build and Push Container Image

Build and push the container image to your registry:

```shell
make docker-build docker-push IMG=<your-registry>/kubecloudscaler:latest
```

Replace `<your-registry>` with your container registry (e.g., `ghcr.io/yourusername`).

### Install Custom Resource Definitions

```shell
make install
```

### Deploy the Operator

Deploy the operator using your custom image:

```shell
make deploy IMG=<your-registry>/kubecloudscaler:latest
```

### Test with Sample Configurations

Apply the provided sample configurations:

```shell
kubectl apply -k config/samples/
```

### Verify Deployment

Check that everything is working:

```shell
kubectl get pods -n kubecloudscaler-system
kubectl get k8s -A
```

### Development Workflow

For active development, you can run the operator locally:

```shell
make run
```

### Uninstall

Clean up the installation:

```shell
# Remove sample configurations
kubectl delete -k config/samples/

# Remove the operator deployment
make undeploy

# Remove CRDs (this will delete all scaler resources!)
make uninstall
```

{{% /steps %}}
  {{< /tab >}}

{{< /tabs >}}

## Next Steps

After installation, you can:

1. **Configure your first scaler** - See the [Usage Guide](../usage) for detailed configuration examples
2. **Monitor scaling operations** - Check the operator logs and resource events
3. **Set up multiple scalers** - Create different scaling policies for different applications or environments

## Troubleshooting

### Common Issues

**Operator not starting**: Check the logs with `kubectl logs -n kubecloudscaler-system deployment/kubecloudscaler-controller-manager`

**Permissions issues**: Ensure your cluster has the necessary RBAC permissions for the operator

**CRD conflicts**: If upgrading, ensure old CRDs are properly updated or removed before installing new versions
