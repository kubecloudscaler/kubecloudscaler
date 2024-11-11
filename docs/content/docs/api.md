Packages:

- [cloudscaler.io/v1alpha1](#cloudscaler.io%2fv1alpha1)

## cloudscaler.io/v1alpha1

Package v1alpha1 contains API Schema definitions for the k8s v1alpha1 API group

Resource Types:

### FixedPeriod

( _Appears on:_ [TimePeriod](#cloudscaler.io/v1alpha1.TimePeriod))

| Field | Description |
| --- | --- |
| `startTime`<br>_string_ |  |
| `endTime`<br>_string_ |  |
| `timezone`<br>_string_ |  |
| `once`<br>_bool_ | Run once at StartTime |
| `gracePeriod`<br>_time.Duration_ | Grace period in seconds for deployments before scaling down |
| `reverse`<br>_bool_ | Reverse the period |

### Gcp

Gcp is the Schema for the scalers API

| Field | Description |
| --- | --- |
| `metadata`<br>_[Kubernetes meta/v1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#objectmeta-v1-meta)_ | Refer to the Kubernetes API documentation for the fields of the<br>`metadata` field. |
| `spec`<br>_[GcpSpec](#cloudscaler.io/v1alpha1.GcpSpec)_ | |     |     |
| --- | --- |
| `dryRun`<br>_bool_ | dry-run mode |
| `periods`<br>_[\[\]ScalerPeriod](#cloudscaler.io/v1alpha1.ScalerPeriod)_ | Time period to scale | |
| `status`<br>_[ScalerStatus](#cloudscaler.io/v1alpha1.ScalerStatus)_ |  |

### GcpSpec

( _Appears on:_ [Gcp](#cloudscaler.io/v1alpha1.Gcp))

GcpSpec defines the desired state of Scaler

| Field | Description |
| --- | --- |
| `dryRun`<br>_bool_ | dry-run mode |
| `periods`<br>_[\[\]ScalerPeriod](#cloudscaler.io/v1alpha1.ScalerPeriod)_ | Time period to scale |

### K8s

Scaler is the Schema for the scalers API

| Field | Description |
| --- | --- |
| `metadata`<br>_[Kubernetes meta/v1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#objectmeta-v1-meta)_ | Refer to the Kubernetes API documentation for the fields of the<br>`metadata` field. |
| `spec`<br>_[K8sSpec](#cloudscaler.io/v1alpha1.K8sSpec)_ | |     |     |
| --- | --- |
| `dryRun`<br>_bool_ | dry-run mode |
| `periods`<br>_[\[\]ScalerPeriod](#cloudscaler.io/v1alpha1.ScalerPeriod)_ | Time period to scale |
| `namespaces`<br>_\[\]string_ | Resources<br>Namespaces |
| `excludeNamespaces`<br>_\[\]string_ | Exclude namespaces from downscaling |
| `resources`<br>_\[\]string_ | Resources |
| `excludeResources`<br>_\[\]string_ | Exclude resources from downscaling |
| `labelSelector`<br>_[Kubernetes meta/v1.LabelSelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#labelselector-v1-meta)_ | Labels selectors |
| `deploymentTimeAnnotation`<br>_string_ | Deployment time annotation |
| `disableEvents`<br>_bool_ | Disable events | |
| `status`<br>_[ScalerStatus](#cloudscaler.io/v1alpha1.ScalerStatus)_ |  |

### K8sSpec

( _Appears on:_ [K8s](#cloudscaler.io/v1alpha1.K8s))

ScalerSpec defines the desired state of Scaler

| Field | Description |
| --- | --- |
| `dryRun`<br>_bool_ | dry-run mode |
| `periods`<br>_[\[\]ScalerPeriod](#cloudscaler.io/v1alpha1.ScalerPeriod)_ | Time period to scale |
| `namespaces`<br>_\[\]string_ | Resources<br>Namespaces |
| `excludeNamespaces`<br>_\[\]string_ | Exclude namespaces from downscaling |
| `resources`<br>_\[\]string_ | Resources |
| `excludeResources`<br>_\[\]string_ | Exclude resources from downscaling |
| `labelSelector`<br>_[Kubernetes meta/v1.LabelSelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#labelselector-v1-meta)_ | Labels selectors |
| `deploymentTimeAnnotation`<br>_string_ | Deployment time annotation |
| `disableEvents`<br>_bool_ | Disable events |

### RecurringPeriod

( _Appears on:_ [ScalerStatusPeriod](#cloudscaler.io/v1alpha1.ScalerStatusPeriod), [TimePeriod](#cloudscaler.io/v1alpha1.TimePeriod))

| Field | Description |
| --- | --- |
| `days`<br>_\[\]string_ |  |
| `startTime`<br>_string_ |  |
| `endTime`<br>_string_ |  |
| `timezone`<br>_string_ |  |
| `once`<br>_bool_ | Run once at StartTime |
| `gracePeriod`<br>_time.Duration_ | Grace period in seconds for deployments before scaling down |
| `reverse`<br>_bool_ | Reverse the period |

### ScalerPeriod

( _Appears on:_ [GcpSpec](#cloudscaler.io/v1alpha1.GcpSpec), [K8sSpec](#cloudscaler.io/v1alpha1.K8sSpec))

| Field | Description |
| --- | --- |
| `type`<br>_string_ |  |
| `time`<br>_[TimePeriod](#cloudscaler.io/v1alpha1.TimePeriod)_ |  |
| `minReplicas`<br>_int32_ | Minimum replicas |
| `maxReplicas`<br>_int32_ | Maximum replicas |

### ScalerStatus

( _Appears on:_ [Gcp](#cloudscaler.io/v1alpha1.Gcp), [K8s](#cloudscaler.io/v1alpha1.K8s))

ScalerStatus defines the observed state of Scaler

| Field | Description |
| --- | --- |
| `currentPeriod`<br>_[ScalerStatusPeriod](#cloudscaler.io/v1alpha1.ScalerStatusPeriod)_ |  |
| `comments`<br>_string_ |  |

### ScalerStatusFailed

( _Appears on:_ [ScalerStatusPeriod](#cloudscaler.io/v1alpha1.ScalerStatusPeriod))

| Field | Description |
| --- | --- |
| `kind`<br>_string_ |  |
| `name`<br>_string_ |  |
| `reason`<br>_string_ |  |

### ScalerStatusPeriod

( _Appears on:_ [ScalerStatus](#cloudscaler.io/v1alpha1.ScalerStatus))

| Field | Description |
| --- | --- |
| `spec`<br>_[RecurringPeriod](#cloudscaler.io/v1alpha1.RecurringPeriod)_ | |
| |
| `specSHA`<br>_string_ |  |
| `success`<br>_[\[\]ScalerStatusSuccess](#cloudscaler.io/v1alpha1.ScalerStatusSuccess)_ |  |
| `failed`<br>_[\[\]ScalerStatusFailed](#cloudscaler.io/v1alpha1.ScalerStatusFailed)_ |  |

### ScalerStatusSuccess

( _Appears on:_ [ScalerStatusPeriod](#cloudscaler.io/v1alpha1.ScalerStatusPeriod))

| Field | Description |
| --- | --- |
| `kind`<br>_string_ |  |
| `name`<br>_string_ |  |

### TimePeriod

( _Appears on:_ [ScalerPeriod](#cloudscaler.io/v1alpha1.ScalerPeriod))

| Field | Description |
| --- | --- |
| `recurring`<br>_[RecurringPeriod](#cloudscaler.io/v1alpha1.RecurringPeriod)_ |  |
| `fixed`<br>_[FixedPeriod](#cloudscaler.io/v1alpha1.FixedPeriod)_ |  |

* * *

_Generated with `gen-crd-api-reference-docs`_
_on git commit `5e4f4f4`._

