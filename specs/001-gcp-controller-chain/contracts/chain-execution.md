# Contract: Chain Execution

## Purpose

Defines the contract for how the handler chain executes and manages handler execution flow.

## Chain Interface

```go
type Chain interface {
    Execute(ctx context.Context, req *ReconciliationContext) (ctrl.Result, error)
}
```

## Execution Flow

1. **Initialize Context**: Create ReconciliationContext with Request
2. **Execute Handlers Sequentially**: For each handler in fixed order:
   - Call `handler.Handle(ctx, req)`
   - Check result and error
   - Apply result actions (continue, stop, requeue, skip)
3. **Return Final Result**: Return ctrl.Result and error

## Handler Execution Order

Fixed order (compile-time):
1. Fetch Handler
2. Finalizer Handler
3. Authentication Handler
4. Period Validation Handler
5. Resource Scaling Handler
6. Status Update Handler

## Execution Rules

### Continue to Next Handler
- **Condition**: `result.Continue == true` AND `error == nil`
- **Action**: Execute next handler in sequence
- **Context**: Pass modified context to next handler

### Stop Chain (Critical Error)
- **Condition**: `error != nil` AND `error` is `CriticalError`
- **Action**: Stop chain execution immediately
- **Return**: Return error to caller, no reconciliation result

### Stop Chain (Handler Request)
- **Condition**: `result.Continue == false` AND `error == nil`
- **Action**: Stop chain execution
- **Return**: Return ctrl.Result (may include requeue)

### Continue with Requeue (Recoverable Error)
- **Condition**: `error != nil` AND `error` is `RecoverableError`
- **Action**: Continue to next handler, but requeue after completion
- **Context**: Pass modified context to next handler
- **Return**: Requeue delay from first handler that requested requeue

### Skip Remaining Handlers
- **Condition**: `req.SkipRemaining == true`
- **Action**: Stop chain execution immediately
- **Return**: Return current ctrl.Result (may include requeue)

## Requeue Behavior

- **First Requeue Wins**: First handler's requeue delay takes precedence
- **Subsequent Requeues Ignored**: If requeue already requested, ignore later requests
- **Delay Tracking**: Chain tracks first `RequeueAfter` value encountered

## Error Handling

### Critical Errors
- Stop chain immediately
- Return error to caller
- No reconciliation result returned
- Context modifications up to error point are preserved

### Recoverable Errors
- Log error with appropriate level
- Continue chain execution
- Request requeue with appropriate delay
- Context modifications continue

## Success Conditions

Chain execution succeeds when:
1. All handlers execute successfully
2. No critical errors encountered
3. Final handler updates status successfully
4. Returns ctrl.Result with appropriate requeue delay

## Failure Conditions

Chain execution fails when:
1. Critical error encountered in any handler
2. Context validation fails (e.g., required field missing)
3. Final status update fails (may be recoverable with requeue)

## Return Values

### Success
```go
return ctrl.Result{
    RequeueAfter: time.Duration, // If requeue requested, else 0
    Requeue: false,
}, nil
```

### Requeue Requested
```go
return ctrl.Result{
    RequeueAfter: firstRequeueDelay,
    Requeue: false,
}, nil
```

### Critical Error
```go
return ctrl.Result{}, criticalError
```

## Context Lifecycle

1. **Creation**: Chain creates context with Request
2. **Modification**: Each handler modifies context as needed
3. **Validation**: Chain validates context after each handler (optional)
4. **Finalization**: Context used for final status update

## Observability

Chain MUST:
- Log handler execution start/end
- Log errors with appropriate level
- Log requeue requests
- Log skip requests
- Use structured logging (zerolog) with context fields
