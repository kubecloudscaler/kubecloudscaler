# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

KubeCloudScaler is a Kubernetes operator that scales cloud resources up or down based on time periods using custom CRDs. It supports:
- **K8s resources**: Deployments, StatefulSets, CronJobs, HPAs, GitHub AutoScalingRunnerSets
- **GCP resources**: Compute Engine VM instances

## Build and Development Commands

```bash
# Build
make build                    # Build manager binary to bin/kubecloudscaler

# Run locally (generates webhook certs automatically)
make run                      # Run controller with debug logging

# Tests
make test                     # Run unit tests with coverage
make test-coverage            # Run tests with HTML coverage report
make test-bench               # Run benchmarks
go test ./internal/controller/gcp/service/handlers/...  # Run specific handler tests
go test ./... -run TestFetch  # Run single test by name

# Lint
make lint                     # Run golangci-lint
make lint-fix                 # Run with auto-fix

# Code generation (run after modifying CRD types)
make generate                 # Generate DeepCopy methods
make manifests                # Generate CRDs, RBAC, webhook configs

# E2E tests (requires Kind)
make test-e2e                 # Sets up Kind cluster, runs e2e tests, tears down
```

## Architecture

### CRD Types (api/)

Three CRD kinds, each with multiple API versions (v1alpha1, v1alpha2, v1alpha3 storage version):
- `K8s` - Scales Kubernetes workloads
- `Gcp` - Scales GCP resources
- `Flow` - Orchestrates scaling workflows

### Controller Pattern

Both K8s and GCP controllers use the **Chain of Responsibility pattern** with discrete handlers:

```
Reconcile() → HandlerChain.Execute()
    ├── FetchHandler      - Fetch scaler resource
    ├── FinalizerHandler  - Manage finalizer lifecycle
    ├── AuthHandler       - Setup K8s/GCP authentication
    ├── PeriodHandler     - Validate time periods
    ├── ScalingHandler    - Scale resources
    └── StatusHandler     - Update status
```

Handlers share state via `ReconciliationContext` and implement:
```go
type Handler interface {
    Handle(ctx context.Context, req *ReconciliationContext) (*ReconciliationResult, error)
}
```

### Directory Structure

- `cmd/` - Application entrypoint
- `api/` - CRD type definitions (v1alpha1, v1alpha2, v1alpha3)
- `internal/controller/` - Controller implementations with handler chains
  - `gcp/service/handlers/` - GCP controller handlers
  - `k8s/service/handlers/` - K8s controller handlers
  - `flow/service/` - Flow controller services
- `internal/webhook/` - Admission webhooks
- `pkg/` - Shared packages (period logic, k8s/gcp resource utilities)
- `config/` - Kustomize manifests for CRDs, RBAC, webhooks

## Testing

Uses **Ginkgo/Gomega** for BDD-style tests. Tests use `envtest` for Kubernetes API simulation.

Key test locations:
- `internal/controller/*/service/handlers/*_test.go` - Handler unit tests
- `internal/controller/*_test.go` - Controller integration tests
- `pkg/**/suite_test.go` - Package test suites

## Code Style Requirements

- TDD mandatory for service layer business logic (`internal/controller/*/service/`)
- 80% minimum test coverage (CI enforced)
- Use `zerolog` for structured logging
- Wrap errors with context: `fmt.Errorf("context: %w", err)`
- Interfaces for all dependencies (dependency injection via constructors)
- Cyclomatic complexity >10 requires justification

## Key Dependencies

- `sigs.k8s.io/controller-runtime` - Kubernetes controller framework
- `cloud.google.com/go/compute` - GCP Compute client
- `github.com/rs/zerolog` - Structured logging
- `github.com/onsi/ginkgo/v2` + `github.com/onsi/gomega` - Testing framework
