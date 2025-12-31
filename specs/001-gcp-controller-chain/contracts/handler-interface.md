# Contract: Handler Interface

## Purpose

Defines the contract that all reconciliation handlers must implement to participate in the Chain of Responsibility pattern.

## Interface Definition

```go
type Handler interface {
    Handle(ctx context.Context, req *ReconciliationContext) (*ReconciliationResult, error)
}
```

## Input: ReconciliationContext

**Type**: `*ReconciliationContext`

**Description**: Mutable context containing all state shared between handlers during reconciliation.

**Required Fields** (after specific handlers):
- `Scaler`: Must be populated by Fetch handler
- `GCPClient`: Must be populated by Authentication handler
- `Period`: Must be populated by Period Validation handler
- `SuccessResults`, `FailedResults`: Populated by Resource Scaling handler

**Modification Rules**:
- Handlers may modify any context field
- Later handlers overwrite earlier changes (last write wins)
- Context is passed by reference, modifications are visible to subsequent handlers

## Output: ReconciliationResult

**Type**: `*ReconciliationResult`

**Description**: Encapsulates the outcome of handler execution and next action.

**Fields**:
- `Continue` (bool): Whether chain should continue to next handler
- `Requeue` (bool): Whether reconciliation should be requeued
- `RequeueAfter` (time.Duration): Delay before requeue (if Requeue is true)
- `Error` (error): Error encountered (nil if no error)
- `ErrorCategory` (ErrorCategory): Category of error (Critical or Recoverable)

## Error Return

**Type**: `error`

**Behavior**:
- `nil`: Handler executed successfully, check ReconciliationResult for next action
- `CriticalError`: Chain must stop immediately, return error to caller
- `RecoverableError`: Chain may continue with requeue, error logged but not returned

## Success Conditions

Handler execution is successful when:
1. Handler completes its reconciliation step
2. Handler modifies context as needed
3. Handler returns `nil` error
4. Handler returns ReconciliationResult with appropriate Continue/Requeue flags

## Failure Conditions

Handler execution fails when:
1. Handler returns non-nil error
2. Error category determines chain behavior:
   - Critical: Chain stops, error returned
   - Recoverable: Chain continues with requeue, error logged

## Handler Responsibilities

Each handler MUST:
1. Process its specific reconciliation step
2. Modify context for subsequent handlers (if applicable)
3. Return appropriate ReconciliationResult
4. Return error only for actual failures (not for normal flow control)

Each handler MUST NOT:
1. Modify context fields it doesn't own (unless explicitly allowed)
2. Skip handlers directly (use SkipRemaining flag in context)
3. Return errors for normal conditions (use ReconciliationResult flags instead)

## Examples

### Success with Continue
```go
result := &ReconciliationResult{
    Continue: true,
    Requeue: false,
    Error: nil,
}
return result, nil
```

### Success with Requeue
```go
result := &ReconciliationResult{
    Continue: false,
    Requeue: true,
    RequeueAfter: 5 * time.Second,
    Error: nil,
}
return result, nil
```

### Critical Error (Stop Chain)
```go
return nil, NewCriticalError("authentication failed: %w", err)
```

### Recoverable Error (Continue with Requeue)
```go
result := &ReconciliationResult{
    Continue: true,
    Requeue: true,
    RequeueAfter: 10 * time.Second,
    ErrorCategory: RecoverableError,
}
return result, NewRecoverableError("rate limit exceeded: %w", err)
```
