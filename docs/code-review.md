# KubeCloudScaler — Senior Go Engineer Code Review

**Date**: 2026-03-12
**Reviewer**: Senior Go Engineer (TDD / DDD / Clean Architecture)
**Scope**: Full codebase — 168 Go files, 58 test files, 34 packages

---

## Overall Assessment

| Dimension | Grade | Notes |
|-----------|-------|-------|
| **Architecture** | B | Solid Chain of Responsibility, clean layer separation. Flow controller is the outlier. |
| **Domain Design** | C- | Anemic models, stringly-typed enums, exported mutable state everywhere. |
| **Error Handling** | C+ | Error classification exists but inconsistent wrapping, swallowed errors in adapters. |
| **Test Quality** | D+ | Structural coverage exists; behavioral coverage ~25-30%. Non-deterministic tests. |
| **Code Duplication** | D | K8s/GCP are 80%+ identical. 5x adapter duplication. Constants duplicated across packages. |
| **Security** | C | GCP client resource leak, unclosed connections, no webhook validation for K8s/GCP CRDs. |
| **Linting/CI** | C- | Tests excluded from linting, wrong Go version, no coverage threshold enforcement. |

**Total findings: 11 CRITICAL, 39 HIGH, 59 MEDIUM, 42 LOW**

---

## Todo List

### P0 — Fix Immediately (production bugs & data loss)

- [x] **P0-1**: Fix `omitempty` + non-zero default on booleans (`ForceExcludeSystemNamespaces`, `RestoreOnDelete`) — users cannot set `false`
- [x] **P0-2**: Fix GCP client resource leak — implement `io.Closer` on `ClientSet`, defer `Close()` in auth handler
- [x] **P0-3**: Fix GCP client ignoring caller context — accept `ctx context.Context` as first parameter
- [x] **P0-4**: Fix race condition on lazy chain initialization — use `sync.Once`
- [x] **P0-5**: Fix namespace cache thread safety — add `sync.RWMutex`
- [x] **P0-6**: Fix data loss in v1alpha1 conversions — store unrepresentable fields in annotations

### P1 — Fix Next Sprint (architecture violations, missing safety nets)

- [x] **P1-1**: Add `CustomValidator` implementations for K8s and GCP webhooks
- [x] **P1-2**: Refactor Flow controller to Chain of Responsibility pattern
- [x] **P1-3**: Add `Owns()` watches to Flow controller for child K8s/Gcp resources
- [x] **P1-4**: Extract shared K8s/GCP code (errors, interfaces, reconcile loop, finalizer handler)
- [x] **P1-5**: Fix `.golangci.yml` — set `go: "1.25"`, `tests: true`, fix depguard messages
- [x] **P1-6**: Add `NewScalerReconciler()` constructors for K8s/GCP, inject `metrics.Recorder` via DI
- [x] **P1-7**: Replace `interface{}` with typed parameters in annotation/period interfaces
- [x] **P1-8**: Rename `IResource` to `Resource`, move interface to consumers

### P2 — Improve When Touching (quality & consistency)

- [x] **P2-1**: Add `Validate()` methods to API types (`ScalerPeriod`, `TimePeriod`, `Resources`)
- [x] **P2-2**: Extract named types for stringly-typed domain concepts (`PeriodType`, `ResourceKind`, `DayOfWeek`)
- [x] **P2-3**: Remove global logger from conversion functions
- [x] **P2-4**: Make exported mutable slices immutable (return copies from functions)
- [x] **P2-5**: Fix GCP StatusHandler unconditional `RequeueAfter` override
- [x] **P2-6**: Extract `typeAssertionError` to `pkg/k8s/resources/base/errors.go`
- [x] **P2-7**: Extract duplicated constants to shared package
- [x] **P2-8**: Replace Flow `resource_creator.go` Create-then-Update with SSA or `CreateOrUpdate`
- [x] **P2-9**: Remove `ctrl.Result` from Flow `StatusUpdaterService` return type
- [x] **P2-10**: Inject `time.Now` into period logic via `Clock` interface
- [x] **P2-11**: Fix `Period` struct field naming (`Period.Period` → `Spec`, `GetStartTime` → `StartTime`)
- [x] **P2-12**: Fix type name typo `VMnstances` → `VMInstances`
- [x] **P2-13**: Fix overnight recurring period rejection in Flow webhook
- [x] **P2-14**: Move mocks from production packages to `_test.go` or `testutil/`
- [x] **P2-15**: Handle discarded `processResource` errors in `base/processor.go`
- [x] **P2-16**: Add coverage threshold enforcement to Makefile/CI

