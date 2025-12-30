# Implementation Plan: Rewrite GCP Controller Using Chain of Responsibility Pattern

**Branch**: `001-gcp-controller-chain` | **Date**: 2025-12-30 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-gcp-controller-chain/spec.md`

## Summary

Refactor the GCP controller's monolithic Reconcile function into a Chain of Responsibility pattern with discrete handlers for each reconciliation step (fetch, finalizer, authentication, period validation, resource scaling, status update). This improves code maintainability, testability, and aligns with Clean Architecture principles while maintaining 100% backward compatibility and functional equivalence.

**Pattern Reference**: Implementation is based on the [Chain of Responsibility pattern in Go](https://refactoring.guru/design-patterns/chain-of-responsibility/go/example) from refactoring.guru, adapted with a centralized chain approach for enhanced error handling, observability, and Kubernetes-specific requirements.

## Technical Context

**Language/Version**: Go 1.25.1 (constitution requires 1.22.0+)
**Primary Dependencies**:
- `sigs.k8s.io/controller-runtime` - Kubernetes controller framework (existing)
- `cloud.google.com/go/compute` - GCP Compute Engine API client (existing)
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
- Must maintain 100% backward compatibility with existing GCP scaler resources
- Must maintain functional equivalence with current implementation
- Must follow Clean Architecture principles (handlers in service layer)
- Must use interface-driven development with dependency injection
- Must reduce cyclomatic complexity by at least 50% (SC-002)
- Must achieve 80% test coverage (SC-003)
- Must comply with constitutional requirements (Go conventions, error handling, observability)

**Scale/Scope**:
- Refactor single controller: `internal/controller/gcp/scaler_controller.go`
- Create handler chain infrastructure in `internal/controller/gcp/service/`
- Create 6+ discrete handlers (fetch, finalizer, authentication, period validation, resource scaling, status update)
- Maintain existing API contracts and resource specifications
- Preserve all existing test cases

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Pre-Design Check

✅ **Principle I (Clean Architecture)**: **COMPLIANCE REQUIRED** - Feature refactors controller to follow Clean Architecture by moving business logic to service layer (`internal/controller/gcp/service/`) with clear separation of concerns

✅ **Principle II (Interface-Driven Development)**: **COMPLIANCE REQUIRED** - Feature implements handler interface pattern with dependency injection, enabling interface-based testing and modularity

✅ **Principle III (TDD)**: **COMPLIANCE REQUIRED** - Feature improves testability enabling TDD for handler implementations in service layer. Must achieve 80% test coverage (SC-003)

✅ **Principle IV (Observability)**: No violation - Refactoring maintains existing logging structure using zerolog

✅ **Principle V (Go Conventions)**: No violation - Refactoring follows Go conventions and improves code structure

✅ **Principle VI (Error Handling)**: **COMPLIANCE REQUIRED** - Feature implements categorized error handling (critical vs recoverable) with proper error wrapping and context propagation

✅ **Principle VII (Kubernetes Operator Patterns)**: No violation - Refactoring maintains controller-runtime patterns and reconciliation behavior

✅ **Principle VIII (Security & Compliance)**: **COMPLIANCE REQUIRED** - Feature maintains backward compatibility with existing CRD versions and security practices

**Gates**: All gates pass. Feature directly supports constitutional requirements for Clean Architecture, interface-driven development, and TDD.

### Post-Design Check

✅ **Principle I (Clean Architecture)**: **COMPLIANCE ACHIEVED** - Design places handlers in service layer (`internal/controller/gcp/service/`) with clear separation: handlers contain business logic, controller contains reconciliation orchestration, interfaces separate dependencies

✅ **Principle II (Interface-Driven Development)**: **COMPLIANCE ACHIEVED** - Design defines Handler interface for all handlers, uses dependency injection for all handler dependencies (Kubernetes client, GCP client, logger), enables interface-based testing

✅ **Principle III (TDD)**: **COMPLIANCE ACHIEVED** - Design enables TDD for each handler independently, handlers can be tested with mocked dependencies, supports 80% test coverage requirement

✅ **Principle IV (Observability)**: **COMPLIANCE ACHIEVED** - Design maintains structured logging using zerolog, handlers log execution with context, chain logs handler execution flow

✅ **Principle V (Go Conventions)**: **COMPLIANCE ACHIEVED** - Design follows Go conventions: interfaces, composition, dependency injection, error handling patterns

✅ **Principle VI (Error Handling)**: **COMPLIANCE ACHIEVED** - Design implements categorized error handling (critical vs recoverable), uses error wrapping for traceability, proper error propagation through chain

✅ **Principle VII (Kubernetes Operator Patterns)**: **COMPLIANCE ACHIEVED** - Design maintains controller-runtime patterns, preserves reconciliation behavior, supports graceful shutdown and resource cleanup

✅ **Principle VIII (Security & Compliance)**: **COMPLIANCE ACHIEVED** - Design maintains backward compatibility with existing CRD versions, preserves security practices, no secrets exposed in logs

**Gates**: All gates pass. Design maintains constitutional compliance and improves code structure while preserving functionality.

## Project Structure

### Documentation (this feature)

```text
specs/001-gcp-controller-chain/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
# Refactored GCP controller structure
internal/controller/gcp/
├── scaler_controller.go          # MODIFIED: Simplified Reconcile function that delegates to handler chain
├── scaler_controller_test.go     # MODIFIED: Updated tests for refactored controller
├── suite_test.go                 # EXISTING: Test suite setup (unchanged)
└── service/                      # NEW: Service layer with handler chain
    ├── interfaces.go             # NEW: Handler interface and chain interface definitions
    ├── chain.go                  # NEW: Chain implementation and execution logic
    ├── context.go                # NEW: Reconciliation context structure
    ├── handlers/                 # NEW: Individual handler implementations
    │   ├── fetch_handler.go      # NEW: Fetch scaler resource handler
    │   ├── finalizer_handler.go  # NEW: Finalizer management handler
    │   ├── auth_handler.go       # NEW: Authentication and GCP client setup handler
    │   ├── period_handler.go     # NEW: Period validation handler
    │   ├── scaling_handler.go    # NEW: Resource scaling handler
    │   └── status_handler.go     # NEW: Status update handler
    └── handlers_test.go          # NEW: Unit tests for handlers
