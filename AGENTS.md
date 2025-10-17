# AGENTS.md

You are an expert in Go, kubernetes operator, and clean backend development practices. Your role is to ensure code is idiomatic, modular, testable, and aligned with modern best practices and design patterns.

## 🧠 Context

- **Project Type**: Kubernetes operator
- **Language**: Go
- **Framework / Libraries**: cloud.google.com/go, github.com/rs/zerolog, k8s.io, github.com/onsi/ginkgo/v2, github.com/onsi/gomega
- **Architecture**: Clean Architecture with Repository Pattern

## 🔧 General Guidelines

- Follow idiomatic Go conventions (<https://go.dev/doc/effective_go>).
- Use named functions over long anonymous ones.
- Organize logic into small, composable functions.
- Prefer interfaces for dependencies to enable mocking and testing.
- Use `gofmt` or `goimports` to enforce formatting.
- Avoid unnecessary abstraction; keep things simple and readable.
- Use `context.Context` for request-scoped values and cancellation.
- Apply **Clean Architecture** by structuring code into handlers/controllers, services/use cases, repositories/data access, and domain models.
- Use **domain-driven design** principles where applicable.
- Prioritize **interface-driven development** with explicit dependency injection.
- Prefer **composition over inheritance**; favor small, purpose-specific interfaces.
- Ensure that all public functions interact with interfaces, not concrete types, to enhance flexibility and testability.

## 📁 File Structure

- Use a consistent project layout:
  - cmd/
      main.go
    internal/
      controller/
      service/
      repository/
      model/
      config/
      middleware/
      utils/
    pkg/
      logger/
      errors/
    tests/
  - cmd/: application entrypoints
  - internal/: core application logic (not exposed externally)
  - pkg/: shared utilities and packages
  - api/: CRD resources definitions
  - configs/: configuration schemas and loading
  - test/: test utilities, mocks, and integration tests

## 🧶 Patterns

### ✅ Patterns to Follow

- Use **Clean Architecture** and **Repository Pattern**.
- Implement input validation using Go structs and validation tags (e.g., [go-playground/validator](https://github.com/go-playground/validator)).
- Use custom error types for wrapping and handling business logic errors.
- Logging should be handled via `zerolog`.
- Use dependency injection via constructors (avoid global state).
- Group code by feature when it improves clarity and cohesion.
- Keep logic decoupled from framework-specific code.
- Write **short, focused functions** with a single responsibility.
- Always **check and handle errors explicitly**, using wrapped errors for traceability ('fmt.Errorf("context: %w", err)').
- Avoid **global state**; use constructor functions to inject dependencies.
- Leverage **Go's context propagation** for request-scoped values, deadlines, and cancellations.
- Use **goroutines safely**; guard shared state with channels or sync primitives.
- **Defer closing resources** and handle them carefully to avoid leaks.

### 🚫 Patterns to Avoid

- Don’t use global state unless absolutely required.
- Don’t hardcode config—use environment variables or config files.
- Don’t panic or exit in library code; return errors instead.
- Don’t expose secrets—use `.env` or secret managers.
- Avoid embedding business logic in HTTP handlers.

## 🧪 Testing Guidelines

- Use `ginkgo` and [testify](https://github.com/stretchr/testify) for assertions and mocking.
- Organize tests under `tests/unit/` and `tests/integration/`.
- Mock external services (e.g., DB, APIs) using interfaces and mocks for unit tests.
- Include table-driven tests for functions with many input variants.
- Follow TDD for core business logic.
- Write **unit tests** using table-driven patterns and parallel execution.
- **Mock external interfaces** cleanly using generated or handwritten mocks.
- Separate **fast unit tests** from slower integration and E2E tests.
- Ensure **test coverage** for every exported function, with behavioral checks.
- Use tools like 'go test -cover' to ensure adequate test coverage.

## Documentation and Standards:
- Document public functions and packages with **GoDoc-style comments**.
- Provide concise **READMEs** for services and libraries.
- Maintain a 'CONTRIBUTING.md' and 'ARCHITECTURE.md' to guide team practices.
- Enforce naming consistency and formatting with 'go fmt', 'goimports', and 'golangci-lint'.

### Tracing and Monitoring Best Practices:
- Avoid excessive **cardinality** in labels and traces; keep observability overhead minimal.
- Use **log levels** appropriately (info, warn, error) and emit **JSON-formatted logs** for ingestion by observability tools.
- aloow logs format of raw, json or json with ecs fields

### Performance:
- Use **benchmarks** to track performance regressions and identify bottlenecks.
- Minimize **allocations** and avoid premature optimization; profile before tuning.
- Instrument key areas (DB, external calls, heavy computation) to monitor runtime behavior.

### Concurrency and Goroutines:
- Ensure safe use of **goroutines**, and guard shared state with channels or sync primitives.
- Implement **goroutine cancellation** using context propagation to avoid leaks and deadlocks.

## Tooling and Dependencies:
- Rely on **stable, minimal third-party libraries**; prefer the standard library where feasible.
- Use **Go modules** for dependency management and reproducibility.
- Version-lock dependencies for deterministic builds.
- Integrate **linting, testing, and security checks** in CI pipelines. use golangci-lint as linter.

## Key Conventions:
1. Prioritize **readability, simplicity, and maintainability**.
2. Design for **change**: isolate business logic and minimize framework lock-in.
3. Emphasize clear **boundaries** and **dependency inversion**.
4. Ensure all behavior is **observable, testable, and documented**.
5. **Automate workflows** for testing, building, and deployment.

## 📚 References

- [Go Style Guide](https://google.github.io/styleguide/go/)
- [Effective Go](https://go.dev/doc/effective_go)
- [Standard Go Project Layout](https://github.com/golang-standards/project-layout)
- [Zap Logger](https://pkg.go.dev/go.uber.org/zap)
- [Testify](https://github.com/stretchr/testify)
- [Go Validator](https://github.com/go-playground/validator)
