---
title: API Reference
---

**Packages**
- [kubecloudscaler.cloud/v1alpha1](#kubecloudscalercloudv1alpha1)


## kubecloudscaler.cloud/v1alpha1


Package v1alpha1 contains API Schema definitions for the k8s v1alpha1 API group


### Resource Types
- [Gcp](#gcp)
- [K8s](#k8s)



#### FixedPeriod







_Appears in:_
- [TimePeriod](#timeperiod)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `startTime` _string_ |  |  | Pattern: `^\d\{4\}-(0?[1-9]\|1[0,1,2])-(0?[1-9]\|[12][0-9]\|3[01]) ([0-1]?[0-9]\|2[0-3]):[0-5]?[0-9]:[0-5]?[0-9]$` <br /> |
| `endTime` _string_ |  |  | Pattern: `^\d\{4\}-(0?[1-9]\|1[0,1,2])-(0?[1-9]\|[12][0-9]\|3[01]) ([0-1]?[0-9]\|2[0-3]):[0-5]?[0-9]:[0-5]?[0-9]$` <br /> |
| `timezone` _string_ |  |  |  |
| `once` _boolean_ | Run once at StartTime |  |  |
| `gracePeriod` _string_ | Grace period in seconds for deployments before scaling down |  | Pattern: `^\d*s$` <br /> |
| `reverse` _boolean_ | Reverse the period |  |  |


#### Gcp



Gcp is the Schema for the scalers API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `kubecloudscaler.cloud/v1alpha1` | | |
| `kind` _string_ | `Gcp` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[GcpSpec](#gcpspec)_ |  |  |  |
| `status` _[ScalerStatus](#scalerstatus)_ |  |  |  |


#### GcpSpec



GcpSpec defines the desired state of Scaler



_Appears in:_
- [Gcp](#gcp)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `dryRun` _boolean_ | dry-run mode |  |  |
| `periods` _[ScalerPeriod](#scalerperiod) array_ | Time period to scale |  |  |


#### K8s



Scaler is the Schema for the scalers API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `kubecloudscaler.cloud/v1alpha1` | | |
| `kind` _string_ | `K8s` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[K8sSpec](#k8sspec)_ |  |  |  |
| `status` _[ScalerStatus](#scalerstatus)_ |  |  |  |


#### K8sSpec



ScalerSpec defines the desired state of Scaler



_Appears in:_
- [K8s](#k8s)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `dryRun` _boolean_ | dry-run mode |  |  |
| `periods` _[ScalerPeriod](#scalerperiod) array_ | Time period to scale |  |  |
| `namespaces` _string array_ | Resources<br />Namespaces |  |  |
| `excludeNamespaces` _string array_ | Exclude namespaces from downscaling |  |  |
| `forceExcludeSystemNamespaces` _boolean_ | Force exclude system namespaces |  |  |
| `resources` _string array_ | Resources |  |  |
| `excludeResources` _string array_ | Exclude resources from downscaling |  |  |
| `labelSelector` _[LabelSelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#labelselector-v1-meta)_ | Labels selectors |  |  |
| `deploymentTimeAnnotation` _string_ | Deployment time annotation |  |  |
| `disableEvents` _boolean_ | Disable events |  |  |
| `authSecret` _string_ | AuthSecret name |  |  |


#### RecurringPeriod







_Appears in:_
- [ScalerStatusPeriod](#scalerstatusperiod)
- [TimePeriod](#timeperiod)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `days` _string array_ |  |  |  |
| `startTime` _string_ |  |  | Pattern: `^([0-1]?[0-9]\|2[0-3]):[0-5][0-9]$` <br /> |
| `endTime` _string_ |  |  | Pattern: `^([0-1]?[0-9]\|2[0-3]):[0-5][0-9]$` <br /> |
| `timezone` _string_ |  |  |  |
| `once` _boolean_ | Run once at StartTime |  |  |
| `gracePeriod` _string_ |  |  | Pattern: `^\d*s$` <br /> |
| `reverse` _boolean_ | Reverse the period |  |  |


#### ScalerPeriod







_Appears in:_
- [GcpSpec](#gcpspec)
- [K8sSpec](#k8sspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ |  |  | Enum: [down up] <br /> |
| `time` _[TimePeriod](#timeperiod)_ |  |  |  |
| `minReplicas` _integer_ | Minimum replicas |  |  |
| `maxReplicas` _integer_ | Maximum replicas |  |  |


#### ScalerStatus



ScalerStatus defines the observed state of Scaler



_Appears in:_
- [Gcp](#gcp)
- [K8s](#k8s)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `currentPeriod` _[ScalerStatusPeriod](#scalerstatusperiod)_ |  |  |  |
| `comments` _string_ |  |  |  |


#### ScalerStatusFailed







_Appears in:_
- [ScalerStatusPeriod](#scalerstatusperiod)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `kind` _string_ |  |  |  |
| `name` _string_ |  |  |  |
| `reason` _string_ |  |  |  |


#### ScalerStatusPeriod







_Appears in:_
- [ScalerStatus](#scalerstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `spec` _[RecurringPeriod](#recurringperiod)_ |  |  |  |
| `specSHA` _string_ |  |  |  |
| `type` _string_ |  |  |  |
| `success` _[ScalerStatusSuccess](#scalerstatussuccess) array_ |  |  |  |
| `failed` _[ScalerStatusFailed](#scalerstatusfailed) array_ |  |  |  |


#### ScalerStatusSuccess







_Appears in:_
- [ScalerStatusPeriod](#scalerstatusperiod)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `kind` _string_ |  |  |  |
| `name` _string_ |  |  |  |


#### TimePeriod







_Appears in:_
- [ScalerPeriod](#scalerperiod)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `recurring` _[RecurringPeriod](#recurringperiod)_ |  |  |  |
| `fixed` _[FixedPeriod](#fixedperiod)_ |  |  |  |


