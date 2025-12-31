# Implementation Plan: Rewrite K8s Controller Using Chain of Responsibility Pattern

**Branch**: `002-k8s-controller-chain` | **Date**: 2025-12-30 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/002-k8s-controller-chain/spec.md`

## Summary

Refactor the K8s controller's monolithic Reconcile function into a Chain of Responsibility pattern using the classic Go pattern from refactoring.guru where handlers have `execute()` and `setNext()` methods. This breaks down the complex reconciliation logic into discrete handlers (fetch, finalizer, authentication, period validation, resource scaling, status update) that are linked together via `setNext()` calls. Each handler maintains a reference to the next handler and explicitly calls `next.execute()` to pass control. This improves code maintainability, testability, and aligns with Clean Architecture principles while maintaining 100% backward compatibility and functional equivalence.

## Technical Context

**Language/Version**: Go 1.25.1 (constitution requires 1.22.0+)
**Primary Dependencies**:
- `sigs.k8s.io/controller-runtime` - Kubernetes controller framework (existing)
- `k8s.io/client-go` - Kubernetes client library (existing)
- `github.com/rs/zerolog` - Structured logging (constitution requirement)
- `github.com/onsi/ginkgo/v2` - BDD testing framework (constitution requirement)
- `github.com/onsi/gomega` - Matcher library (constitution requirement)

**Storage**: N/A (refactoring existing controller logic, no new storage)
**Testing**:
- Ginkgo/Gomega for BDD-style tests (constitution requirement)
- Existing test suite must pass with refactored implementation
- Unit tests for individual handlers with mocked dependencies

**Target Platform**: Linux (Kubernetes operator, runs in containers)
**Project Type**: Single project (Kubernetes operator)
**Performance Goals**:
- Handler unit tests execute in under 100ms each (SC-004)
- No performance degradation compared to current implementation
- Reconciliation latency remains unchanged

**Constraints**:
- Must maintain 100% backward compatibility with existing K8s scaler resources
- Must maintain functional equivalence with current implementation
- Must follow Clean Architecture principles (handlers in service layer)
- Must use interface-driven development with dependency injection
- Must implement classic Chain of Responsibility pattern with `execute()` and `setNext()` methods (refactoring.guru style)
- Must reduce cyclomatic complexity by at least 50% (SC-002)
- Must achieve 80% test coverage (SC-003)
- Must comply with constitutional requirements (Go conventions, error handling, observability)

**Scale/Scope**:
- Refactor single controller: `internal/controller/k8s/scaler_controller.go`
- Create handler chain infrastructure in `internal/controller/k8s/service/`
- Create 6+ discrete handlers (fetch, finalizer, authentication, period validation, resource scaling, status update)
- Maintain existing API contracts and resource specifications
- Preserve all existing test cases

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Clean Architecture & Repository Pattern
✅ **PASS**: Handlers will be placed in `internal/controller/k8s/service/` layer (service layer within controller), following Clean Architecture principles. Handlers use interfaces for dependencies (Kubernetes client, logger) enabling repository pattern compliance.

### II. Interface-Driven Development & Dependency Injection
✅ **PASS**: Handler interface will be defined with `execute()` and `setNext()` methods. All dependencies (Kubernetes client, logger, K8s client factory) will be injected via constructors. No global state will be used.

### III. Test-Driven Development (TDD)
✅ **PASS**: TDD is mandatory for handler implementations in `internal/controller/k8s/service/` layer. Tests will be written first, then implementation. Minimum 80% test coverage required and enforced via CI/CD.

### IV. Observability & Structured Logging
✅ **PASS**: Structured JSON logging via `zerolog` will be used in handlers. Logging will use appropriate levels (info, warn, error, debug) for handler execution flow.

### V. Go Conventions & Code Style
✅ **PASS**: Code will follow idiomatic Go conventions. Use tabs for indentation, named functions, small composable functions. Format with `gofmt`/`goimports`. Enforce with `golangci-lint`.

### VI. Error Handling & Resource Management
✅ **PASS**: Errors will be handled explicitly using wrapped errors. Error categorization (critical vs recoverable) will be implemented. Context propagation will be used throughout handler chain. Resources will be closed with defer statements where applicable.

### VII. Kubernetes Operator Patterns
✅ **PASS**: Controller will follow Kubernetes operator patterns using controller-runtime. `context.Context` will be used for request-scoped values and cancellation. Graceful shutdown and resource cleanup will be supported. Thread-safe operations will be maintained.

### VIII. Security & Compliance
✅ **PASS**: Secrets will not be exposed in logs or error messages. Kubernetes security best practices will be followed. Namespace isolation will be supported. Backward compatibility with existing CRD versions will be maintained.

**Gate Status**: ✅ **ALL GATES PASS** - Proceed to Phase 0 research

### Post-Design Constitution Check

*Re-evaluated after Phase 1 design completion.*

All constitutional requirements remain satisfied after design:

- ✅ **Clean Architecture**: Handler chain structure in `internal/controller/k8s/service/` maintains clear layer separation
- ✅ **Interface-Driven Development**: Handler interface with `execute()` and `setNext()` methods enables dependency injection
- ✅ **TDD**: Handler implementations in service layer will follow TDD with tests written first
- ✅ **Observability**: Structured logging via `zerolog` integrated into handler execution flow
- ✅ **Go Conventions**: Code structure follows idiomatic Go patterns with small, composable handlers
- ✅ **Error Handling**: Error categorization (critical vs recoverable) implemented with proper error wrapping
- ✅ **Kubernetes Operator Patterns**: Controller maintains compatibility with controller-runtime patterns
- ✅ **Security & Compliance**: Secrets handling and backward compatibility preserved

**Post-Design Gate Status**: ✅ **ALL GATES PASS** - Design is constitutionally compliant

## Project Structure

### Documentation (this feature)

```text
specs/002-k8s-controller-chain/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
# Refactored K8s controller structure
internal/controller/k8s/
├── scaler_controller.go          # MODIFIED: Simplified Reconcile function that delegates to handler chain
├── scaler_controller_test.go     # MODIFIED: Updated tests for refactored controller
├── suite_test.go                 # EXISTING: Test suite setup (unchanged)
└── service/                      # NEW: Service layer with handler chain (classic pattern)
    ├── interfaces.go             # NEW: Handler interface with execute() and setNext() methods
    ├── context.go                # NEW: Reconciliation context structure
    ├── errors.go                 # NEW: Error categorization types
    ├── handlers/                 # NEW: Individual handler implementations
    │   ├── fetch_handler.go      # NEW: Fetch scaler resource handler
    │   ├── finalizer_handler.go  # NEW: Finalizer management handler
    │   ├── auth_handler.go       # NEW: Authentication and K8s client setup handler
    │   ├── period_handler.go     # NEW: Period validation handler
    │   ├── scaling_handler.go    # NEW: Resource scaling handler
    │   └── status_handler.go     # NEW: Status update handler
    └── handlers_test.go          # NEW: Unit tests for handlers
```

**Structure Decision**: Single project structure. Handlers will be organized in `internal/controller/k8s/service/handlers/` directory following Clean Architecture principles. The classic Chain of Responsibility pattern will be implemented where each handler has `execute()` and `setNext()` methods, and handlers are linked together via `setNext()` calls during chain construction.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No violations - all constitutional requirements are met.