### Test Improvements

- [x] **T1**: Fix non-deterministic tests (replace `if err != nil` conditional assertions with deterministic expectations)
- [x] **T2**: Add RecoverableError test coverage for all 4 handler paths that return it
- [x] **T3**: Write tests for `pkg/resources/resources.go` factory
- [x] **T4**: Write tests for `pkg/k8s/resources/base/processor.go`
- [x] **T5**: Write tests for `pkg/k8s/resources/base/strategies.go`
- [x] **T6**: Add CRD conversion round-trip tests (v1alpha1 ↔ v1alpha3, v1alpha2 ↔ v1alpha3)
- [x] **T7**: Replace stub webhook tests (K8s, GCP) with real validation tests
- [x] **T8**: Replace `cmd/main_test.go` placeholder assertions (`Expect(true).To(BeTrue())`)
- [x] **T9**: Convert Flow service tests from raw `testing.T` to Ginkgo BDD
- [x] **T10**: Delete fake "performance tests" that discard results, replace with `testing.B` benchmarks
- [x] **T11**: Add period `IsActive` / activation tests with time injection
- [x] **T12**: Add RequeueAfter value assertions to handler tests
- [ ] **T13**: Expand E2E tests to cover actual reconciliation, scaling, and status verification
- [x] **T14**: Add table-driven tests for multi-scenario functions (validResourceList, period types, resource kinds)

---

## Detailed Findings

### 1. API / Domain Layer

#### 1.1 CRITICAL: `omitempty` + non-zero default on booleans

**Files**: `api/v1alpha3/k8s_types.go`, `api/v1alpha3/gcp_types.go`, `api/v1alpha2/` equivalents

```go
// +kubebuilder:default:=true
ForceExcludeSystemNamespaces bool `json:"forceExcludeSystemNamespaces,omitempty"`
RestoreOnDelete              bool `json:"restoreOnDelete,omitempty"`
```

When users set `restoreOnDelete: false`, JSON marshaling drops the field (because `omitempty` + zero value), then the defaulting webhook resets it to `true`. **Users can never disable this feature.**

**Fix**: Remove `omitempty` from bool fields with non-zero defaults, or switch to `*bool` pointer fields.

#### 1.2 CRITICAL: Data loss in v1alpha1 conversions

**Files**: `api/v1alpha1/k8s_conversion.go`, `api/v1alpha1/gcp_conversion.go`

| Lost Field | Direction | Impact |
|------------|-----------|--------|
| `K8s.ExcludeResources` | v1alpha1 → v1alpha3 → v1alpha1 | Silently dropped |
| `Gcp.ExcludeResources` | v1alpha1 → v1alpha3 → v1alpha1 | Silently dropped |
| `Gcp.DeploymentTimeAnnotation` | v1alpha1 → v1alpha3 → v1alpha1 | Silently dropped |
| `Gcp.DefaultPeriodType` | v1alpha3 → v1alpha1 → v1alpha3 | Hardcoded to `"down"` |

**Fix**: Store unrepresentable fields in annotations (standard Kubernetes conversion pattern).

#### 1.3 HIGH: Anemic domain models

All `api/common/` types are pure data bags. `ScalerPeriod`, `TimePeriod`, `Resources`, `ScalerStatus` have no `Validate()` method, no business methods. Invariants (mutual exclusion of Recurring/Fixed, valid day names, min <= max replicas) are either enforced only by kubebuilder markers or not at all.

#### 1.4 HIGH: Stringly-typed domain concepts

| Field | Files | Proposed Type |
|-------|-------|---------------|
| `ScalerPeriod.Type` | `common/periods_type.go:7` | `PeriodType` string type |
| `DefaultPeriodType` | `v1alpha3/gcp_types.go:64` | Reuse `PeriodType` |
| `Resources.Types` | `common/resources_type.go:16` | `ResourceKind` string type |
| `RecurringPeriod.Days` | `common/periods_type.go:27` | `DayOfWeek` string type |

#### 1.5 MEDIUM: Missing mutual exclusion validation on `TimePeriod`

```go
type TimePeriod struct {
    Recurring *RecurringPeriod `json:"recurring,omitempty"`
    Fixed     *FixedPeriod     `json:"fixed,omitempty"`
}
```

Nothing prevents both `Recurring` and `Fixed` from being set simultaneously, or both being nil. No `+kubebuilder:validation` or CEL expression enforces this.

#### 1.6 MEDIUM: `GracePeriod` regex too permissive

