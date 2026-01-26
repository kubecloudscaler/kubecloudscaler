# agents.md

This file provides comprehensive guidance to AI agents (including Claude Code) when working with code in this repository. All agents MUST adhere to the principles and constraints defined in this document.

## Project Overview

KubeCloudScaler is a Kubernetes operator that scales cloud resources up or down based on time periods using custom CRDs. It supports:
- **K8s resources**: Deployments, StatefulSets, CronJobs, HPAs, GitHub AutoScalingRunnerSets
- **GCP resources**: Compute Engine VM instances

## Constitutional Principles

The following principles from [.specify/memory/constitution.md](.specify/memory/constitution.md) are MANDATORY and supersede all other practices:

### I. Clean Architecture & Repository Pattern
All code MUST follow Clean Architecture principles with clear layer separation:
- `cmd/` - Application entrypoints
- `internal/controller/` - Controllers (reconciliation logic)
- `internal/controller/*/service/` - Business logic and use cases (service layer within controllers)
- `internal/repository/` - Data access layer
- `internal/model/` - Domain models
- `internal/config/` - Configuration management
- `pkg/` - Shared utilities and packages (exposed externally)
- `api/` - CRD resource definitions

The Repository Pattern MUST be used to separate data access logic from business logic. All public functions MUST interact with interfaces, not concrete types.

### II. Interface-Driven Development & Dependency Injection
- All dependencies MUST be defined as interfaces
- Dependency injection MUST be performed via constructors (no global state)
- Interfaces MUST be small and purpose-specific
- High-level modules MUST NOT depend on low-level modules (Dependency Inversion Principle)

### III. Test-Driven Development (TDD)
TDD is MANDATORY for all business logic in the `internal/controller/*/service/` layer:
1. Tests written → User approved → Tests fail → Then implement
2. Red-Green-Refactor cycle MUST be strictly enforced
3. All exported functions MUST have test coverage with behavioral checks
4. Minimum test coverage of 80% MUST be maintained and enforced via CI/CD

### IV. Observability & Structured Logging
- Structured JSON logging via `zerolog` is REQUIRED
- Prometheus metrics MUST be exposed for monitoring
- Logging MUST use appropriate levels (info, warn, error)
- Minimal cardinality in labels and traces to keep observability overhead low

