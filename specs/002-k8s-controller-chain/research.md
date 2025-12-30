# Research: Rewrite K8s Controller Using Chain of Responsibility Pattern

## Chain of Responsibility Pattern Implementation

### Decision: Use classic Chain of Responsibility pattern with execute() and setNext() methods

**Pattern Reference**: Based on the [Chain of Responsibility pattern in Go](https://refactoring.guru/design-patterns/chain-of-responsibility/go/example) from refactoring.guru.

**Rationale**:
- The specification explicitly requires the classic Chain of Responsibility pattern from refactoring.guru where handlers have `execute()` and `setNext()` methods
- Each handler maintains a reference to the next handler and explicitly calls `next.execute()` to pass control
- This pattern is ideal for breaking down complex sequential operations into discrete, testable units
- Each handler has a single responsibility, improving maintainability and testability
- Pattern enables independent testing of each handler with mocked dependencies
- Fixed order at compile time (via `setNext()` calls) ensures predictable execution and simplifies debugging
- Pattern aligns with Clean Architecture by separating concerns into service layer

**Classic Pattern Structure**:
```go
// Handler interface with execute() and setNext() methods
type Handler interface {
    execute(ctx *ReconciliationContext) error
    setNext(next Handler)
}

// Handler implementation
type FetchHandler struct {
    next Handler
}

func (h *FetchHandler) execute(ctx *ReconciliationContext) error {
    // ... handler logic ...
    if h.next != nil {
        return h.next.execute(ctx)
    }
    return nil
}

func (h *FetchHandler) setNext(next Handler) {
    h.next = next
}
```

**Chain Construction**:
```go
// Build chain by linking handlers via setNext() calls
fetchHandler := NewFetchHandler()
finalizerHandler := NewFinalizerHandler()
authHandler := NewAuthHandler()
periodHandler := NewPeriodHandler()
scalingHandler := NewScalingHandler()
statusHandler := NewStatusHandler()

// Link handlers in order
fetchHandler.setNext(finalizerHandler)
finalizerHandler.setNext(authHandler)
authHandler.setNext(periodHandler)
periodHandler.setNext(scalingHandler)
scalingHandler.setNext(statusHandler)

// Start chain execution
return fetchHandler.execute(ctx)
```

**Alternatives considered**:
- **Centralized Chain Pattern** (GCP controller approach): Rejected - Specification explicitly requires classic pattern with `execute()` and `setNext()` methods
- **Strategy Pattern**: Rejected - Strategy is for interchangeable algorithms, not sequential processing
- **Pipeline Pattern**: Considered - Similar to Chain but typically for data transformation. Chain better fits reconciliation flow with early termination capability
- **Template Method Pattern**: Rejected - Would require inheritance hierarchy, conflicts with Go's composition-over-inheritance approach
- **Middleware Pattern**: Considered - Similar structure but typically for request/response. Chain better fits reconciliation context

**Implementation approach**:
- Define `Handler` interface with `execute(ctx *ReconciliationContext) error` and `setNext(next Handler)` methods
- Each handler implements the interface and maintains a `next Handler` field
- Handlers call `next.execute(ctx)` to pass control to the next handler
- Chain is constructed by calling `setNext()` on each handler in order
- First handler in chain is called to start execution
- Shared `ReconciliationContext` is passed through the chain, allowing handlers to share state and results

## Error Categorization Strategy

### Decision: Categorize errors as critical vs recoverable

**Rationale**:
- Critical errors (authentication failures, invalid configuration) indicate conditions that cannot be resolved by retry
- Recoverable errors (temporary rate limits, transient network issues) can be handled with retry/requeue
- Categorization enables appropriate error handling: stop chain for critical, continue with retry for recoverable
- Improves system resilience by not failing reconciliation on transient issues
- Handlers can return categorized errors that the chain can handle appropriately

**Error Handling in Classic Pattern**:
- Handlers return errors from `execute()` method
- Errors can be wrapped with error types: `CriticalError` or `RecoverableError`
- When a handler encounters a critical error, it returns the error and does not call `next.execute()`
- When a handler encounters a recoverable error, it can either:
  - Return the error (chain stops, but caller can requeue)
  - Set requeue flag in context and continue to next handler
- The controller (caller of first handler) handles error categorization and requeue logic

**Alternatives considered**:
- **Fail-fast (all errors stop chain)**: Rejected - Too strict, would fail on transient issues that could be retried
- **Continue on all errors**: Rejected - Too permissive, would continue on fatal errors that should stop reconciliation
- **Error wrapping with type checking**: Considered - More flexible but adds complexity. Categorization is simpler and sufficient

**Implementation approach**:
- Define error types: `CriticalError` and `RecoverableError` (or use error wrapping with `errors.Is()` checks)
- Handler `execute()` method returns error that can be checked for category
- Controller checks error category: critical stops reconciliation, recoverable allows requeue
- Recoverable errors can set requeue delay in context for controller to use

## Handler Execution Order

### Decision: Fixed order at compile time via setNext() calls

**Rationale**:
- Reconciliation steps have logical dependencies (fetch before validate, validate before scale)
- Fixed order is simpler, more predictable, and easier to test
- Compile-time ordering (via `setNext()` calls) catches ordering issues early
- No runtime configuration needed reduces complexity
- Order determined by sequence of `setNext()` calls during chain construction

**Handler Order**:
1. Fetch - Fetch scaler resource from Kubernetes API
2. Finalizer - Manage finalizer lifecycle (add/remove)
3. Authentication - Setup K8s client with authentication
4. Period Validation - Validate and determine current time period
5. Resource Scaling - Scale K8s resources based on period
6. Status Update - Update scaler status with operation results

**Alternatives considered**:
- **Runtime configuration**: Rejected - Adds unnecessary complexity, no use case for dynamic reordering
- **Dependency-based ordering**: Considered - More flexible but adds complexity. Fixed order is sufficient for known reconciliation flow

**Implementation approach**:
- Handlers registered in fixed order during chain construction via `setNext()` calls
- Order: Fetch → Finalizer → Authentication → Period Validation → Resource Scaling → Status Update
- Chain constructor method builds chain by calling `setNext()` on each handler in sequence
- First handler is returned/used to start chain execution

## Context Modification Strategy

### Decision: Later handlers overwrite earlier changes (last write wins)

**Rationale**:
- Simple and predictable behavior
- Later handlers have more complete information from earlier handlers
- Avoids complex merge logic that could introduce bugs
- Aligns with chain execution order where later handlers process results from earlier ones
- Handlers modify shared `ReconciliationContext` directly

**Alternatives considered**:
- **First write wins**: Rejected - Earlier handlers have less information, later handlers should take precedence
- **Merge/combine modifications**: Rejected - Adds complexity, merge logic varies by field type, error-prone

**Implementation approach**:
- Shared `ReconciliationContext` struct passed by reference through handler chain
- Handlers modify context fields directly
- Later handlers overwrite earlier changes when modifying same field
- No merge logic needed - simple assignment

## Requeue Behavior Strategy

### Decision: First handler's requeue delay takes precedence (earliest handler wins)

**Rationale**:
- Simple and predictable behavior
- First handler to request requeue likely has the most urgent need
- Avoids complex merge logic for determining delay
- Aligns with chain execution order

**Alternatives considered**:
- **Last handler's delay wins**: Rejected - Later handlers may have less urgent requeue needs
- **Shortest delay wins**: Considered - More optimal but adds complexity
- **Longest delay wins**: Rejected - Could delay urgent requeues unnecessarily

**Implementation approach**:
- `ReconciliationContext` has `RequeueAfter` field
- First handler to set `RequeueAfter` (non-zero value) wins
- Subsequent handlers can check if requeue already set and skip setting their own delay
- Controller uses `RequeueAfter` from context when returning `ctrl.Result`

## Handler Skipping Strategy

### Decision: Handler can skip all remaining handlers by not calling next.execute()

**Rationale**:
- Simple and explicit - handler simply doesn't call `next.execute()`
- Aligns with classic Chain of Responsibility pattern where handler decides whether to continue
- Handlers can set flags in context (e.g., `SkipRemaining`) to indicate early termination
- Controller can check context flags to determine if chain completed or was skipped

**Alternatives considered**:
- **Skip specific handlers**: Rejected - Adds complexity, no use case identified
- **Skip with condition**: Considered - Handlers already have access to context, can make decisions

**Implementation approach**:
- Handler checks condition (e.g., "no action" period detected)
- If condition met, handler sets `SkipRemaining` flag in context and returns without calling `next.execute()`
- Controller checks `SkipRemaining` flag to determine if chain completed early
- Handler can return success or error depending on whether skip is expected behavior

## Reconciliation Context Structure

### Decision: Shared context struct passed by reference through handler chain

**Rationale**:
- Enables handlers to share state and results
- Simple and efficient - single struct passed through chain
- Handlers can read from and write to context
- Context contains all necessary data: scaler resource, Kubernetes client, logger, results, flags

**Context Fields**:
- `Request` - Controller request (NamespacedName)
- `Client` - Kubernetes client for API operations
- `K8sClient` - K8s client for resource operations (set by auth handler)
- `DynamicClient` - Dynamic client for resource operations (set by auth handler)
- `Logger` - Structured logger
- `Scaler` - K8s scaler resource (set by fetch handler)
- `Secret` - Authentication secret (set by auth handler)
- `Period` - Current time period (set by period handler)
- `ResourceConfig` - Resource configuration (set by period handler)
- `SuccessResults` - Successful scaling operations (set by scaling handler)
- `FailedResults` - Failed scaling operations (set by scaling handler)
- `ShouldFinalize` - Flag indicating finalizer cleanup needed (set by finalizer handler)
- `SkipRemaining` - Flag indicating chain should stop early (set by any handler)
- `RequeueAfter` - Requeue delay duration (set by any handler)

**Alternatives considered**:
- **Immutable context**: Rejected - Would require creating new context for each handler, inefficient
- **Context interface**: Considered - More flexible but adds complexity. Struct is sufficient

**Implementation approach**:
- Define `ReconciliationContext` struct with all necessary fields
- Pass context by reference (pointer) through handler chain
- Handlers modify context fields directly
- Context initialized in controller before calling first handler

## Handler Interface Design

### Decision: Handler interface with execute() and setNext() methods

**Rationale**:
- Specification explicitly requires classic pattern with `execute()` and `setNext()` methods
- `execute()` method processes reconciliation step and returns error
- `setNext()` method establishes chain linkage
- Each handler maintains `next Handler` field
- Handlers call `next.execute()` to pass control

**Interface Definition**:
```go
type Handler interface {
    execute(ctx *ReconciliationContext) error
    setNext(next Handler)
}
```

**Handler Implementation Pattern**:
```go
type FetchHandler struct {
    next Handler
    // ... other fields (logger, client, etc.) ...
}

func (h *FetchHandler) execute(ctx *ReconciliationContext) error {
    // ... handler logic ...
    if h.next != nil {
        return h.next.execute(ctx)
    }
    return nil
}

func (h *FetchHandler) setNext(next Handler) {
    h.next = next
}
```

**Alternatives considered**:
- **Single Handle() method**: Rejected - Specification requires `execute()` and `setNext()` methods
- **Chain manager pattern**: Rejected - Specification requires handlers to maintain next reference

**Implementation approach**:
- Define `Handler` interface with `execute()` and `setNext()` methods
- Each handler struct implements interface and maintains `next Handler` field
- Handlers call `next.execute(ctx)` to pass control
- Chain constructed by calling `setNext()` on each handler in order

## Error Return Strategy

### Decision: Handlers return errors from execute() method, controller handles categorization

**Rationale**:
- Simple and explicit - handlers return errors, controller decides how to handle
- Aligns with classic pattern where handlers return errors
- Controller has full context to make requeue/retry decisions
- Handlers don't need to know about requeue logic

**Error Handling Flow**:
1. Handler encounters error during `execute()`
2. Handler categorizes error (critical vs recoverable) and wraps appropriately
3. Handler returns error (does not call `next.execute()` for critical errors)
4. Controller receives error from chain execution
5. Controller checks error category and handles appropriately:
   - Critical errors: Return error to controller-runtime (no requeue)
   - Recoverable errors: Return `ctrl.Result` with requeue delay

**Alternatives considered**:
- **Error in context**: Considered - Handlers set error in context, controller checks. Rejected - Less explicit, harder to trace
- **Panic/recover**: Rejected - Violates Go error handling best practices

**Implementation approach**:
- Handlers return `error` from `execute()` method
- Errors wrapped with `CriticalError` or `RecoverableError` types
- Controller checks error type using `errors.Is()` or type assertion
- Controller returns appropriate `ctrl.Result` based on error category

## Testing Strategy

### Decision: Unit test each handler independently with mocked dependencies

**Rationale**:
- Handlers are independent units that can be tested in isolation
- Mocked dependencies (Kubernetes client, logger, K8s client factory) enable fast unit tests
- Each handler test should execute in under 100ms (SC-004)
- Test coverage must reach at least 80% (SC-003)

**Testing Approach**:
- Each handler has corresponding test file: `*_handler_test.go`
- Tests use Ginkgo/Gomega for BDD-style assertions
- Mock interfaces for dependencies (Kubernetes client, logger, K8s client factory)
- Test handler behavior: success cases, error cases, edge cases
- Test chain linkage: verify `setNext()` and `next.execute()` calls
- Integration tests for full chain execution

**Alternatives considered**:
- **Integration tests only**: Rejected - Too slow, doesn't meet 100ms requirement per handler
- **End-to-end tests**: Rejected - Too slow, requires full cluster setup

**Implementation approach**:
- Unit tests for each handler with mocked dependencies
- Integration tests for full chain execution
- Test error propagation through chain
- Test context modification and sharing
- Test handler skipping behavior
- Measure test execution time and coverage