```go
// +kubebuilder:validation:Pattern=`^\d*s$`
```

Matches `"s"` (zero digits). Should be `^\d+s$`.

#### 1.7 MEDIUM: `ScalerStatus` in wrong file

`ScalerStatus` and subtypes are in `periods_type.go` but have nothing to do with periods. Belongs in `status_type.go`.

#### 1.8 MEDIUM: JSON tag mismatch

```go
Successful []ScalerStatusSuccess `json:"success,omitempty"`
```

Go field `Successful`, JSON key `success`. Confusing asymmetry.

#### 1.9 MEDIUM: RBAC markers scattered across API versions

v1alpha1 has RBAC for `apps` group, v1alpha3 has RBAC for `core`. RBAC markers belong on controller files, not API types.

#### 1.10 HIGH: Global logger in conversion functions

All 4 conversion files import `github.com/rs/zerolog/log`. Conversions should be pure functions with zero infrastructure dependencies.

#### 1.11 MEDIUM: `Flow` missing `+kubebuilder:storageversion` marker

`K8s` and `Gcp` both have it, `Flow` does not (`api/v1alpha3/flow_types.go:105-107`).

#### 1.12 LOW: Scaffolding comments still present

`"EDIT THIS FILE! THIS IS SCAFFOLDING FOR YOU TO OWN!"` in `gcp_types.go`, `flow_types.go` across multiple versions.

---

### 2. Controller Layer & Handler Chains

#### 2.1 CRITICAL: Race condition on lazy chain initialization

**Files**: `internal/controller/k8s/scaler_controller.go:86-88`, `internal/controller/gcp/scaler_controller.go:82-84`

```go
if r.chain == nil {
    r.chain = r.initializeChain()
}
```

`Reconcile()` is called concurrently by controller-runtime. This lazy initialization is not thread-safe.

**Fix**: Use `sync.Once` or remove the lazy fallback and require `SetupWithManager` exclusively.

#### 2.2 MAJOR: K8s/GCP code duplication (~80% identical)

| Duplicated Component | K8s Location | GCP Location |
|---------------------|--------------|--------------|
| `errors.go` (100% identical) | `k8s/service/errors.go` | `gcp/service/errors.go` |
| `interfaces.go` (Handler interface) | `k8s/service/interfaces.go` | `gcp/service/interfaces.go` |
| `Reconcile()` method | `k8s/scaler_controller.go:68-129` | `gcp/scaler_controller.go:64-125` |
| `initializeChain()` | `k8s/scaler_controller.go:154-173` | `gcp/scaler_controller.go:150-169` |
| `toScalingResults` helpers | `k8s/scaler_controller.go:131-145` | `gcp/scaler_controller.go:127-141` |
| `ScalerFinalizer` constant | `k8s/finalizer_handler.go:27` | `gcp/finalizer_handler.go:28` |

**Fix**: Extract to `internal/controller/shared/`.

#### 2.3 MAJOR: Flow controller violates mandatory Chain of Responsibility

**File**: `internal/controller/flow/flow_controller.go:98-143`

CLAUDE.md: "All controllers MUST implement reconciliation logic using the Chain of Responsibility pattern. The `Reconcile()` method MUST NOT contain business logic directly." The Flow reconciler has business logic in `Reconcile()`.

#### 2.4 MAJOR: Flow controller missing `Owns()` watches

**File**: `internal/controller/flow/flow_controller.go:200-205`

Creates child `K8s`/`Gcp` resources with owner references but doesn't watch them. If a child is deleted externally, the Flow controller won't reconcile to recreate it.

#### 2.5 MAJOR: Unclassified error branch in reconcilers

**Files**: `k8s/scaler_controller.go:110-112`, `gcp/scaler_controller.go:106-108`

Errors that are neither `CriticalError` nor `RecoverableError` are recorded as `ResultRecoverableError` in metrics — misleading. Should have its own label or be explicitly classified.

#### 2.6 MAJOR: Missing RBAC markers for Secret access

**Files**: `k8s/scaler_controller.go:50-52`, `gcp/scaler_controller.go:46-48`

`AuthHandler` reads `corev1.Secret` objects but no RBAC marker for secrets is on the controller file.

#### 2.7 MAJOR: GCP context uses concrete `*ClientSet` instead of interface

**File**: `internal/controller/gcp/service/context.go:75`

```go
GCPClient *gcpUtils.ClientSet
```

Per CLAUDE.md: "All dependencies MUST be defined as interfaces."

