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
- `internal/controller/*/service/handlers/` - Chain of Responsibility handlers
- `internal/repository/` - Data access layer
- `internal/model/` - Domain models
- `internal/config/` - Configuration management
- `internal/webhook/` - Admission webhooks (validating and mutating)
- `pkg/` - Shared utilities and packages (exposed externally)
- `api/` - CRD resource definitions (multi-version: v1alpha1, v1alpha2, v1alpha3)
- `api/common/` - Shared types across API versions (periods, resources, status)

The Repository Pattern MUST be used to separate data access logic from business logic. All public functions MUST interact with interfaces, not concrete types.

### II. Interface-Driven Development & Dependency Injection
- All dependencies MUST be defined as interfaces
- Dependency injection MUST be performed via constructors (no global state)
- Interfaces MUST be small and purpose-specific (Interface Segregation Principle)
- Interfaces MUST be defined where they are consumed, not where they are implemented
- High-level modules MUST NOT depend on low-level modules (Dependency Inversion Principle)
- Avoid `interface{}` / `any` when a concrete type or generic constraint is more appropriate

### III. Test-Driven Development (TDD)
TDD is MANDATORY for all business logic in the `internal/controller/*/service/` layer:
1. Tests written → User approved → Tests fail → Then implement
2. Red-Green-Refactor cycle MUST be strictly enforced
3. All exported functions MUST have test coverage with behavioral checks
4. Minimum test coverage of 80% MUST be maintained and enforced via CI/CD

### IV. Observability & Structured Logging
- Structured JSON logging via `zerolog` is REQUIRED for application code
- `zap` is used for controller-runtime internal logging (via `ctrl.SetLogger`)
- Prometheus metrics MUST be exposed for monitoring via `prometheus/client_golang`
- Logging MUST use appropriate levels:
  - `debug` - Verbose development-time information
  - `info` - Normal operational events (reconcile success, scaling actions)
  - `warn` - Recoverable issues (temporary API failures, retries)
  - `error` - Unrecoverable failures requiring attention
- Minimal cardinality in labels and traces to keep observability overhead low
- NEVER log secrets, tokens, or credentials at any log level

