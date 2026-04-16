# API - CRD Type Definitions

## CRD Kinds

Three CRD kinds, each with multiple API versions:
- `K8s` - Scales Kubernetes workloads (Deployments, StatefulSets, CronJobs, HPAs, ScaledObjects)
- `Gcp` - Scales GCP resources (Compute Engine VM instances)
- `Flow` - Orchestrates scaling workflows across K8s and GCP resources

All CRDs are **cluster-scoped** (`+kubebuilder:resource:scope=Cluster`).

## API Versions

- `v1alpha1` - First version
- `v1alpha2` - Second version
- **`v1alpha3`** - Storage version (`+kubebuilder:storageversion`)

Shared types live in `api/common/` (periods, resources, status).

## Multi-Version CRD Management

- v1alpha3 is the **storage version**
- Conversion webhooks handle: v1alpha1 <-> v1alpha3 and v1alpha2 <-> v1alpha3
- All conversions MUST be lossless (round-trip safe)
- Conversion functions live in `api/v1alpha3/*_conversion.go`
- Test conversions bidirectionally

## Adding a New CRD Field

1. Add field to v1alpha3 types in `api/v1alpha3/`
2. Update common types in `api/common/` if shared across kinds
3. Add kubebuilder markers for validation (`+kubebuilder:validation:*`)
4. Update conversion functions in `api/v1alpha3/*_conversion.go`
5. Run `make generate && make manifests`
6. Update webhook validation if applicable
7. Add tests for new field behavior and conversion

## Resource Types

Use valid `common.ResourceKind` constants (defined in `api/common/resources_type.go`):
- K8s: `common.ResourceDeployments`, `common.ResourceStatefulSets`, `common.ResourceCronJobs`, `common.ResourceHPA`, `common.ResourceGithubARS`, `common.ResourceScaledObjects`
- GCP: only `common.ResourceVMInstances` (`"vm-instances"`)
- Do NOT use arbitrary strings like `"instance"`, `"disk"`

## Validation

- Use CEL expressions in CRD markers for declarative validation where possible
- Webhook validation in `internal/webhook/v1alpha3/`
- Status uses the subresource pattern (`+kubebuilder:subresource:status`)
- NEVER update status and spec in the same API call