#### 2.8 MAJOR: Potential nil dereference on `ctx.Period` in status handlers

**Files**: `k8s/status_handler.go:92`, `gcp/status_handler.go:100`

```go
Str("period", ctx.Period.Name)  // ctx.Period could be nil
```

If the chain order changes or a handler is skipped, this panics.

#### 2.9 MAJOR: Flow `resource_creator.go` Create-then-Update is a TOCTOU race

**File**: `internal/controller/flow/service/resource_creator.go:161-201`

Between `Get` and `Update`, another process could modify the resource. Use `controllerutil.CreateOrUpdate` or Server-Side Apply.

#### 2.10 MAJOR: Flow `interface{}` in `ResourceInfo`

**File**: `internal/controller/flow/types/types.go:37`

```go
Resource interface{} // K8sResource or GcpResource
```

Forces unsafe type assertions. Use generics or separate typed mappings.

#### 2.11 MAJOR: Flow `StatusUpdaterService` returns `ctrl.Result` — layer violation

**File**: `internal/controller/flow/service/status_updater.go:48-52`

Service layer should not know about controller-runtime types.

#### 2.12 MINOR: GCP `BuildHandlerChain` in `chain.go` is dead code

**File**: `internal/controller/gcp/service/chain.go:27-37`

Exists but is never called by any controller.

#### 2.13 MINOR: GCP StatusHandler overwrites `RequeueAfter` unconditionally

**File**: `gcp/status_handler.go:106` vs K8s `status_handler.go:98-100`

GCP: `ctx.RequeueAfter = utils.ReconcileSuccessDuration` (always overwrites)
K8s: `if ctx.RequeueAfter == 0 { ... }` (respects first-write-wins)

#### 2.14 MINOR: GCP PeriodHandler also overwrites `RequeueAfter` unconditionally

**Files**: `gcp/period_handler.go:101-103`, `gcp/period_handler.go:123`

Same "first write wins" contract violation.

#### 2.15 MINOR: K8s FetchHandler missing error wrapping

**File**: `k8s/fetch_handler.go:47` — passes raw `err` into `NewRecoverableError`. GCP version correctly wraps with `fmt.Errorf`.

#### 2.16 MINOR: Flow controller returns `RequeueAfter` alongside non-nil error

**File**: `flow/flow_controller.go:166`

Controller-runtime may ignore `RequeueAfter` and use exponential backoff instead when error is non-nil.

#### 2.17 MINOR: K8s AuthHandler ignores type assertion `ok` result

**File**: `k8s/auth_handler.go:89` — `cc, _ := cached.(*cachedClient)` silently ignores failure.

#### 2.18 MINOR: Duplicate `createPeriodsMap` in Flow services

**Files**: `flow_validator.go:82-89`, `resource_mapper.go:63-70`

Identical methods on different receivers.

#### 2.19 MINOR: Duplicate interfaces in Flow controller vs service

**Files**: `flow/interfaces.go:31-68`, `flow/service/interfaces.go:32-82`

---

### 3. pkg/ Shared Packages

#### 3.1 CRITICAL: GCP clients never closed — resource leak

**File**: `pkg/gcp/utils/client/gcp.go`

Three `compute.*RESTClient` connections created. `Close()` never called anywhere in the codebase.

#### 3.2 CRITICAL: GCP client ignores caller context

**File**: `pkg/gcp/utils/client/gcp.go:26`

```go
ctx := context.Background()
```

Ignores caller cancellation/deadline.

#### 3.3 CRITICAL: Namespace cache not thread-safe

**File**: `pkg/k8s/utils/namespace_manager.go:34-39`

`cachedNsList` and `cacheExpiry` mutated without synchronization under concurrent reconciliation.

#### 3.4 HIGH: `time.Now()` not injectable in period logic

**File**: `pkg/period/period.go:114,157`

```go
localTime := time.Now().In(timeLocation)
```

Makes period evaluation non-deterministic and untestable at boundaries.

#### 3.5 HIGH: All `Period` fields exported and mutable

**File**: `pkg/period/types.go:11-22`

```go
type Period struct {
    Period       *common.RecurringPeriod  // Field named same as type
    Name         string
    Type         string
    IsActive     bool
    GetStartTime time.Time               // Field named like a method
    GetEndTime   time.Time               // Field named like a method
    ...
}
```

Callers can freely mutate `IsActive`, `Hash`, `MinReplicas` etc.

#### 3.6 HIGH: Exported mutable slice `AvailableResources`

**File**: `pkg/resources/vars.go:8`

```go
var AvailableResources = []string{...}
```

