# Project Context

## Purpose

**KubeCloudScaler** (also known as **kcs**) is a Kubernetes operator that automatically scales cloud resources up or down based on time-based schedules defined through Custom Resource Definitions (CRDs).

The operator supports:
- **Kubernetes resources**: Deployments, StatefulSets, CronJobs, HorizontalPodAutoscalers, and GitHub AutoscalingRunnerSets
- **GCP resources**: Compute Engine instances and Cloud SQL instances (planned/extended support)

The project is inspired by [kube-downscaler](https://codeberg.org/hjacobs/kube-downscaler) and enables cost optimization by automatically scaling resources during off-peak hours or scheduled maintenance windows.

## Tech Stack

- **Language**: Go 1.22.0+
- **Framework / Libraries**:
  - Kubernetes Operator SDK (controller-runtime)
  - `cloud.google.com/go` - GCP client libraries
  - `github.com/rs/zerolog` - Structured JSON logging
  - `k8s.io/client-go` - Kubernetes client libraries
  - `github.com/onsi/ginkgo/v2` - BDD testing framework
  - `github.com/onsi/gomega` - Matcher library for tests
  - `golangci-lint` - Linting and code quality
- **Architecture**: Clean Architecture with Repository Pattern
- **Package Management**: Go modules
- **Container Registry**: OCI registry (ghcr.io)
- **Deployment**: Helm charts

## Project Conventions

### Code Style

- Follow **idiomatic Go conventions** as defined in [Effective Go](https://go.dev/doc/effective_go) and [Google's Go Style Guide](https://google.github.io/styleguide/go/)
- Use **named functions** over long anonymous ones
- Organize logic into **small, composable functions** with single responsibility
- Use **tabs for indentation** (Go standard)
- Use **single quotes** for strings (except to avoid escaping)
- Omit **semicolons** (unless required for disambiguation)
- Always use **strict equality** (`===`) instead of loose equality (`==`)
- **Limit line length** to 80 characters where practical
- Use **trailing commas** in multiline object/array literals
- Use `gofmt` or `goimports` to enforce formatting
- Enforce naming consistency with `golangci-lint`

#### Naming Conventions

- **PascalCase**: Components, type definitions, interfaces
- **kebab-case**: Directory names, file names
- **camelCase**: Variables, functions, methods, hooks, properties, props
- **UPPERCASE**: Environment variables, constants, global configurations
- Prefix event handlers with `handle`: `handleClick`, `handleSubmit`
- Prefix boolean variables with verbs: `isLoading`, `hasError`, `canSubmit`
- Prefix custom hooks with `use`: `useAuth`, `useForm`
- Use complete words over abbreviations (except: `err`, `req`, `res`, `props`, `ref`)

### Architecture Patterns

- **Clean Architecture**: Structure code into layers:
  - `cmd/` - Application entrypoints
  - `internal/controller/` - Controllers (reconciliation logic)
  - `internal/service/` - Business logic and use cases
  - `internal/repository/` - Data access layer
  - `internal/model/` - Domain models
  - `internal/config/` - Configuration management
  - `internal/utils/` - Internal utilities
  - `pkg/` - Shared utilities and packages (exposed externally)
  - `api/` - CRD resource definitions
  - `configs/` - Configuration schemas and loading
  - `test/` - Test utilities, mocks, and integration tests

- **Repository Pattern**: Separate data access logic from business logic
- **Interface-Driven Development**:
  - Prefer interfaces for dependencies to enable mocking and testing
  - Use explicit dependency injection via constructors
  - Ensure all public functions interact with interfaces, not concrete types
  - Favor small, purpose-specific interfaces
- **Composition over Inheritance**: Use composition to build complex types
- **Domain-Driven Design**: Apply DDD principles where applicable
- **Dependency Inversion**: High-level modules should not depend on low-level modules

#### Key Patterns to Follow

- Use **Clean Architecture** and **Repository Pattern**
- Implement input validation using Go structs and validation tags
- Use **custom error types** for wrapping and handling business logic errors
- Logging via `zerolog` with JSON formatting (raw, json, or json with ECS fields)
- Use **dependency injection** via constructors (avoid global state)
- Group code by feature when it improves clarity and cohesion
- Keep logic decoupled from framework-specific code
- Write **short, focused functions** with a single responsibility
- Always **check and handle errors explicitly**, using wrapped errors for traceability (`fmt.Errorf("context: %w", err)`)
- Avoid **global state**; use constructor functions to inject dependencies
- Leverage **Go's context propagation** for request-scoped values, deadlines, and cancellations
- Use **goroutines safely**; guard shared state with channels or sync primitives
- **Defer closing resources** and handle them carefully to avoid leaks

#### Patterns to Avoid

- Don't use global state unless absolutely required
- Don't hardcode config—use environment variables or config files
- Don't panic or exit in library code; return errors instead
- Don't expose secrets—use `.env` or secret managers
- Avoid embedding business logic in HTTP handlers
- Avoid unnecessary abstraction; keep things simple and readable

### Testing Strategy

- Use **Ginkgo** (BDD framework) and **Gomega** (matcher library) for assertions and testing
- Use **testify** for additional testing utilities and mocking
- Organize tests under `test/unit/` and `test/integration/`
- **Mock external services** (e.g., Kubernetes API, GCP APIs) using interfaces and mocks for unit tests
- Include **table-driven tests** for functions with many input variants
- Follow **TDD** (Test-Driven Development) for core business logic
- Write **unit tests** using table-driven patterns and parallel execution
- **Mock external interfaces** cleanly using generated or handwritten mocks
- Separate **fast unit tests** from slower integration and E2E tests
- Ensure **test coverage** for every exported function, with behavioral checks
- Use tools like `go test -cover` to ensure adequate test coverage
- Write **benchmarks** to track performance regressions and identify bottlenecks

### Git Workflow

- Use semantic versioning for releases
- Follow conventional commit messages
- Maintain a `CHANGELOG.md` for tracking changes
- Use feature branches for development
- Require code review before merging to main branch

## Domain Context

### Kubernetes Operator

This project is a **Kubernetes operator** that manages Custom Resources (CRs) to control the scaling behavior of Kubernetes and cloud resources. The operator follows the controller-runtime pattern with reconciliation loops.

### Custom Resources

The operator defines three main CRD types:
- **K8s**: Manages Kubernetes resources (Deployments, StatefulSets, CronJobs, HPAs, AutoscalingRunnerSets)
- **Gcp**: Manages GCP resources (Compute Engine, Cloud SQL)
- **Flow**: Orchestrates multiple K8s and GCP resources together

### Scaling Periods

Resources are scaled based on **time periods** defined in the CR spec:
- **Recurring periods**: Daily, weekly, or custom schedules
- **Timezone support**: All times are timezone-aware
- **Period types**: "up", "down", or "restore" (restore original values)
- **Replica ranges**: Min/max replicas for scaling operations

### Resource Management

- Resources are identified by **label selectors** and **namespace filters**
- System namespaces are automatically excluded (configurable)
- Resources can be marked with `kubecloudscaler.cloud/ignore` label to skip scaling
- Original resource values are stored in annotations and restored when periods end
- The operator uses **finalizers** to ensure proper cleanup

### Observability

- **Logging**: Structured JSON logs via zerolog (raw, json, or json with ECS fields)
- **Metrics**: Prometheus metrics exposed for monitoring
- **Tracing**: Minimal cardinality in labels and traces to keep observability overhead low
- **Log levels**: Use info, warn, error appropriately

## Important Constraints

### Technical Constraints

- Must follow **Kubernetes operator patterns** and best practices
- Must use `context.Context` for request-scoped values and cancellation
- **No global state** - all dependencies must be injected
- Must handle errors explicitly - no panics in library code
- Must support **graceful shutdown** and resource cleanup
- Must be **thread-safe** when using goroutines
- Must minimize **allocations** and avoid premature optimization
- Must instrument key areas (Kubernetes API calls, GCP API calls, heavy computation)

### Business Constraints

- Must be **backward compatible** with existing CRD versions
- Must support **multiple CRD API versions** (v1alpha1, v1alpha2, v1alpha3)
- Must handle **cluster-wide operations** safely
- Must respect **RBAC** (Role-Based Access Control) permissions
- Must be **cost-effective** - the operator itself should have minimal resource footprint

### Regulatory/Operational Constraints

- Must not expose secrets in logs or error messages
- Must follow **Kubernetes security best practices**
- Must support **namespace isolation** for multi-tenant scenarios
- Must be **observable** for operations teams

## External Dependencies

### Kubernetes APIs

- **Kubernetes API Server**: For managing Kubernetes resources (Deployments, StatefulSets, CronJobs, HPAs, Namespaces)
- **Custom Resource Definitions**: The operator defines and manages its own CRDs
- **Dynamic Client**: For managing third-party resources (e.g., GitHub AutoscalingRunnerSets)

### Google Cloud Platform (GCP)

- **Compute Engine API**: For scaling GCP VM instances
- **Cloud SQL Admin API**: For scaling Cloud SQL instances
- **GCP Authentication**: Uses service account credentials or workload identity

### Development Tools

- **controller-gen**: For generating CRD manifests and deep copy methods
- **golangci-lint**: For code linting and quality checks
- **ginkgo/gomega**: For BDD-style testing
- **kustomize**: For Kubernetes resource management
- **Helm**: For packaging and deployment

### Observability

- **Prometheus**: For metrics collection
- **JSON Logging**: For log aggregation (supports raw, json, or json with ECS fields)