### V. Go Conventions & Code Style
- Code MUST follow idiomatic Go conventions (Effective Go and Google's Go Style Guide)
- Use tabs for indentation
- Named functions over long anonymous ones
- Small composable functions with single responsibility
- Limit line length to 80 characters where practical
- Use `gofmt` or `goimports` for formatting
- Enforce naming consistency with `golangci-lint`

### VI. Error Handling & Resource Management
- All errors MUST be handled explicitly using wrapped errors: `fmt.Errorf("context: %w", err)`
- No panics in library code - return errors instead
- Resources MUST be closed with defer statements
- Context propagation MUST be used for request-scoped values, deadlines, and cancellations

### VII. Kubernetes Operator Patterns
- MUST follow Kubernetes operator patterns using controller-runtime
- MUST use `context.Context` for request-scoped values and cancellation
- MUST support graceful shutdown and resource cleanup
- MUST be thread-safe when using goroutines
- MUST handle cluster-wide operations safely and respect RBAC permissions

### VIII. Security & Compliance
- Secrets MUST NOT be exposed in logs or error messages
- MUST follow Kubernetes security best practices
- MUST support namespace isolation for multi-tenant scenarios
- MUST be backward compatible with existing CRD versions
- MUST support multiple CRD API versions (v1alpha1, v1alpha2, v1alpha3)

## Architecture Constraints

### Technical Constraints
- MUST use Go 1.22.0+
- MUST use controller-runtime for Kubernetes operator functionality
- MUST use `cloud.google.com/go` for GCP client libraries
- MUST use `github.com/rs/zerolog` for structured logging
- MUST use `github.com/onsi/ginkgo/v2` and `github.com/onsi/gomega` for BDD testing
- MUST use `golangci-lint` for linting and code quality
- MUST minimize allocations and avoid premature optimization
- MUST instrument key areas (Kubernetes API calls, GCP API calls, heavy computation)

### Business Constraints
- MUST be cost-effective with minimal resource footprint:
  - Memory requests <100MB
  - CPU requests <100m
- MUST support multiple CRD API versions for backward compatibility
- MUST handle cluster-wide operations safely
- MUST respect RBAC permissions

### Code Complexity
- Cyclomatic complexity >10 requires justification
- New architectural patterns require rationale and approval

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

## Testing Strategy

### Framework and Organization
- Use **Ginkgo/Gomega** for BDD-style tests
- Tests use `envtest` for Kubernetes API simulation
- Organize tests under `test/unit/` and `test/integration/`
- Separate fast unit tests from slower integration and E2E tests

### Key Test Locations
- `internal/controller/*/service/handlers/*_test.go` - Handler unit tests
- `internal/controller/*_test.go` - Controller integration tests
- `pkg/**/suite_test.go` - Package test suites

### Testing Requirements
- Mock external services (Kubernetes API, GCP APIs) using interfaces
- Include table-driven tests for functions with many input variants
- Write benchmarks to track performance regressions
- Maintain minimum 80% test coverage (CI enforced)

## Development Workflow

### TDD Workflow for Service Layer
When implementing business logic in `internal/controller/*/service/`:

1. **Write Tests First**: Create test cases that describe expected behavior
2. **Get User Approval**: Confirm tests match requirements
3. **Run Tests (Red)**: Verify tests fail as expected
4. **Implement Code (Green)**: Write minimal code to pass tests
5. **Refactor**: Improve code quality while keeping tests green
6. **Verify Coverage**: Ensure 80%+ coverage maintained

### Git Workflow
- Use semantic versioning for releases
- Follow conventional commit messages
- Maintain a `CHANGELOG.md` for tracking changes
- Use feature branches for development
- Require code review before merging to main branch

### Code Review Requirements
All PRs MUST verify:
- Compliance with constitutional principles
- Complexity justification for cyclomatic complexity >10
- Tests included for new functionality
- Documentation updated for user-facing changes
- 80% minimum test coverage maintained

## Key Dependencies

- `sigs.k8s.io/controller-runtime` - Kubernetes controller framework
- `cloud.google.com/go/compute` - GCP Compute client
- `github.com/rs/zerolog` - Structured logging
- `github.com/onsi/ginkgo/v2` + `github.com/onsi/gomega` - Testing framework

## Agent Guidelines

### Before Making Changes
1. Read relevant code files first - NEVER propose changes without reading
2. Understand existing patterns and architecture
3. Verify compliance with constitutional principles
4. Plan changes using TodoWrite tool for multi-step tasks

### When Writing Code
1. Follow TDD for service layer (`internal/controller/*/service/`)
2. Write tests BEFORE implementation
3. Use interfaces for all dependencies
4. Inject dependencies via constructors
5. Wrap errors with context: `fmt.Errorf("context: %w", err)`
6. Use `zerolog` for structured logging
7. Keep cyclomatic complexity ≤10

### Security Considerations
- Avoid command injection, XSS, SQL injection, and OWASP top 10 vulnerabilities
- Never expose secrets in logs or error messages
- Handle all errors explicitly - no silent failures
- Use context for cancellation and timeouts

### Anti-Patterns to Avoid
- Over-engineering solutions beyond requirements
- Adding features not explicitly requested
- Creating abstractions for one-time operations
- Premature optimization
- Global state or singleton patterns
- Backwards-compatibility hacks for unused code

### After Implementation
1. Run tests: `make test`
2. Verify coverage: `make test-coverage`
3. Run linter: `make lint`
4. Generate code if CRDs modified: `make generate && make manifests`
5. Update documentation for user-facing changes

## Governance

This document synthesizes [CLAUDE.md](CLAUDE.md) and [.specify/memory/constitution.md](.specify/memory/constitution.md). The constitution supersedes all other practices. For amendments to constitutional principles, see the Governance section in the constitution.

**Last Updated**: 2026-01-26