Any consumer can `append()` causing data races.

#### 3.7 HIGH: `base/processor.go` silently discards errors

**File**: `pkg/k8s/resources/base/processor.go:106`

```go
_ = p.processResource(ctx, item, &scalerStatusSuccess, &scalerStatusFailed)
```

Context cancellation errors are lost.

#### 3.8 HIGH: `typeAssertionError` duplicated in 5 adapter packages

**Files**: `deployments/adapter.go`, `statefulsets/adapter.go`, `cronjobs/adapter.go`, `hpa/adapter.go`, `ars/adapter.go`

Functionally identical with inconsistent error messages.

#### 3.9 HIGH: `interface{}` in annotation/period interfaces

**File**: `pkg/k8s/utils/interfaces.go:48,62-68`

```go
GetPeriod() interface{}
AddAnnotations(annotations map[string]string, period interface{}) map[string]string
```

Forces runtime type switches. `*period.Period` is always the concrete type.

#### 3.10 HIGH: `os.Getenv` called directly in namespace manager

**File**: `pkg/k8s/utils/namespace_manager.go:161`

```go
ownNamespace := os.Getenv("POD_NAMESPACE")
```

Untestable. An `EnvironmentProvider` already exists in `pkg/k8s/utils/client/`.

#### 3.11 HIGH: ARS adapter silently skips conversion failures

**File**: `pkg/k8s/resources/github_autoscalingrunnersets/adapter.go:77-79`

```go
if err := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, runnerSet); err != nil {
    continue  // Silently skipped
}
```

#### 3.12 HIGH: Constants duplicated between K8s and GCP utils

**Files**: `pkg/k8s/utils/consts.go`, `pkg/gcp/utils/consts.go`

`AnnotationsPrefix`, `FieldManager`, annotation keys — all duplicated verbatim.

#### 3.13 MEDIUM: `IResource` Java-style naming, defined at producer

**File**: `pkg/resources/types.go:19`

#### 3.14 MEDIUM: `isPeriodActive` returns 5 values

**File**: `pkg/period/period.go:139-142`

```go
func isPeriodActive(...) (bool, time.Time, time.Time, *bool, error)
```

Should return a struct.

#### 3.15 MEDIUM: Magic strings in resource factory

**File**: `pkg/resources/resources.go:24-35`

Resource names `"deployments"`, `"statefulsets"` used as magic strings in switch.

#### 3.16 MEDIUM: Errors not wrapping sentinels in resource factory

**File**: `pkg/resources/resources.go:43`

```go
fmt.Errorf("K8s config is required for deployments resource")
```

Should wrap a sentinel: `fmt.Errorf("deployments resource: %w", ErrK8sConfigRequired)`.

#### 3.17 MEDIUM: Silent failures in getter/setter adapter functions

**Files**: All resource adapter packages

```go
func getReplicas(item base.ResourceItem) *int32 {
    d, ok := item.(*deploymentItem)
    if !ok {
        return nil  // Silent failure
    }
```

#### 3.18 MEDIUM: `time.After` resource leak in VM handler

**File**: `pkg/gcp/resources/vm-instances/vm-instances.go:200`

`time.After` cannot be stopped/GC'd until it fires. Use `time.NewTimer` with `defer timer.Stop()`.

#### 3.19 MEDIUM: Type name typo `VMnstances`

**File**: `pkg/gcp/resources/vm-instances/types.go:11`

#### 3.20 MEDIUM: Mocks in production packages

**Files**: `pkg/k8s/utils/mocks.go`, `internal/controller/flow/service/mocks.go`

#### 3.21 MEDIUM: JSON tags on non-serializable interface fields

**File**: `pkg/k8s/utils/types.go:25-26`

```go
Client        kubernetes.Interface  `json:"client"`
DynamicClient dynamic.Interface     `json:"dynamicClient,omitempty"`
```

#### 3.22 MEDIUM: `matchExpressions` not implemented in GCP utils

**File**: `pkg/gcp/utils/utils.go:137` — `// TODO: Implement matchExpressions`

`In`, `NotIn`, `Exists`, `DoesNotExist` operators silently ignored.

#### 3.23 MEDIUM: `SetActivePeriod` side-effect mutation

**File**: `internal/utils/utils.go:54-93`

Mutates `status` pointer argument while also returning a period. Mixed responsibilities.

#### 3.24 MEDIUM: Error original context lost in `SetActivePeriod`

**File**: `internal/utils/utils.go:62-63`