### V. Go Conventions & Code Style (Go 1.25+)
- Code MUST follow idiomatic Go conventions (Effective Go, Google's Go Style Guide, Go Proverbs)
- Use tabs for indentation
- Named functions over long anonymous ones
- Small composable functions with single responsibility
- Line length limit: 140 characters (enforced by `lll` linter)
- Use `gofmt` and `goimports` for formatting
- Enforce code quality with `golangci-lint` v2

#### Modern Go Features (1.22+)
- Use `range` over integers: `for i := range n` instead of `for i := 0; i < n; i++`
- Use `range` over functions (iterators) for custom iteration patterns
- Use standard library packages: `slices`, `maps`, `cmp` instead of hand-rolling utilities
- Use `errors.Join` for combining multiple errors
- Use generics where they reduce duplication without sacrificing readability
- Use `context.AfterFunc` for cleanup tied to context cancellation
- Use `log/slog` only if integrating with standard library consumers; prefer `zerolog` for this project

#### Naming Conventions
- CamelCase for exports, camelCase for private
- Acronyms in uppercase: `HTTP`, `URL`, `ID`, `GCP`, `CRD`, `RBAC`
- Receiver names: short (1-2 chars), consistent within a type
- Package names: short, lowercase, no underscores
- Test helpers: prefix with `setup` or `new` (e.g., `newFakeClient`, `setupTestEnv`)

### VI. Error Handling & Resource Management
- All errors MUST be handled explicitly using wrapped errors: `fmt.Errorf("context: %w", err)`
- Use sentinel errors for well-known conditions: `var ErrNotFound = errors.New("not found")`
- Use custom error types for classification (see `CriticalError`, `RecoverableError` in service layer)
- No panics in library code - return errors instead
- Resources MUST be closed with `defer` statements
- Context propagation MUST be used for request-scoped values, deadlines, and cancellations
- Use `errors.Is` and `errors.As` for error inspection, never type assertions on error
- Use `errors.Join` when aggregating errors from multiple operations

### VII. Kubernetes Operator Patterns
- MUST follow Kubernetes operator patterns using controller-runtime v0.23+
- MUST use `context.Context` for request-scoped values and cancellation
- MUST support graceful shutdown and resource cleanup
- MUST be thread-safe when using goroutines
- MUST handle cluster-wide operations safely and respect RBAC permissions

#### Chain of Responsibility Pattern (MANDATORY)
All controllers MUST implement reconciliation logic using the **Chain of Responsibility** pattern. The `Reconcile()` method MUST NOT contain business logic directly — it MUST delegate to a handler chain.

**Structure**:
```go
// Each handler implements this interface
type Handler interface {
    Execute(ctx *ReconciliationContext) error
    SetNext(next Handler)
}
```

**Rules**:
- Each handler is responsible for **one single reconciliation step** (fetch, finalizer, auth, period evaluation, scaling, status update)
- Handlers share state via a `ReconciliationContext` struct, not function parameters
- A handler calls `next.Execute(ctx)` to continue the chain (after checking `next != nil`)
- A handler returns `CriticalError` to stop the chain without requeue
- A handler returns `RecoverableError` to stop the chain with requeue
- A handler returns `nil` to continue to the next handler
- The handler chain order is defined in the controller and MUST NOT be modified without review
- New reconciliation steps MUST be implemented as new handlers, not added inline to existing ones
- Handler constructors accept dependencies as interfaces for testability

**Standard chain order**:
```
FetchHandler → FinalizerHandler → AuthHandler → PeriodHandler → ScalingHandler → StatusHandler
```

#### Reconciliation Best Practices
- Reconcilers MUST be **idempotent** - running the same reconciliation multiple times produces the same result
- Reconcilers MUST be **level-triggered**, not edge-triggered - react to current state, not events
- MUST distinguish error types for proper requeue behavior:
  - **Critical errors**: Stop immediately, do not requeue (e.g., resource not found after deletion)
  - **Recoverable errors**: Requeue with backoff (e.g., transient API failures)
- Use `ctrl.Result{RequeueAfter: duration}` for delayed requeue, not `Requeue: true`
- Keep reconcile loops fast - offload heavy work to goroutines with proper context cancellation
- Use `client.IgnoreNotFound(err)` for resources that may be deleted during reconciliation
- Log reconciliation results at appropriate levels (info for success, error for failures)

#### Status Management
- Use the **status subresource** (`+kubebuilder:subresource:status`) for all CRDs
- Update status independently from spec changes
- Use status conditions following the [Kubernetes API conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties)
- NEVER update status and spec in the same API call

#### Finalizer Patterns
- Use finalizers for cleanup of external resources (GCP VMs, external state)
- Add finalizer during first reconciliation, remove after cleanup
- Check for deletion timestamp before processing: `if !obj.DeletionTimestamp.IsZero()`
- Ensure finalizer cleanup is idempotent

#### Watch & Predicate Patterns
- Use event predicates to filter unnecessary reconciliations
- Use `IgnoreDeletionPredicate` to skip deletion events when not needed
- Use `GenerationChangedPredicate` to only reconcile on spec changes (skip status-only updates)
- Watch owned resources with `Owns()` for automatic reconciliation on child changes
- Use `EnqueueRequestForOwner` for custom owner references

#### Server-Side Apply (SSA)
- Prefer SSA (`client.Apply`) over Update for creating/updating owned resources
- SSA provides conflict detection and field ownership tracking
- Use a unique field manager name per controller

#### Multi-Version CRD Management
- v1alpha3 is the **storage version** (`+kubebuilder:storageversion`)
- Implement conversion webhooks for v1alpha1 ↔ v1alpha3 and v1alpha2 ↔ v1alpha3
- All conversions MUST be lossless (round-trip safe)
- Test conversion functions with bidirectional conversion tests

### VIII. Security & Compliance
- Secrets MUST NOT be exposed in logs or error messages
- MUST follow Kubernetes security best practices
- MUST support namespace isolation for multi-tenant scenarios
- MUST be backward compatible with existing CRD versions
- HTTP/2 disabled by default for CVE mitigation (`NextProtos = []string{"http/1.1"}`)
- TLS certificate watchers for hot-reload without pod restart
- Metrics endpoint secured with authentication and authorization (`FilterProvider`)
- RBAC annotations MUST follow least privilege principle
- Use `+kubebuilder:rbac` markers to generate minimal RBAC roles

### IX. Performance & Resource Efficiency
- MUST be cost-effective with minimal resource footprint:
  - Memory requests <100MB
  - CPU requests <100m
- Minimize Kubernetes API calls: use informer caches, avoid redundant Gets
- Use `prealloc` for slice pre-allocation where size is known
- Avoid unnecessary type conversions (enforced by `unconvert` linter)
- Profile with `pprof` when investigating performance issues
- Benchmark critical paths with `go test -bench`

## Architecture

### CRD Types (api/)

Three CRD kinds, each with multiple API versions (v1alpha1, v1alpha2, v1alpha3 storage version):
- `K8s` - Scales Kubernetes workloads (Deployments, StatefulSets, CronJobs, HPAs)
- `Gcp` - Scales GCP resources (Compute Engine VM instances)
- `Flow` - Orchestrates scaling workflows across K8s and GCP resources

All CRDs are **cluster-scoped** (`+kubebuilder:resource:scope=Cluster`).

### Controller Pattern: Chain of Responsibility

Both K8s and GCP controllers use the **Chain of Responsibility pattern** with discrete handlers:

```
Reconcile() → HandlerChain.Execute()
    ├── FetchHandler      - Fetch scaler resource from API server
    ├── FinalizerHandler  - Manage finalizer lifecycle (add/remove)
    ├── AuthHandler       - Setup K8s/GCP authentication via secrets
    ├── PeriodHandler     - Validate and evaluate time periods
    ├── ScalingHandler    - Execute scaling actions on target resources
    └── StatusHandler     - Update CRD status subresource
```

Handlers share state via `ReconciliationContext` and implement:
```go
type Handler interface {
    Execute(ctx *ReconciliationContext) error
    SetNext(next Handler)
}
```

Each handler:
- Processes one reconciliation step independently
- Modifies context for subsequent handlers
- Calls `next.Execute()` to continue the chain (checks `next != nil` first)
- Returns `CriticalError` to stop chain without requeue
- Returns `RecoverableError` to stop chain with requeue
- Returns `nil` to continue to next handler

### Flow Controller

The Flow controller orchestrates multi-resource scaling:
- `FlowProcessor` - Core workflow processing
- `FlowValidator` - Validates flow configuration
- `ResourceCreator` - Creates child K8s/Gcp resources
- `ResourceMapper` - Maps flow definitions to resource specs
- `StatusUpdater` - Aggregates status from child resources
- `TimeCalculator` - Computes timing delays for cascade scaling

### Directory Structure

```
├── cmd/                           # Application entrypoint
│   └── main.go                   # Manager setup, controller + webhook registration
├── api/                           # CRD type definitions
│   ├── common/                   # Shared types (periods, resources, status)
│   ├── v1alpha1/                 # First API version
│   ├── v1alpha2/                 # Second API version
│   └── v1alpha3/                 # Storage version + conversion webhooks
├── internal/
│   ├── controller/               # Controller implementations
│   │   ├── k8s/                 # K8s resource scaling controller
│   │   │   ├── scaler_controller.go
│   │   │   └── service/handlers/ # Chain of Responsibility handlers
│   │   ├── gcp/                 # GCP resource scaling controller
│   │   │   ├── scaler_controller.go
│   │   │   └── service/handlers/
│   │   └── flow/                # Flow orchestration controller
│   │       └── service/         # Flow business logic services
│   ├── webhook/v1alpha3/        # Admission webhooks
│   └── utils/                   # Internal utility functions
├── pkg/                           # Shared packages (period logic, resource utilities)
├── config/                        # Kustomize manifests (CRDs, RBAC, webhooks, manager)
├── helm/                          # Helm chart
├── test/e2e/                     # End-to-end tests
├── hack/                          # Build and development scripts
└── docs/                          # Documentation
```

## Build and Development Commands

```bash
# Build
make build                    # Build manager binary to bin/kubecloudscaler

# Run locally (generates webhook certs automatically)
make run                      # Run controller with debug logging

# Tests
make test                     # Run unit tests with coverage
make test-coverage            # Run tests with HTML coverage report + per-package analysis
make test-bench               # Run benchmarks
go test ./internal/controller/gcp/service/handlers/...  # Run specific handler tests
go test ./... -run TestFetch  # Run single test by name

# Lint
make lint                     # Run golangci-lint v2
make lint-fix                 # Run with auto-fix
make lint-config              # Verify linter configuration

# Code generation (run after modifying CRD types)
make generate                 # Generate DeepCopy methods
make manifests                # Generate CRDs, RBAC, webhook configs

# Documentation
make doc                      # Generate API documentation via api2md

# E2E tests (requires Kind)
make test-e2e                 # Sets up Kind cluster, runs e2e tests, tears down

# Deployment
make install                  # Install CRDs into cluster
make deploy                   # Deploy controller to cluster
make helm                     # Generate Helm chart from kustomize
```

## Testing Strategy

### Framework and Organization
- Use **Ginkgo v2 / Gomega** for BDD-style tests
- Tests use `envtest` for Kubernetes API simulation (via `setup-envtest`)
- Use `fake.NewClientBuilder()` for unit tests with fake Kubernetes clients
- Separate fast unit tests from slower integration and E2E tests

### Test Structure (Ginkgo BDD)
```go
var _ = Describe("FetchHandler", func() {
    var (
        handler  service.Handler
        reconCtx *service.ReconciliationContext
    )

    BeforeEach(func() {
        // Setup test fixtures
    })

    Context("when the scaler resource exists", func() {
        It("should fetch and set the scaler in context", func() {
            err := handler.Execute(reconCtx)
            Expect(err).ToNot(HaveOccurred())
            Expect(reconCtx.Scaler).ToNot(BeNil())
        })
    })

    Context("when the scaler resource is not found", func() {
        It("should return a critical error", func() {
            err := handler.Execute(reconCtx)
            Expect(err).To(HaveOccurred())
            Expect(service.IsCriticalError(err)).To(BeTrue())
        })
    })
})
```

### Key Test Patterns
- **Fake client**: `fake.NewClientBuilder().WithScheme(scheme).WithObjects(existingObjs...).Build()`
- **No-op logger**: `zerolog.Nop()` for silent test logging
- **Scheme registration**: Register API types before creating fake clients
- **Mock handlers**: Implement `Handler` interface with function fields for custom behavior
- **Performance assertions**: `Expect(duration).To(BeNumerically("<", maxDuration))`
- **Error type checks**: `Expect(service.IsCriticalError(err)).To(BeTrue())`

### Test File Locations
- `internal/controller/*/service/handlers/*_test.go` - Handler unit tests
- `internal/controller/*_test.go` - Controller integration tests
- `internal/controller/*/service/*_test.go` - Service layer tests
- `internal/webhook/v1alpha3/*_test.go` - Webhook tests
- `pkg/**/suite_test.go` - Package test suites
- `test/e2e/` - End-to-end tests with Kind cluster

### Testing Requirements
- Mock external services (Kubernetes API, GCP APIs) using interfaces
- Include table-driven tests for functions with many input variants
- Write benchmarks to track performance regressions
- Maintain minimum 80% test coverage (CI enforced)
- Test error classification (critical vs recoverable) explicitly
- Test handler chain order and next-handler invocation
- Test CRD conversion functions bidirectionally (v1alpha1 ↔ v1alpha3, v1alpha2 ↔ v1alpha3)

### envtest Best Practices
- Start envtest environment in `BeforeSuite`, stop in `AfterSuite`
- Use separate namespaces per test to avoid conflicts
- Clean up created resources in `AfterEach` or use unique names
- Set reasonable timeouts for Eventually/Consistently assertions
- Use `envtest.Environment.Config` for test client configuration

## Linting Configuration (golangci-lint v2)

### Enabled Linters (key ones)
| Linter | Purpose | Configuration |
|--------|---------|---------------|
| `errcheck` | Unchecked errors | Type assertions checked |
| `govet` | Official static analysis | - |
| `staticcheck` | Advanced static analysis | All checks enabled |
| `gosec` | Security analysis | Severity: medium |
| `gocyclo` | Cyclomatic complexity | Max: 12 |
| `gocognit` | Cognitive complexity | Max: 20 |
| `funlen` | Function length | Max: 200 lines, 50 statements |
| `revive` | Modern golint replacement | 20+ rules enabled |
| `depguard` | Import control | Blocks logrus, zap |
| `ginkgolinter` | Ginkgo-specific checks | - |
| `lll` | Line length | Max: 140 chars |
| `mnd` | Magic numbers | All checks, `make`/`strings.SplitN` excluded |
| `prealloc` | Slice pre-allocation | - |
| `bodyclose` | HTTP body closure | - |
| `noctx` | HTTP requests without context | - |

### Exclusions
- Tests (`_test.go`): Excluded from `gocyclo`, `errcheck`, `dupl`, `gosec`, `funlen`, `gocognit`, `mnd`
- Main/config files: Excluded from `mnd`
- Generated/vendor files: Excluded from `depguard`, `revive`
- `nolint` directives require explanation and specific linter name

### Code Complexity Thresholds
- Cyclomatic complexity >12 triggers `gocyclo`
- Cognitive complexity >20 triggers `gocognit`
- Functions >200 lines or >50 statements trigger `funlen`
- Duplicate blocks >100 tokens trigger `dupl`
- Any complexity override requires justification via `//nolint:linter // reason`

## Development Workflow

### TDD Workflow for Service Layer
When implementing business logic in `internal/controller/*/service/`:

1. **Write Tests First**: Create test cases that describe expected behavior using Ginkgo/Gomega
2. **Get User Approval**: Confirm tests match requirements
3. **Run Tests (Red)**: Verify tests fail as expected (`make test`)
4. **Implement Code (Green)**: Write minimal code to pass tests
5. **Refactor**: Improve code quality while keeping tests green
6. **Verify Coverage**: Ensure 80%+ coverage maintained (`make test-coverage`)
7. **Lint**: Run `make lint` to catch style and quality issues

### Adding a New Handler
1. Create handler in `internal/controller/*/service/handlers/`
2. Implement `Handler` interface (`Execute`, `SetNext`)
3. Write tests first (TDD) covering success, critical error, and recoverable error paths
4. Register handler in the chain within the controller's `buildChain()` method
5. Update RBAC markers if new API permissions are needed
6. Run `make manifests` if RBAC markers changed

### Adding a New CRD Field
1. Add field to v1alpha3 types in `api/v1alpha3/`
2. Update common types in `api/common/` if shared across kinds
3. Add kubebuilder markers for validation (`+kubebuilder:validation:*`)
4. Update conversion functions in `api/v1alpha3/*_conversion.go`
5. Run `make generate && make manifests`
6. Update webhook validation if applicable
7. Add tests for new field behavior and conversion

### Git Workflow
- Use semantic versioning for releases
- Follow conventional commit messages: `feat:`, `fix:`, `chore:`, `docs:`, `refactor:`, `test:`
- Maintain a `CHANGELOG.md` for tracking changes
- Use feature branches for development
- Require code review before merging to main branch

### Code Review Requirements
All PRs MUST verify:
- Compliance with constitutional principles
- Complexity justification for cyclomatic complexity >12
- Tests included for new functionality (TDD enforced for service layer)
- Documentation updated for user-facing changes
- 80% minimum test coverage maintained
- Linter passes (`make lint`)
- Generated code up to date (`make generate && make manifests`)

## Key Dependencies

| Dependency | Version | Purpose |
|-----------|---------|---------|
| `sigs.k8s.io/controller-runtime` | v0.23.0 | Kubernetes controller framework |
| `k8s.io/api` | v0.35.0 | Kubernetes API types |
| `k8s.io/apimachinery` | v0.35.0 | Kubernetes API machinery |
| `k8s.io/client-go` | v0.35.0 | Kubernetes client |
| `cloud.google.com/go/compute` | v1.54.0 | GCP Compute client |
| `google.golang.org/api` | v0.262.0 | Google API client |
| `github.com/rs/zerolog` | v1.34.0 | Structured logging |
| `github.com/onsi/ginkgo/v2` | v2.27.5 | BDD testing framework |
| `github.com/onsi/gomega` | v1.39.0 | Matcher library |
| `github.com/stretchr/testify` | v1.11.1 | Test assertions |
| `github.com/prometheus/client_golang` | v1.23.2 | Prometheus metrics |
| `go.opentelemetry.io/otel` | v1.39.0 | OpenTelemetry tracing |

### Tool Versions
| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.25.1 | Language runtime |
| `golangci-lint` | v2.5.0 | Linting and code quality |
| `controller-gen` | v0.18.0 | CRD, RBAC, webhook generation |
| `kustomize` | v5.7.1 | Kubernetes manifest management |
| `setup-envtest` | release-0.23 | Test environment setup |
| `helmify` | v0.4.19 | Helm chart generation |

## Agent Guidelines

### Before Making Changes
1. Read relevant code files first - NEVER propose changes without reading
2. Understand existing patterns and architecture (Chain of Responsibility, handler interfaces)
3. Verify compliance with constitutional principles
4. Plan changes using TodoWrite tool for multi-step tasks
5. Check existing tests to understand expected behavior

### When Writing Code
1. Follow TDD for service layer (`internal/controller/*/service/`)
2. Write tests BEFORE implementation using Ginkgo/Gomega
3. Use interfaces for all dependencies (defined at consumer site)
4. Inject dependencies via constructors
5. Wrap errors with context: `fmt.Errorf("context: %w", err)`
6. Classify errors: use `NewCriticalError` vs `NewRecoverableError` appropriately
7. Use `zerolog` for structured logging with appropriate levels
8. Keep cyclomatic complexity ≤12 (gocyclo) and cognitive complexity ≤20 (gocognit)
9. Use modern Go idioms: `range` over ints, `slices`/`maps` packages, generics where beneficial
10. Respect the handler chain pattern - don't bypass it with direct reconciler logic

### Kubernetes-Specific Guidelines
- Use `client.IgnoreNotFound(err)` when fetching resources that may be deleted
- Always use the status subresource for status updates
- Add RBAC markers (`+kubebuilder:rbac`) with minimal permissions
- Use predicates to filter unnecessary reconciliation events
- Ensure reconcile loops are idempotent and level-triggered
- Handle finalizers correctly: add on create, clean up on delete, remove after cleanup
- Use `RequeueAfter` with appropriate durations, not bare `Requeue: true`

### Security Considerations
- Avoid command injection, XSS, SQL injection, and OWASP top 10 vulnerabilities
- Never expose secrets in logs or error messages
- Handle all errors explicitly - no silent failures
- Use context for cancellation and timeouts
- Validate all external input (webhook validation, CRD field validation)
- Use CEL expressions in CRD markers for declarative validation where possible

### Anti-Patterns to Avoid
- Over-engineering solutions beyond requirements
- Adding features not explicitly requested
- Creating abstractions for one-time operations
- Premature optimization (profile first, optimize second)
- Global state or singleton patterns
- Backwards-compatibility hacks for unused code
- Using `Update` when `Patch` or SSA would be safer
- Ignoring error classification (treating all errors the same)
- Putting business logic directly in reconcilers (use handlers/services)
- Hardcoding requeue durations without constants
- Using `init()` functions (except for scheme registration)

### After Implementation
1. Run tests: `make test`
2. Verify coverage: `make test-coverage`
3. Run linter: `make lint`
4. Generate code if CRDs modified: `make generate && make manifests`
5. Update documentation for user-facing changes
6. Verify handler chain order is correct if handlers were added/modified

## Governance

This document synthesizes [CLAUDE.md](CLAUDE.md) and [.specify/memory/constitution.md](.specify/memory/constitution.md). The constitution supersedes all other practices. For amendments to constitutional principles, see the Governance section in the constitution.

**Last Updated**: 2026-02-19
