# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
make build                    # Produces bin/kubecloudscaler
make manifests                # Regenerate CRDs/RBAC/webhooks after API changes
make generate                 # Regenerate DeepCopy code after API type changes

# Test
make test                     # All unit tests (excludes e2e), uses envtest
go test ./pkg/period/... -v   # Single package
go test ./internal/... -run TestName -v  # Single test by name
make test-coverage            # HTML coverage report
make test-e2e                 # End-to-end tests (spins up Kind cluster)

# Lint / Format
make lint                     # golangci-lint
make lint-fix                 # With auto-fixes
make fmt && make vet          # Format + vet

# Local run (with webhooks)
make generate-certs           # Generate certs into tmp/k8s-webhook-server/
make run                      # Run controller locally against current kubeconfig
```

## Architecture

**kubecloudscaler** is a Kubernetes operator that scales cloud resources based on configured time periods. It has three controllers.

### Three Controllers

**K8s Scaler** (`internal/controller/k8s/`) — uses Chain of Responsibility:
1. `FetchHandler` → fetches the `K8s` CR
2. `FinalizerHandler` → manages finalizers
3. `AuthHandler` → builds `kubernetes.Interface` + `dynamic.Interface` clients (local or remote via secret)
4. `PeriodHandler` → validates timing, determines active period
5. `ScalingHandler` → scales Deployments, StatefulSets, CronJobs, HPAs, GitHub ARS
6. `StatusHandler` → writes `SuccessResults`/`FailedResults` back to status

**GCP Scaler** (`internal/controller/gcp/`) — same chain pattern, manages GCP VM instances and Cloud SQL.

**Flow Controller** (`internal/controller/flow/`) — service-oriented (no handler chain). Reads a `Flow` CR and creates/owns `K8s` and `Gcp` scaler objects. Each resource in a flow can have `startTimeDelay`/`endTimeDelay` offsets that shift the period times. Uses owner references so scaler objects are deleted when the Flow is deleted.

### Key Packages

**`pkg/period/`** — Period activation logic. `period.New(scalerPeriod)` returns a `Period` with `IsActive`, `Type` (up/down), `Hash` (SHA1 for change detection), and `Once` flag. Supports recurring (daily/weekly with timezone) and fixed (one-time date range) period types.

**`pkg/resources/`** — Factory that returns typed resource handlers via `resources.NewResource(name, config)`. Each handler implements `Scale()`, `Fetch()`, `Restore()`. Current types: `deployments`, `statefulsets`, `cronjobs`, `github-ars`, `hpa`, `vm-instances`.

**`internal/utils/`** — `ValidatePeriod()` is the central function used by both K8s and GCP `PeriodHandler`s to check if a period is active, handle "once" periods (via SHA comparison), and return typed sentinel errors (`ErrPeriodNotActive`, `ErrRunOncePeriod`).

### Error Handling Pattern

All handlers return one of three outcomes:
- `service.NewCriticalError(err)` — stops chain, no requeue (invalid config, not found)
- `service.NewRecoverableError(err)` — stops chain, requeues after `ctx.RequeueAfter` (or `utils.ReconcileErrorDuration` = 10m as default)
- `nil` — success, call `h.next.Execute(ctx)`

The controller checks with `service.IsCriticalError(err)` / `service.IsRecoverableError(err)`.

**Conflict errors** (optimistic concurrency / HTTP 409) in status updates use `retry.RetryOnConflict(retry.DefaultRetry, func() error { /* re-fetch + re-apply + update */ })` — do not simply requeue.

### ReconciliationContext

Shared mutable state threaded through the handler chain. Key fields set by each handler:
- Controller sets: `Ctx`, `Request`, `Client`, `Logger`
- FetchHandler sets: `Scaler`
- AuthHandler sets: `K8sClient`, `DynamicClient`, `Secret`
- PeriodHandler sets: `Period`, `ResourceConfig`; may set `SkipRemaining = true` (no active period)
- ScalingHandler sets: `SuccessResults`, `FailedResults`
- Any handler may set: `RequeueAfter` (first write wins), `SkipRemaining`

### API Versions

`v1alpha3` is the storage version. `v1alpha1` and `v1alpha2` exist with conversion webhooks. All new work goes in `api/v1alpha3/`. Shared types (period definitions, status structs) live in `api/common/`.

### Linting Notes

- Line length limit: 140 chars
- Cyclomatic complexity limit: 12 (gocyclo), 20 (gocognit)
- Magic numbers flagged (mnd) — define constants
- Test files are exempt from most complexity/style rules
- `//nolint:xxx // reason` required when suppressing linter
