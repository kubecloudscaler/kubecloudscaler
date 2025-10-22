# API Reference

## Packages
- [kubecloudscaler.cloud/common](#kubecloudscalercloudcommon)
- [kubecloudscaler.cloud/v1alpha3](#kubecloudscalercloudv1alpha3)


## kubecloudscaler.cloud/common

Package common contains shared API Schema definitions for the kubecloudscaler project.

Package common contains shared API Schema definitions for the kubecloudscaler project.

Package common provides shared API Schema definitions for the kubecloudscaler project.



#### FixedPeriod



FixedPeriod defines a fixed time period for scaling operations.



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


#### RecurringPeriod



RecurringPeriod defines a recurring time period for scaling operations.



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


#### Resources



Resources defines the configuration for managed resources.



_Appears in:_
- [GcpResource](#gcpresource)
- [GcpSpec](#gcpspec)
- [K8sResource](#k8sresource)
- [K8sSpec](#k8sspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `types` _string array_ | Types of resources<br />K8s: deployments, statefulsets, ... (default: deployments)<br />GCP: VM-instances, ... (default: vm-instances) |  |  |
| `names` _string array_ | Names of resources to manage |  |  |
| `labelSelector` _[LabelSelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#labelselector-v1-meta)_ | Labels selectors |  |  |


#### ScalerPeriod



ScalerPeriod defines a scaling period with time constraints and replica limits.



_Appears in:_
- [FlowSpec](#flowspec)
- [GcpSpec](#gcpspec)
- [K8sSpec](#k8sspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ |  |  | Enum: [down up] <br /> |
| `time` _[TimePeriod](#timeperiod)_ |  |  |  |
| `minReplicas` _integer_ | Minimum replicas |  |  |
| `maxReplicas` _integer_ | Maximum replicas |  |  |
| `name` _string_ | Name of the period |  | Pattern: `^(\|[a-zA-Z0-9][a-zA-Z0-9_-]*)$` <br /> |


#### ScalerStatus



ScalerStatus defines the observed state of Scaler.



_Appears in:_
- [Gcp](#gcp)
- [K8s](#k8s)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `currentPeriod` _[ScalerStatusPeriod](#scalerstatusperiod)_ |  |  |  |
| `comments` _string_ |  |  |  |


#### ScalerStatusFailed



ScalerStatusFailed represents a failed scaling operation.



_Appears in:_
- [ScalerStatusPeriod](#scalerstatusperiod)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `kind` _string_ |  |  |  |
| `name` _string_ |  |  |  |
| `reason` _string_ |  |  |  |


#### ScalerStatusPeriod



ScalerStatusPeriod defines the current period status for a scaler.



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



ScalerStatusSuccess represents a successful scaling operation.



_Appears in:_
- [ScalerStatusPeriod](#scalerstatusperiod)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `kind` _string_ |  |  |  |
| `name` _string_ |  |  |  |
| `comment` _string_ |  |  |  |


#### TimePeriod



TimePeriod defines the time configuration for a scaling period.



_Appears in:_
- [ScalerPeriod](#scalerperiod)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `recurring` _[RecurringPeriod](#recurringperiod)_ |  |  |  |
| `fixed` _[FixedPeriod](#fixedperiod)_ |  |  |  |



## kubecloudscaler.cloud/v1alpha3

Package v1alpha3 contains API Schema definitions for the kubecloudscaler v1alpha3 API group.

Package v1alpha3 contains API Schema definitions for the  v1alpha3 API group.

### Resource Types
- [Flow](#flow)
- [FlowList](#flowlist)
- [Gcp](#gcp)
- [K8s](#k8s)



#### Flow



Flow is the Schema for the flows API



_Appears in:_
- [FlowList](#flowlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `kubecloudscaler.cloud/v1alpha3` | | |
| `kind` _string_ | `Flow` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[FlowSpec](#flowspec)_ | spec defines the desired state of Flow |  |  |
| `status` _[FlowStatus](#flowstatus)_ | status defines the observed state of Flow |  |  |


#### FlowList



FlowList contains a list of Flow





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `kubecloudscaler.cloud/v1alpha3` | | |
| `kind` _string_ | `FlowList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[Flow](#flow) array_ |  |  |  |


#### FlowResource



FlowResource defines a resource within a flow.



_Appears in:_
- [Flows](#flows)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `delay` _string_ | Delay is the duration to delay the start of the period<br />It is a duration in minutes<br />It is optional and if not provided, the period will start at the start time of the period |  | Pattern: `^\d*m$` <br /> |


#### FlowSpec



FlowSpec defines the desired state of Flow



_Appears in:_
- [Flow](#flow)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `periods` _[ScalerPeriod](#scalerperiod) array_ | Time period to scale |  |  |
| `resources` _[Resources](#resources)_ | Resources |  |  |
| `flows` _[Flows](#flows) array_ |  |  |  |


#### FlowStatus



FlowStatus defines the observed state of Flow.



_Appears in:_
- [Flow](#flow)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#condition-v1-meta) array_ | conditions represent the current state of the Flow resource.<br />Each condition has a unique type and reflects the status of a specific aspect of the resource.<br />Standard condition types include:<br />- "Available": the resource is fully functional<br />- "Progressing": the resource is being created or updated<br />- "Degraded": the resource failed to reach or maintain its desired state<br />The status of each condition is one of True, False, or Unknown. |  |  |


#### Flows



Flows defines a flow configuration with period and resources.



_Appears in:_
- [FlowSpec](#flowspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `periodName` _string_ |  |  |  |
| `resources` _[FlowResource](#flowresource) array_ |  |  |  |


#### Gcp



Gcp is the Schema for the gcps API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `kubecloudscaler.cloud/v1alpha3` | | |
| `kind` _string_ | `Gcp` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[GcpSpec](#gcpspec)_ | spec defines the desired state of Gcp |  |  |
| `status` _[ScalerStatus](#scalerstatus)_ | status defines the observed state of Gcp |  |  |


#### GcpConfig



GcpConfig defines the configuration for GCP resource management.



_Appears in:_
- [GcpResource](#gcpresource)
- [GcpSpec](#gcpspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `projectId` _string_ | ProjectID |  |  |
| `region` _string_ | Region |  |  |
| `authSecret` _string_ | AuthSecret name |  |  |
| `restoreOnDelete` _boolean_ | Restore resource state on CR deletion (default: true) | true |  |
| `waitForOperation` _boolean_ | Wait for operation to complete |  |  |
| `defaultPeriodType` _string_ | Default status for resources | down | Enum: [down up] <br /> |


#### GcpResource



GcpResource defines a GCP resource configuration in a flow.



_Appears in:_
- [Resources](#resources)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `resources` _[Resources](#resources)_ |  |  |  |
| `config` _[GcpConfig](#gcpconfig)_ |  |  |  |


#### GcpSpec



GcpSpec defines the desired state of Gcp



_Appears in:_
- [Gcp](#gcp)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `dryRun` _boolean_ | dry-run mode |  |  |
| `periods` _[ScalerPeriod](#scalerperiod) array_ | Time period to scale |  |  |
| `resources` _[Resources](#resources)_ | Resources |  |  |
| `config` _[GcpConfig](#gcpconfig)_ |  |  |  |


#### K8s



K8s is the Schema for the k8s API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `kubecloudscaler.cloud/v1alpha3` | | |
| `kind` _string_ | `K8s` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[K8sSpec](#k8sspec)_ | spec defines the desired state of K8s |  |  |
| `status` _[ScalerStatus](#scalerstatus)_ | status defines the observed state of K8s |  |  |


#### K8sConfig



K8sConfig defines the configuration for Kubernetes resource management.



_Appears in:_
- [K8sResource](#k8sresource)
- [K8sSpec](#k8sspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `namespaces` _string array_ | Namespaces |  |  |
| `excludeNamespaces` _string array_ | Exclude namespaces from downscaling; will be ignored if `Namespaces` is set |  |  |
| `forceExcludeSystemNamespaces` _boolean_ | Force exclude system namespaces | true |  |
| `deploymentTimeAnnotation` _string_ | Deployment time annotation |  |  |
| `disableEvents` _boolean_ | Disable events |  |  |
| `authSecret` _string_ | AuthSecret name |  |  |
| `restoreOnDelete` _boolean_ | Restore resource state on CR deletion (default: true) | true |  |


#### K8sResource



K8sResource defines a Kubernetes resource configuration in a flow.



_Appears in:_
- [Resources](#resources)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `resources` _[Resources](#resources)_ |  |  |  |
| `config` _[K8sConfig](#k8sconfig)_ |  |  |  |


#### K8sSpec



K8sSpec defines the desired state of K8s



_Appears in:_
- [K8s](#k8s)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `dryRun` _boolean_ | dry-run mode |  |  |
| `periods` _[ScalerPeriod](#scalerperiod) array_ | Time period to scale |  |  |
| `resources` _[Resources](#resources)_ | Resources |  |  |
| `config` _[K8sConfig](#k8sconfig)_ |  |  |  |


#### Resources



Resources defines the configuration for managed resources in a flow.



_Appears in:_
- [FlowSpec](#flowspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `k8s` _[K8sResource](#k8sresource) array_ |  |  |  |
| `gcp` _[GcpResource](#gcpresource) array_ |  |  |  |