```go
logger.Error().Err(err).Msg("unable to load noaction period")
return nil, ErrLoadNoactionPeriod
```

Original `err` logged but replaced with sentinel. Caller loses root cause.

#### 3.25 MEDIUM: Exported mutable global `DefaultRecorder`

**File**: `internal/metrics/metrics.go:48`

```go
var DefaultRecorder Recorder = &noopRecorder{}
```

Any package can reassign.

#### 3.26 LOW: `base/processor.go` naive pluralization

**File**: `pkg/k8s/resources/base/processor.go:127`

`kindPlural := p.strategy.GetKind() + "s"` — produces `"hpas"` for HPA.

#### 3.27 LOW: `base/strategies.go` missing `periodTypeUp` constant

**File**: `pkg/k8s/resources/base/strategies.go:72`

`case "up":` is a raw string while `periodTypeDown` is a constant.

#### 3.28 LOW: `SecretValidator` interface defined but never implemented

**File**: `pkg/k8s/utils/client/interfaces.go:47-49`

#### 3.29 LOW: `ClientFactory.CreateClient` returns concrete `*kubernetes.Clientset`

**File**: `pkg/k8s/utils/client/interfaces.go:36`

Should return `kubernetes.Interface` for testability.

---

### 4. Webhooks

#### 4.1 HIGH: K8s and GCP webhooks have NO validation

**Files**: `internal/webhook/v1alpha3/k8s_webhook.go`, `gcp_webhook.go`

```go
func SetupK8sWebhookWithManager(mgr ctrl.Manager) error {
    return ctrl.NewWebhookManagedBy(mgr, &kubecloudscalerv1alpha3.K8s{}).
        Complete()  // No .WithValidator()
}
```

Users can submit CRDs with empty project IDs, invalid periods, nonsensical replica counts.

#### 4.2 HIGH: Overnight recurring periods rejected by Flow webhook

**File**: `internal/webhook/v1alpha3/flow_webhook.go:238-239`

```go
if endTime.Before(startTime) {
    return 0, fmt.Errorf("end time is before start time")
}
```

A `23:00`-`06:00` period is rejected. Overnight windows should be supported.

#### 4.3 HIGH: No validation of resource references in flows

**File**: `internal/webhook/v1alpha3/flow_webhook.go:179-223`

`resource.Name` in flows is never validated against declared resources in `flow.Spec.Resources.K8s`/`.Gcp`. Typos pass validation but fail at runtime.

#### 4.4 MEDIUM: Both Recurring and Fixed can be set simultaneously

**File**: `flow_webhook.go:226-263`

`getPeriodDuration` silently uses `Recurring` and ignores `Fixed` when both are non-nil.

#### 4.5 LOW: Unused logger variables with dishonest `nolint:unused`

**Files**: `gcp_webhook.go:29-30`, `k8s_webhook.go:29-30`

```go
//nolint:unused // Variable is used for logging in webhook operations
var gcplog = logf.Log.WithName("gcp-resource")  // Never referenced
```

---

### 5. Composition Root (`cmd/main.go`)

#### 5.1 HIGH: No constructor DI for K8s/GCP reconcilers

**File**: `cmd/main.go:257-272`

```go
&k8sController.ScalerReconciler{Client: ..., Scheme: ..., Logger: ...}
```

Exported fields directly set. Flow controller correctly uses `NewFlowReconciler()`.

#### 5.2 HIGH: `metrics.GetRecorder()` is a service locator

Controllers call `metrics.GetRecorder()` internally — a textbook service locator anti-pattern. The `Recorder` interface should be injected via constructor.

#### 5.3 MEDIUM: Package-level mutable `logger` variable

**File**: `cmd/main.go:58-64`

`logger`, `logFormat`, `logLevel` should be local to `main()`.

#### 5.4 LOW: `ENABLE_WEBHOOKS` env read 3 times

**File**: `cmd/main.go:280-297`

Read once into a local variable.

#### 5.5 LOW: Error message typo in metrics cert watcher

**File**: `cmd/main.go:222`

```go
setupLog.Error(err, "to initialize metrics certificate watcher", "error", err)
```

Missing verb ("**Failed** to initialize...") and `err` passed twice.

---

### 6. Linter Configuration

#### 6.1 HIGH: Wrong Go version

**File**: `.golangci.yml:5`

```yaml
run:
  go: "1.21"  # Project uses Go 1.25.1
```

Linter won't flag issues with Go 1.22+ features.

#### 6.2 HIGH: Tests completely excluded from linting

**File**: `.golangci.yml:6`

```yaml
run:
  tests: false
```

