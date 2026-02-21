# API Reference

## Packages
- [common](#common)
- [kubecloudscaler.cloud/v1alpha3](#kubecloudscalercloudv1alpha3)


## common

Package common contains shared API Schema definitions for the kubecloudscaler project.

#### common.ScalerPeriod

ScalerPeriod defines a scaling period with time constraints and replica limits.

_Appears in:_
- [kubecloudscaler.cloud/v1alpha3.FlowSpec](#kubecloudscalercloudv1alpha3flowspec)
- [kubecloudscaler.cloud/v1alpha3.GcpSpec](#kubecloudscalercloudv1alpha3gcpspec)
- [kubecloudscaler.cloud/v1alpha3.K8sSpec](#kubecloudscalercloudv1alpha3k8sspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ |   |   | Enum: [down up] |
| `time` _[common.TimePeriod](#commontimeperiod)_ |   |   |   |
| `minReplicas` _integer_ | Minimum replicas |   |   |
| `maxReplicas` _integer_ | Maximum replicas |   |   |
| `name` _string_ | Name of the period |   | Pattern: `^(\|[a-zA-Z0-9][a-zA-Z0-9_-]*)$` |



#### common.TimePeriod

TimePeriod defines the time configuration for a scaling period.
Name of the period

_Appears in:_
- [common.ScalerPeriod](#commonscalerperiod)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `recurring` _[common.RecurringPeriod](#commonrecurringperiod)_ |   |   |   |
| `fixed` _[common.FixedPeriod](#commonfixedperiod)_ |   |   |   |



#### common.RecurringPeriod

RecurringPeriod defines a recurring time period for scaling operations.

_Appears in:_
- [common.TimePeriod](#commontimeperiod)
- [common.ScalerStatusPeriod](#commonscalerstatusperiod)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `days` _string array_ |   |   |   |
| `startTime` _string_ |   |   | Pattern: `^([0-1]?[0-9]\|2[0-3]):[0-5][0-9]$` |
| `endTime` _string_ |   |   | Pattern: `^([0-1]?[0-9]\|2[0-3]):[0-5][0-9]$` |
| `timezone` _string_ |   |   |   |
| `once` _boolean_ | Run once at StartTime |   |   |
| `gracePeriod` _string_ |   |   | Pattern: `^\d*s$` |
| `reverse` _boolean_ | Reverse the period |   |   |



#### common.FixedPeriod

FixedPeriod defines a fixed time period for scaling operations.

_Appears in:_
- [common.TimePeriod](#commontimeperiod)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `startTime` _string_ |   |   | Pattern: `^\d\{4\}-(0?[1-9]\|1[0,1,2])-(0?[1-9]\|[12][0-9]\|3[01])` |
| `endTime` _string_ |   |   | Pattern: `^\d\{4\}-(0?[1-9]\|1[0,1,2])-(0?[1-9]\|[12][0-9]\|3[01])` |
| `timezone` _string_ |   |   |   |
| `once` _boolean_ | Run once at StartTime |   |   |
| `gracePeriod` _string_ | Grace period in seconds for deployments before scaling down |   | Pattern: `^\d*s$` |
| `reverse` _boolean_ | Reverse the period |   |   |



#### common.ScalerStatus

ScalerStatus defines the observed state of Scaler.

_Appears in:_
- [kubecloudscaler.cloud/v1alpha3.Gcp](#kubecloudscalercloudv1alpha3gcp)
- [kubecloudscaler.cloud/v1alpha3.K8s](#kubecloudscalercloudv1alpha3k8s)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `currentPeriod` _[common.ScalerStatusPeriod](#commonscalerstatusperiod)_ |   |   |   |
| `comments` _string_ |   |   |   |



#### common.ScalerStatusPeriod

ScalerStatusPeriod defines the current period status for a scaler.

_Appears in:_
- [common.ScalerStatus](#commonscalerstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `spec` _[common.RecurringPeriod](#commonrecurringperiod)_ |   |   |   |
| `specSHA` _string_ |   |   |   |
| `name` _string_ |   |   |   |
| `type` _string_ |   |   |   |
| `success` _[common.ScalerStatusSuccess](#commonscalerstatussuccess) array_ |   |   |   |
| `failed` _[common.ScalerStatusFailed](#commonscalerstatusfailed) array_ |   |   |   |



#### common.ScalerStatusSuccess

ScalerStatusSuccess represents a successful scaling operation.

_Appears in:_
- [common.ScalerStatusPeriod](#commonscalerstatusperiod)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `kind` _string_ |   |   |   |
| `name` _string_ |   |   |   |
| `comment` _string_ |   |   |   |



#### common.ScalerStatusFailed

ScalerStatusFailed represents a failed scaling operation.

_Appears in:_
- [common.ScalerStatusPeriod](#commonscalerstatusperiod)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `kind` _string_ |   |   |   |
| `name` _string_ |   |   |   |
| `reason` _string_ |   |   |   |



#### common.Resources

Resources defines the configuration for managed resources.

_Appears in:_
- [kubecloudscaler.cloud/v1alpha3.K8sResource](#kubecloudscalercloudv1alpha3k8sresource)
- [kubecloudscaler.cloud/v1alpha3.GcpResource](#kubecloudscalercloudv1alpha3gcpresource)
- [kubecloudscaler.cloud/v1alpha3.GcpSpec](#kubecloudscalercloudv1alpha3gcpspec)
- [kubecloudscaler.cloud/v1alpha3.K8sSpec](#kubecloudscalercloudv1alpha3k8sspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `types` _string array_ | Types of resources K8s: deployments, statefulsets, ... (default: deployments) GCP: VM-instances, ... (default: vm-instances) |   |   |
| `names` _string array_ | Names of resources to manage |   |   |
| `labelSelector` _[LabelSelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#labelselector-v1-meta)_ | Labels selectors |   |   |





## kubecloudscaler.cloud/v1alpha3

Package v1alpha3 contains API Schema definitions for the kubecloudscaler v1alpha3 API group.

### Resource Types
- [kubecloudscaler.cloud/v1alpha3.Flow](#kubecloudscalercloudv1alpha3flow)
- [kubecloudscaler.cloud/v1alpha3.Gcp](#kubecloudscalercloudv1alpha3gcp)
- [kubecloudscaler.cloud/v1alpha3.K8s](#kubecloudscalercloudv1alpha3k8s)


#### kubecloudscaler.cloud/v1alpha3.Flow

Flow is the Schema for the flows API

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `kubecloudscaler.cloud/v1alpha3` |   |   |
| `kind` _string_ | `Flow` |   |   |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |   |   |
| `spec` _[kubecloudscaler.cloud/v1alpha3.FlowSpec](#kubecloudscalercloudv1alpha3flowspec)_ | spec defines the desired state of Flow |   |   |
| `status` _[kubecloudscaler.cloud/v1alpha3.FlowStatus](#kubecloudscalercloudv1alpha3flowstatus)_ | status defines the observed state of Flow |   |   |



#### kubecloudscaler.cloud/v1alpha3.Gcp

Gcp is the Schema for the gcps API

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `kubecloudscaler.cloud/v1alpha3` |   |   |
| `kind` _string_ | `Gcp` |   |   |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |   |   |
| `spec` _[kubecloudscaler.cloud/v1alpha3.GcpSpec](#kubecloudscalercloudv1alpha3gcpspec)_ | spec defines the desired state of Gcp |   |   |
| `status` _[common.ScalerStatus](#commonscalerstatus)_ | status defines the observed state of Gcp |   |   |



#### kubecloudscaler.cloud/v1alpha3.K8s

K8s is the Schema for the k8s API

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `kubecloudscaler.cloud/v1alpha3` |   |   |
| `kind` _string_ | `K8s` |   |   |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |   |   |
| `spec` _[kubecloudscaler.cloud/v1alpha3.K8sSpec](#kubecloudscalercloudv1alpha3k8sspec)_ | spec defines the desired state of K8s |   |   |
| `status` _[common.ScalerStatus](#commonscalerstatus)_ | status defines the observed state of K8s |   |   |



#### kubecloudscaler.cloud/v1alpha3.FlowSpec

FlowSpec defines the desired state of Flow

_Appears in:_
- [kubecloudscaler.cloud/v1alpha3.Flow](#kubecloudscalercloudv1alpha3flow)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `periods` _[common.ScalerPeriod](#commonscalerperiod) array_ | Time period to scale |   |   |
| `resources` _[kubecloudscaler.cloud/v1alpha3.Resources](#kubecloudscalercloudv1alpha3resources)_ | Resources |   |   |
| `flows` _[kubecloudscaler.cloud/v1alpha3.Flows](#kubecloudscalercloudv1alpha3flows) array_ |   |   |   |



#### kubecloudscaler.cloud/v1alpha3.Resources

Resources defines the configuration for managed resources in a flow.

_Appears in:_
- [kubecloudscaler.cloud/v1alpha3.FlowSpec](#kubecloudscalercloudv1alpha3flowspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `k8s` _[kubecloudscaler.cloud/v1alpha3.K8sResource](#kubecloudscalercloudv1alpha3k8sresource) array_ |   |   |   |
| `gcp` _[kubecloudscaler.cloud/v1alpha3.GcpResource](#kubecloudscalercloudv1alpha3gcpresource) array_ |   |   |   |



#### kubecloudscaler.cloud/v1alpha3.K8sResource

K8sResource defines a Kubernetes resource configuration in a flow.

_Appears in:_
- [kubecloudscaler.cloud/v1alpha3.Resources](#kubecloudscalercloudv1alpha3resources)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |   |   |   |
| `resources` _[common.Resources](#commonresources)_ |   |   |   |
| `config` _[kubecloudscaler.cloud/v1alpha3.K8sConfig](#kubecloudscalercloudv1alpha3k8sconfig)_ |   |   |   |



#### kubecloudscaler.cloud/v1alpha3.GcpResource

GcpResource defines a GCP resource configuration in a flow.

_Appears in:_
- [kubecloudscaler.cloud/v1alpha3.Resources](#kubecloudscalercloudv1alpha3resources)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |   |   |   |
| `resources` _[common.Resources](#commonresources)_ |   |   |   |
| `config` _[kubecloudscaler.cloud/v1alpha3.GcpConfig](#kubecloudscalercloudv1alpha3gcpconfig)_ |   |   |   |



#### kubecloudscaler.cloud/v1alpha3.Flows

Flows defines a flow configuration with period and resources.

_Appears in:_
- [kubecloudscaler.cloud/v1alpha3.FlowSpec](#kubecloudscalercloudv1alpha3flowspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `periodName` _string_ |   |   |   |
| `resources` _[kubecloudscaler.cloud/v1alpha3.FlowResource](#kubecloudscalercloudv1alpha3flowresource) array_ |   |   |   |



#### kubecloudscaler.cloud/v1alpha3.FlowResource

FlowResource defines a resource within a flow.

_Appears in:_
- [kubecloudscaler.cloud/v1alpha3.Flows](#kubecloudscalercloudv1alpha3flows)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |   |   |   |
| `startTimeDelay` _string_ | StartTimeDelay is the duration to delay the start of the period It is a duration in minutes It is optional and if not provided, the period will start at the start time of the period | 0m | Pattern: `^\d*m$` |
| `endTimeDelay` _string_ | EndTimeDelay is the duration to delay the end of the period It is a duration in minutes It is optional and if not provided, the period will end at the end time of the period | 0m | Pattern: `^\d*m$` |



#### kubecloudscaler.cloud/v1alpha3.FlowStatus

FlowStatus defines the observed state of Flow.
EndTimeDelay is the duration to delay the end of the period It is a duration in minutes It is optional and if not provided, the period will end at the end time of the period

_Appears in:_
- [kubecloudscaler.cloud/v1alpha3.Flow](#kubecloudscalercloudv1alpha3flow)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.33/#condition-v1-meta) array_ | conditions represent the current state of the Flow resource. Each condition has a unique type and reflects the status of a specific aspect of the resource. Standard condition types include: - "Available": the resource is fully functional - "Progressing": the resource is being created or updated - "Degraded": the resource failed to reach or maintain its desired state The status of each condition is one of True, False, or Unknown. |   |   |



#### kubecloudscaler.cloud/v1alpha3.GcpSpec

GcpSpec defines the desired state of Gcp

_Appears in:_
- [kubecloudscaler.cloud/v1alpha3.Gcp](#kubecloudscalercloudv1alpha3gcp)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `dryRun` _boolean_ | dry-run mode |   |   |
| `periods` _[common.ScalerPeriod](#commonscalerperiod) array_ | Time period to scale |   |   |
| `resources` _[common.Resources](#commonresources)_ | Resources |   |   |
| `config` _[kubecloudscaler.cloud/v1alpha3.GcpConfig](#kubecloudscalercloudv1alpha3gcpconfig)_ |   |   |   |



#### kubecloudscaler.cloud/v1alpha3.GcpConfig

GcpConfig defines the configuration for GCP resource management.

_Appears in:_
- [kubecloudscaler.cloud/v1alpha3.GcpResource](#kubecloudscalercloudv1alpha3gcpresource)
- [kubecloudscaler.cloud/v1alpha3.GcpSpec](#kubecloudscalercloudv1alpha3gcpspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `projectId` _string_ | ProjectID |   |   |
| `region` _string_ | Region |   |   |
| `authSecret` _string_ | AuthSecret name |   |   |
| `restoreOnDelete` _boolean_ | RestoreOnDelete applies defaultPeriodType to all managed resources when the CR is deleted. Note: this does NOT restore the pre-CR state of resources. It applies the defaultPeriodType value (default: "down"), meaning VMs will be stopped on deletion unless defaultPeriodType is set to "up". To restore VMs to their original state, set defaultPeriodType accordingly. | true |   |
| `waitForOperation` _boolean_ | Wait for operation to complete |   |   |
| `defaultPeriodType` _string_ | Default status for resources | down | Enum: [down up] |



#### kubecloudscaler.cloud/v1alpha3.K8sSpec

K8sSpec defines the desired state of K8s

_Appears in:_
- [kubecloudscaler.cloud/v1alpha3.K8s](#kubecloudscalercloudv1alpha3k8s)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `dryRun` _boolean_ | dry-run mode |   |   |
| `periods` _[common.ScalerPeriod](#commonscalerperiod) array_ | Time period to scale |   |   |
| `resources` _[common.Resources](#commonresources)_ | Resources |   |   |
| `config` _[kubecloudscaler.cloud/v1alpha3.K8sConfig](#kubecloudscalercloudv1alpha3k8sconfig)_ |   |   |   |



#### kubecloudscaler.cloud/v1alpha3.K8sConfig

K8sConfig defines the configuration for Kubernetes resource management.

_Appears in:_
- [kubecloudscaler.cloud/v1alpha3.K8sResource](#kubecloudscalercloudv1alpha3k8sresource)
- [kubecloudscaler.cloud/v1alpha3.K8sSpec](#kubecloudscalercloudv1alpha3k8sspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `namespaces` _string array_ | Namespaces |   |   |
| `excludeNamespaces` _string array_ | Exclude namespaces from downscaling; will be ignored if `Namespaces` is set |   |   |
| `forceExcludeSystemNamespaces` _boolean_ | Force exclude system namespaces | true |   |
| `deploymentTimeAnnotation` _string_ | Deployment time annotation |   |   |
| `disableEvents` _boolean_ | Disable events |   |   |
| `authSecret` _string_ | AuthSecret name |   |   |
| `restoreOnDelete` _boolean_ | Restore resource state on CR deletion (default: true) | true |   |





