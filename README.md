![Release Status](https://github.com/kubecloudscaler/kubecloudscaler/actions/workflows/release.yml/badge.svg) [![Apache 2.0 License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://choosealicense.com/licenses/apache-2.0/) [![Go](https://img.shields.io/badge/Go-1.25-blue.svg?logo=go)](https://go.dev/)

# KubeCloudScaler

**KubeCloudScaler** is a Kubernetes operator that automates time-based scaling of cloud and cluster resources using custom CRDs. Define when your workloads should scale up or down, and the operator handles the rest.

Inspired by [kube-downscaler](https://codeberg.org/hjacobs/kube-downscaler).

## Features

- **Kubernetes resources**: Deployments, StatefulSets, CronJobs, HPAs, GitHub AutoScaling Runner Sets
- **GCP resources**: Compute Engine VM instances
- **Flow orchestration**: coordinate scaling across multiple K8s and GCP resources with time delays
- **Recurring and fixed periods**: schedule by day/time or specific date ranges
- **Timezone support**: per-period timezone configuration
- **Dry-run mode**: validate scaling behavior without applying changes
- **Namespace filtering**: target or exclude specific namespaces
- **Label selectors**: fine-grained resource targeting
- **Restore on delete**: revert resource state when the CR is removed
- **Webhook validation**: catch misconfigurations before they are applied
- **Multi-version CRDs**: v1alpha1, v1alpha2, and v1alpha3 (storage version)

## Getting Started

### Installation via Helm

```bash
helm install kubecloudscaler oci://ghcr.io/kubecloudscaler/kubecloudscaler/kubecloudscaler \
  --namespace kubecloudscaler-system --create-namespace
```

### Examples

#### Scale Kubernetes deployments down every night

```yaml
apiVersion: kubecloudscaler.cloud/v1alpha3
kind: K8s
metadata:
  name: nightly-downscale
spec:
  periods:
    - type: down
      time:
        recurring:
          days: [all]
          startTime: "19:00"
          endTime: "07:00"
          timezone: "Europe/Paris"
      minReplicas: 0
  resources:
    types: [deployments]
  config:
    forceExcludeSystemNamespaces: true
```

#### Stop GCP VMs on weekends

```yaml
apiVersion: kubecloudscaler.cloud/v1alpha3
kind: Gcp
metadata:
  name: weekend-stop
spec:
  periods:
    - type: down
      time:
        recurring:
          days: [sat, sun]
          startTime: "00:00"
          endTime: "23:59"
          timezone: "Europe/Paris"
  resources:
    types: [vm-instances]
    names: [my-vm-1, my-vm-2]
  config:
    projectId: my-gcp-project
    region: europe-west1
```

#### Orchestrate scaling across K8s and GCP with Flow

```yaml
apiVersion: kubecloudscaler.cloud/v1alpha3
kind: Flow
metadata:
  name: full-env-downscale
spec:
  periods:
    - type: down
      name: nightly
      time:
        recurring:
          days: [mon, tue, wed, thu, fri]
          startTime: "20:00"
          endTime: "07:00"
          timezone: "Europe/Paris"
  resources:
    k8s:
      - name: apps
        resources:
          types: [deployments]
        config:
          namespaces: [staging]
    gcp:
      - name: vms
        resources:
          types: [vm-instances]
        config:
          projectId: my-gcp-project
          region: europe-west1
  flows:
    - periodName: nightly
      resources:
        - name: apps
        - name: vms
          startTimeDelay: "5m"
```

### Apply a resource

```bash
kubectl apply -f <scaler-resource.yaml>
```

## CRD Reference

| Kind | Scope | Description |
|------|-------|-------------|
| `K8s` | Cluster | Scales Kubernetes workloads |
| `Gcp` | Cluster | Scales GCP resources |
| `Flow` | Cluster | Orchestrates scaling across K8s and GCP resources |

### Supported resource types

| Kind | Resource types |
|------|---------------|
| K8s | `deployments`, `statefulsets`, `cronjobs`, `hpa`, `github-ars` |
| Gcp | `vm-instances` |

### Period configuration

Periods support two modes:

- **Recurring**: repeat on specific days and times (`days`, `startTime`, `endTime`, `timezone`)
- **Fixed**: one-time window with explicit datetime (`startTime`, `endTime` in `YYYY-MM-DD HH:MM:SS` format)

Additional options: `once` (trigger once at start), `reverse` (invert the period), `gracePeriod` (delay before scaling).

## Documentation

Full documentation is available at [kubecloudscaler.cloud](https://kubecloudscaler.cloud).

## Development

```bash
make build          # Build binary
make test           # Run unit tests
make lint           # Run golangci-lint
make manifests      # Regenerate CRDs, RBAC, webhooks
make generate       # Regenerate DeepCopy code
make run            # Run controller locally
```

## License

Licensed under the [Apache License 2.0](http://www.apache.org/licenses/LICENSE-2.0).