`errcheck`, `govet`, `staticcheck` never run on test files.

#### 6.3 MEDIUM: `depguard` deny messages reference wrong project

**File**: `.golangci.yml:58-61`

```yaml
desc: "Use github.com/AgicapTech/sre-kernel-go-lib for logging"
```

Should reference `zerolog`.

#### 6.4 MEDIUM: Missing `errorlint` linter

The project mandates `errors.Is`/`errors.As` but `errorlint` is not enabled to enforce this.

#### 6.5 MEDIUM: Makefile — no coverage threshold enforcement

CLAUDE.md mandates 80% minimum. Neither `make test` nor `make test-coverage` fails on < 80%.

#### 6.6 MEDIUM: Makefile `GOLANGCI_LINT_VERSION` mismatch

**File**: `Makefile:234` — `v2.11.0` vs CLAUDE.md documented `v2.5.0`.

#### 6.7 LOW: Makefile comment copy-paste errors

`helmify` and `gen-crd-docs` targets both say "Download golangci-lint locally if necessary."

#### 6.8 LOW: `funlen.lines` comment says 100, value is 200

**File**: `.golangci.yml:74`

---

### 7. Test Quality

#### 7.1 CRITICAL: Non-deterministic tests that can never fail

5+ tests use `if err != nil { ... } else { ... }` — they pass regardless of outcome.

**Affected files**:
- `k8s/scaler_controller_test.go:146-155`
- `gcp/scaler_controller_test.go:138-147`
- `gcp/compatibility_test.go:64-68, 114, 191, 251-256`
- `k8s+gcp auth_handler_test.go`
- `k8s+gcp period_handler_test.go`

#### 7.2 CRITICAL: Zero RecoverableError test coverage

Production returns `RecoverableError` in 4 handler paths. None tested:
- `fetch_handler.go` (transient Get failure)
- `finalizer_handler.go` (Update failure)
- `status_handler.go` (Update failure)
- `status_handler.go` (Status update failure)

#### 7.3 CRITICAL: Three critical packages completely untested

| Package | Risk |
|---------|------|
| `pkg/resources/resources.go` (factory) | Routing bug affects all resources |
| `pkg/k8s/resources/base/processor.go` | Foundation for all K8s scaling |
| `pkg/k8s/resources/base/strategies.go` | Core scaling decision logic |

#### 7.4 CRITICAL: Stub test files providing false confidence

- `internal/webhook/v1alpha3/gcp_webhook_test.go` — all TODOs, zero assertions
- `internal/webhook/v1alpha3/k8s_webhook_test.go` — all TODOs, zero assertions
- `cmd/main_test.go` — 7 assertions of `Expect(true).To(BeTrue())`

#### 7.5 CRITICAL: E2E tests only verify pod starts

**File**: `test/e2e/e2e_test.go`

Zero reconciliation, scaling, or status verification. No CRDs created in tests.

#### 7.6 HIGH: Flow tests use raw `testing.T` instead of Ginkgo

All 5 files in `internal/controller/flow/service/` violate mandatory Ginkgo BDD requirement. Also white-box testing in production package (`package service` not `package service_test`).

#### 7.7 HIGH: No CRD conversion round-trip tests

CLAUDE.md mandates bidirectional conversion tests. None exist.

#### 7.8 HIGH: No period `IsActive` / activation tests

Period evaluation is core business logic. Completely untested.

#### 7.9 HIGH: Missing benchmarks for critical paths

Only annotation/namespace benchmarks exist. Missing: period evaluation, resource processing, handler chain execution.

#### 7.10 MEDIUM: Performance tests that test nothing

5 handler tests have `It("should complete in under 100ms")` that discard the result and make no timing assertion.

#### 7.11 MEDIUM: No table-driven tests anywhere

Zero table-driven tests across 58 test files. Candidates: `validResourceList`, period types, resource kinds, conversion scenarios.

#### 7.12 MEDIUM: No `RequeueAfter` value assertions

Only one test checks `RequeueAfter > 0`. None verify exact durations.

#### 7.13 MEDIUM: `os.Setenv` in tests without cleanup

**File**: `pkg/k8s/utils/client/k8s_test.go:57-62`

Manipulates process-global state without `t.Setenv()` or `DeferCleanup`.

#### 7.14 LOW: Global logger in resource tests

5 test files use `&log.Logger` instead of `zerolog.Nop()`.

#### 7.15 LOW: White-box testing in 6 packages