```

**Structure Decision**: Single project structure. Refactoring maintains existing controller location while introducing service layer (`internal/controller/gcp/service/`) following Clean Architecture. Handlers are organized in `service/handlers/` subdirectory for clarity. This structure aligns with constitutional requirements and enables independent testing of each handler.

**Pattern Implementation**: Uses an enhanced centralized Chain of Responsibility pattern (adapted from [refactoring.guru](https://refactoring.guru/design-patterns/chain-of-responsibility/go/example)):
- **Classic Pattern**: Handlers have `execute()` and `setNext()` methods, each handler knows about the next
- **Our Adaptation**: Centralized `Chain` struct manages handler list and execution, handlers only implement `Handle()` method
- **Benefits**: Better error handling, observability, requeue management, and testability for Kubernetes controllers

## Pattern Implementation Comparison

### Classic Chain of Responsibility (refactoring.guru)

The [classic pattern](https://refactoring.guru/design-patterns/chain-of-responsibility/go/example) uses:

```go
type Department interface {
    execute(*Patient)
    setNext(Department)
}

type Reception struct {
    next Department
}

func (r *Reception) execute(p *Patient) {
    // Process patient
    r.next.execute(p)  // Pass to next handler
}
```

**Characteristics**:
- Each handler knows about the next handler
- Handlers call `next.execute()` to pass control
- Chain is built by calling `setNext()` on each handler
- Simple, direct control flow

### Our Enhanced Centralized Chain

Our implementation uses:

```go
type Handler interface {
    Handle(ctx context.Context, req *ReconciliationContext) (*ReconciliationResult, error)
}

type HandlerChain struct {
    handlers []Handler
    logger   *zerolog.Logger
}

func (c *HandlerChain) Execute(ctx context.Context, req *ReconciliationContext) (ctrl.Result, error) {
    for i, handler := range c.handlers {
        result, err := handler.Handle(ctx, req)
        // Centralized error handling, logging, requeue management
    }
}
```

**Characteristics**:
- Handlers don't know about each other (better decoupling)
- Centralized chain manages execution, error handling, and observability
- Handlers return result/error instead of calling next handler
- Enhanced with Kubernetes-specific features (requeue, error categorization)

### Why Our Adaptation

| Requirement | Classic Pattern | Our Adaptation | Benefit |
|-------------|----------------|----------------|---------|
| **Error Handling** | Handler decides to call next | Chain categorizes errors centrally | Better error management |
| **Observability** | Scattered logging | Centralized structured logging | Better debugging |
| **Requeue Logic** | Not applicable | Chain tracks requeue requests | Kubernetes-specific |
| **Testability** | Handlers coupled | Handlers independent | Easier unit testing |
| **State Management** | Passed object | Shared context | Better state sharing |

**Conclusion**: Our centralized chain approach is better suited for Kubernetes controllers where we need sophisticated error handling, observability, and requeue behavior that's easier to manage in a centralized location.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No violations requiring justification.
