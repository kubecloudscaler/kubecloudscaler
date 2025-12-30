# Handler Interface Contract

## Overview

The Handler interface defines the contract for all handlers in the Chain of Responsibility pattern for K8s controller reconciliation.

## Interface Definition

```go
type Handler interface {
    execute(ctx *ReconciliationContext) error
    setNext(next Handler)
}
```

## Method Specifications

### execute(ctx *ReconciliationContext) error

**Purpose**: Processes a reconciliation step and passes control to the next handler in the chain.

**Parameters**:
- `ctx *ReconciliationContext` - Shared context containing all reconciliation state (must not be nil)

**Returns**:
- `error` - Error encountered during execution:
  - `nil` - Success, chain continues (if `next.execute()` called)
  - `CriticalError` - Critical error, chain stops immediately
  - `RecoverableError` - Recoverable error, chain stops but can be requeued

**Behavior**:
- Handler processes its reconciliation step
- Handler may modify `ctx` to share state with subsequent handlers
- Handler calls `ctx.next.execute(ctx)` to pass control to next handler (if not stopping)
- Handler should check if `next` is nil before calling `next.execute()`
- Handler should not call `next.execute()` if it encounters a critical error

**Error Handling**:
- Critical errors: Return error, do not call `next.execute()`
- Recoverable errors: Return error, do not call `next.execute()` (controller will requeue)
- Success: Call `next.execute(ctx)` if `next` is not nil, return its result

**Preconditions**:
- `ctx` must not be nil
- Required context fields must be set by previous handlers (e.g., `Scaler` must be set by fetch handler)

**Postconditions**:
- Handler-specific context fields are set (e.g., `Period` set by period handler)
- If handler succeeds and `next` is not nil, `next.execute()` is called
- If handler fails, error is returned and chain stops

### setNext(next Handler)

**Purpose**: Establishes the next handler in the chain.

**Parameters**:
- `next Handler` - Next handler in the chain (can be nil to indicate end of chain)

**Returns**:
- None

**Behavior**:
- Sets the `next` field of the handler
- Can be called multiple times (last call wins)
- Can be called with `nil` to indicate end of chain

**Preconditions**:
- None

**Postconditions**:
- Handler's `next` field is set to the provided handler

## Handler Implementation Requirements

All handler implementations must:

1. Implement the `Handler` interface
2. Maintain a `next Handler` field
3. Call `next.execute(ctx)` to pass control (if not stopping)
4. Check if `next` is nil before calling `next.execute()`
5. Return errors appropriately (critical vs recoverable)
6. Modify context fields as needed for subsequent handlers
7. Use dependency injection via constructor (no global state)

## Example Implementation

```go
type FetchHandler struct {
    next Handler
    // ... other fields (logger, client, etc.) ...
}

func (h *FetchHandler) execute(ctx *ReconciliationContext) error {
    // ... handler logic ...
    if err != nil {
        return NewCriticalError(err)
    }

    if h.next != nil {
        return h.next.execute(ctx)
    }
    return nil
}

func (h *FetchHandler) setNext(next Handler) {
    h.next = next
}
```

## Chain Construction Contract

Handlers must be linked in the following order:

1. FetchHandler
2. FinalizerHandler
3. AuthHandler
4. PeriodHandler
5. ScalingHandler
6. StatusHandler

**Construction Pattern**:
```go
fetchHandler := NewFetchHandler()
finalizerHandler := NewFinalizerHandler()
authHandler := NewAuthHandler()
periodHandler := NewPeriodHandler()
scalingHandler := NewScalingHandler()
statusHandler := NewStatusHandler()

fetchHandler.setNext(finalizerHandler)
finalizerHandler.setNext(authHandler)
authHandler.setNext(periodHandler)
periodHandler.setNext(scalingHandler)
scalingHandler.setNext(statusHandler) // Last handler, next is nil

// Start chain execution
return fetchHandler.execute(ctx)
```
