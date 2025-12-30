<!--
Sync Impact Report:
Version change: N/A → 1.0.0 (initial constitution from project.md)
Modified principles: N/A (new constitution)
Added sections: Core Principles, Architecture Constraints, Development Workflow, Governance
Removed sections: N/A
Templates requiring updates:
  ✅ plan-template.md - Constitution Check section already references constitution
  ✅ spec-template.md - No direct constitution references found
  ✅ tasks-template.md - No direct constitution references found
  ✅ checklist-template.md - No direct constitution references found
Follow-up TODOs: None
-->

# KubeCloudScaler Constitution

## Clarifications

### Session 2025-12-30

- Q: What defines "core business logic" for TDD requirements? → A: All business logic in `internal/controller/*/service/` layer (the service layer within the flow controller)
- Q: What is the minimum test coverage threshold and how is it enforced? → A: 80% minimum with CI enforcement
- Q: What is the amendment approval process for constitution changes? → A: Maintainer approval via PR review
- Q: What complexity threshold requires justification in code reviews? → A: Cyclomatic complexity >10 or new architectural patterns
- Q: What are the quantitative resource footprint targets for the operator? → A: Memory <100MB, CPU <100m (requests)

## Core Principles

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

**Rationale**: Clean Architecture ensures maintainability, testability, and clear separation of concerns. The Repository Pattern enables mocking and testing while keeping business logic decoupled from data access.

### II. Interface-Driven Development & Dependency Injection
All dependencies MUST be defined as interfaces. Dependency injection MUST be performed via constructors (no global state). Interfaces MUST be small and purpose-specific. High-level modules MUST NOT depend on low-level modules (Dependency Inversion Principle).

**Rationale**: Interface-driven development enables testing through mocking, improves modularity, and allows for easier refactoring. Dependency injection via constructors makes dependencies explicit and testable.

### III. Test-Driven Development (TDD)
TDD is mandatory for all business logic in the `internal/controller/*/service/` layer: Tests written → User approved → Tests fail → Then implement. The Red-Green-Refactor cycle MUST be strictly enforced. All exported functions MUST have test coverage with behavioral checks. Minimum test coverage of 80% MUST be maintained and enforced via CI/CD.

**Rationale**: TDD ensures code correctness, improves design, and provides documentation through tests. It catches regressions early and enforces good design practices. The 80% coverage threshold balances thoroughness with practicality while ensuring core functionality is tested.

### IV. Observability & Structured Logging
Structured JSON logging via `zerolog` is REQUIRED (raw, json, or json with ECS fields). Prometheus metrics MUST be exposed for monitoring. Logging MUST use appropriate levels (info, warn, error). Minimal cardinality in labels and traces to keep observability overhead low.

**Rationale**: Observability is critical for Kubernetes operators running in production. Structured logging enables log aggregation and analysis, while metrics provide operational insights.

### V. Go Conventions & Code Style
Code MUST follow idiomatic Go conventions as defined in Effective Go and Google's Go Style Guide. Use tabs for indentation, named functions over long anonymous ones, small composable functions with single responsibility. Limit line length to 80 characters where practical. Use `gofmt` or `goimports` for formatting. Enforce naming consistency with `golangci-lint`.

**Rationale**: Consistency in code style improves readability and maintainability. Following Go conventions ensures the codebase is familiar to Go developers and integrates well with the Go ecosystem.

### VI. Error Handling & Resource Management
All errors MUST be handled explicitly using wrapped errors for traceability (`fmt.Errorf("context: %w", err)`). No panics in library code - return errors instead. Resources MUST be closed with defer statements. Context propagation MUST be used for request-scoped values, deadlines, and cancellations.

**Rationale**: Explicit error handling prevents silent failures and improves debuggability. Proper resource management prevents leaks and ensures graceful shutdown.

### VII. Kubernetes Operator Patterns
The operator MUST follow Kubernetes operator patterns and best practices using controller-runtime. MUST use `context.Context` for request-scoped values and cancellation. MUST support graceful shutdown and resource cleanup. MUST be thread-safe when using goroutines. MUST handle cluster-wide operations safely and respect RBAC permissions.

**Rationale**: Following Kubernetes operator patterns ensures compatibility with the Kubernetes ecosystem, proper resource management, and operational reliability.

### VIII. Security & Compliance
Secrets MUST NOT be exposed in logs or error messages. MUST follow Kubernetes security best practices. MUST support namespace isolation for multi-tenant scenarios. MUST be backward compatible with existing CRD versions and support multiple CRD API versions (v1alpha1, v1alpha2, v1alpha3).

**Rationale**: Security is paramount for operators managing production resources. Backward compatibility ensures smooth upgrades and adoption.

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
- MUST be cost-effective - the operator itself MUST maintain minimal resource footprint: memory requests <100MB, CPU requests <100m
- MUST support multiple CRD API versions for backward compatibility
- MUST handle cluster-wide operations safely
- MUST respect RBAC (Role-Based Access Control) permissions

## Development Workflow

### Testing Strategy
- Use **Ginkgo** (BDD framework) and **Gomega** (matcher library) for assertions and testing
- Organize tests under `test/unit/` and `test/integration/`
- Mock external services (e.g., Kubernetes API, GCP APIs) using interfaces and mocks for unit tests
- Include table-driven tests for functions with many input variants
- Separate fast unit tests from slower integration and E2E tests
- Write benchmarks to track performance regressions
- Maintain minimum 80% test coverage enforced via CI/CD pipeline

### Git Workflow
- Use semantic versioning for releases
- Follow conventional commit messages
- Maintain a `CHANGELOG.md` for tracking changes
- Use feature branches for development
- Require code review before merging to main branch

### Code Review Requirements
- All PRs/reviews MUST verify compliance with this constitution
- Complexity MUST be justified with rationale when cyclomatic complexity exceeds 10 or when introducing new architectural patterns
- Tests MUST be included for new functionality
- Documentation MUST be updated for user-facing changes

## Governance

This constitution supersedes all other practices and conventions. Amendments require:
1. Documentation of the proposed change and rationale
2. Approval from project maintainers via PR review process
3. Migration plan if the change affects existing code
4. Version bump according to semantic versioning:
   - **MAJOR**: Backward incompatible governance/principle removals or redefinitions
   - **MINOR**: New principle/section added or materially expanded guidance
   - **PATCH**: Clarifications, wording, typo fixes, non-semantic refinements

**Version**: 1.0.0 | **Ratified**: 2025-12-30 | **Last Amended**: 2025-12-30