`annotation_manager_test.go`, `namespace_manager_test.go`, `integration_test.go`, `vm_instances_test.go`, `utils_test.go`, all Flow service tests — use `package X` not `package X_test`.

---

## Recommended Execution Order

| Priority | Work Item | Effort | Impact |
|----------|-----------|--------|--------|
| **P0-1** | Fix `omitempty` bool defaults | S | Users can't set features to false |
| **P0-2** | Fix GCP client leak + context | S | Resource leak in production |
| **P0-3** | Fix race condition with `sync.Once` | XS | Concurrent crash |
| **P0-4** | Add mutex to namespace cache | XS | Data race |
| **P0-5** | Store lost fields in conversion annotations | M | v1alpha1 data loss |
| **P1-1** | Add K8s/GCP webhook validators | M | Invalid CRDs accepted |
| **P1-2** | Fix `.golangci.yml` (Go version, test linting) | XS | Tests never linted |
| **P1-3** | Extract shared K8s/GCP code | L | ~40% code duplication |
| **P1-4** | Add constructors + DI for reconcilers | S | Service locator anti-pattern |
| **P1-5** | Refactor Flow to Chain of Responsibility | L | Architecture violation |
| **P1-6** | Replace `interface{}` with typed params | M | Runtime type assertions |
| **T1** | Fix non-deterministic tests | M | Tests that can never fail |
| **T2-T5** | Write tests for untested critical packages | L | Core logic untested |
| **T6** | Add CRD conversion round-trip tests | M | Data loss undetectable |
| **P2-**** | Remaining P2 items | M-L | Quality & consistency |
| **T7-T14** | Remaining test improvements | L | Coverage & confidence |

---

## Appendix: File Index

<details>
<summary>All files reviewed (click to expand)</summary>

### API Layer (22 files)
- `api/common/doc.go`
- `api/common/periods_type.go`
- `api/common/resources_type.go`
- `api/v1alpha1/doc.go`, `gcp_conversion.go`, `gcp_types.go`, `groupversion_info.go`, `k8s_conversion.go`, `k8s_types.go`
- `api/v1alpha2/doc.go`, `gcp_conversion.go`, `gcp_types.go`, `groupversion_info.go`, `k8s_conversion.go`, `k8s_types.go`
- `api/v1alpha3/doc.go`, `flow_types.go`, `gcp_conversion.go`, `gcp_types.go`, `groupversion_info.go`, `k8s_conversion.go`, `k8s_types.go`

### Controller Layer (32 files)
- `internal/controller/k8s/scaler_controller.go`, `service/context.go`, `service/errors.go`, `service/interfaces.go`
- `internal/controller/k8s/service/handlers/` — all 6 handlers
- `internal/controller/gcp/scaler_controller.go`, `service/chain.go`, `service/context.go`, `service/errors.go`, `service/interfaces.go`
- `internal/controller/gcp/service/handlers/` — all 6 handlers
- `internal/controller/flow/flow_controller.go`, `interfaces.go`, `types/types.go`
- `internal/controller/flow/service/` — all 7 service files

### pkg/ Layer (56 files)
- `pkg/period/` — 4 source files
- `pkg/resources/` — 4 source files
- `pkg/k8s/resources/base/` — 2 files
- `pkg/k8s/resources/deployments/` — 4 source files
- `pkg/k8s/resources/statefulsets/` — 4 source files
- `pkg/k8s/resources/cronjobs/` — 5 source files
- `pkg/k8s/resources/hpa/` — 3 source files
- `pkg/k8s/resources/github_autoscalingrunnersets/` — 4 source files
- `pkg/k8s/utils/` — 8 source files
- `pkg/k8s/utils/client/` — 6 source files
- `pkg/gcp/resources/vm-instances/` — 3 source files
- `pkg/gcp/utils/` — 3 source files
- `pkg/gcp/utils/client/` — 1 source file

### Internal Utilities (5 files)
- `internal/utils/consts.go`, `utils.go`, `vars.go`
- `internal/config/namespace.go`
- `internal/metrics/metrics.go`

### Webhooks (3 source files)
- `internal/webhook/v1alpha3/flow_webhook.go`, `gcp_webhook.go`, `k8s_webhook.go`

### Composition Root (1 file)
- `cmd/main.go`

### Configuration (2 files)
- `.golangci.yml`
- `Makefile`

### Test Files (58 files)
- All `*_test.go` and `suite_test.go` files across the project
- `test/utils/test_helpers.go`, `test/utils/utils.go`
- `test/e2e/e2e_suite_test.go`, `test/e2e/e2e_test.go`

</details>
