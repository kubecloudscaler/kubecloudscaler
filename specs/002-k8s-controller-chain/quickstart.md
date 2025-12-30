# Quickstart: K8s Controller Chain of Responsibility Pattern

## Overview

This guide demonstrates how to use the refactored K8s controller with the Chain of Responsibility pattern. The controller uses the classic pattern from refactoring.guru where handlers have `execute()` and `setNext()` methods.

## Architecture

The K8s controller reconciliation is handled by a chain of handlers:

1. **FetchHandler** - Fetches scaler resource from Kubernetes API
2. **FinalizerHandler** - Manages finalizer lifecycle
3. **AuthHandler** - Sets up K8s client with authentication
4. **PeriodHandler** - Validates and determines current time period
5. **ScalingHandler** - Scales K8s resources based on period
6. **StatusHandler** - Updates scaler status with operation results

## Handler Chain Construction

The handler chain is constructed by linking handlers via `setNext()` calls:

```go
// Create handlers
fetchHandler := handlers.NewFetchHandler()
finalizerHandler := handlers.NewFinalizerHandler()
authHandler := handlers.NewAuthHandler()
periodHandler := handlers.NewPeriodHandler()
scalingHandler := handlers.NewScalingHandler()
statusHandler := handlers.NewStatusHandler()

// Link handlers in order
fetchHandler.setNext(finalizerHandler)
finalizerHandler.setNext(authHandler)
authHandler.setNext(periodHandler)
periodHandler.setNext(scalingHandler)
scalingHandler.setNext(statusHandler) // Last handler, next is nil

// Start chain execution
reconCtx := &service.ReconciliationContext{
    Request: req,
    Client:  r.Client,
    Logger:  r.Logger,
}
return fetchHandler.execute(reconCtx)
```

## Controller Integration

The controller's `Reconcile` method delegates to the handler chain:

```go
func (r *ScalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // Initialize chain if not set
    if r.Chain == nil {
        r.Chain = r.initializeChain()
    }

    // Create reconciliation context
    reconCtx := &service.ReconciliationContext{
        Request: req,
        Client:  r.Client,
        Logger:  r.Logger,
    }

    // Execute handler chain
    err := r.Chain.Execute(ctx, reconCtx)
    if err != nil {
        // Handle error categorization
        if service.IsCriticalError(err) {
            return ctrl.Result{}, err
        }
        // Recoverable error - requeue
        return ctrl.Result{RequeueAfter: reconCtx.RequeueAfter}, nil
    }

    // Check if requeue requested
    if reconCtx.RequeueAfter > 0 {
        return ctrl.Result{RequeueAfter: reconCtx.RequeueAfter}, nil
    }

    return ctrl.Result{}, nil
}
```

## Adding a New Handler

To add a new handler to the chain:

1. **Create handler struct**:
```go
type ValidationHandler struct {
    next Handler
    // ... other fields ...
}

func NewValidationHandler() Handler {
    return &ValidationHandler{}
}
```

2. **Implement Handler interface**:
```go
func (h *ValidationHandler) execute(ctx *ReconciliationContext) error {
    // ... validation logic ...
    if err != nil {
        return NewCriticalError(err)
    }

    if h.next != nil {
        return h.next.execute(ctx)
    }
    return nil
}

func (h *ValidationHandler) setNext(next Handler) {
    h.next = next
}
```

3. **Add to chain construction**:
```go
validationHandler := handlers.NewValidationHandler()
fetchHandler.setNext(validationHandler)
validationHandler.setNext(finalizerHandler)
// ... rest of chain ...
```

## Error Handling

Handlers categorize errors as critical or recoverable:

```go
// Critical error - stops chain immediately
if err != nil {
    return NewCriticalError(err)
}

// Recoverable error - allows requeue
if err != nil {
    return NewRecoverableError(err)
}
```

The controller handles error categorization:

```go
err := fetchHandler.execute(reconCtx)
if err != nil {
    if service.IsCriticalError(err) {
        return ctrl.Result{}, err // No requeue
    }
    // Recoverable - requeue
    return ctrl.Result{RequeueAfter: utils.ReconcileErrorDuration}, nil
}
```

## Context Modification

Handlers modify the shared `ReconciliationContext`:

```go
// FetchHandler sets Scaler
ctx.Scaler = scaler

// PeriodHandler sets Period
ctx.Period = period

// ScalingHandler sets results
ctx.SuccessResults = success
ctx.FailedResults = failed
```

Later handlers overwrite earlier changes (last write wins).

## Skipping Remaining Handlers

Handlers can skip remaining handlers by not calling `next.execute()`:

```go
// PeriodHandler detects "no action" period
if ctx.Period.Name == "noaction" && ctx.Scaler.Status.CurrentPeriod.Name == "noaction" {
    ctx.SkipRemaining = true
    return nil // Don't call next.execute()
}
```

## Requeue Behavior

Handlers can request requeue by setting `RequeueAfter`:

```go
// PeriodHandler for run-once period
if errors.Is(err, utils.ErrRunOncePeriod) {
    ctx.RequeueAfter = time.Until(period.GetEndTime.Add(RequeueDelaySeconds * time.Second))
    return nil // Don't call next.execute()
}
```

First handler to set `RequeueAfter` wins (earliest handler takes precedence).

## Testing

### Unit Testing Handlers

Each handler can be tested independently:

```go
func TestFetchHandler(t *testing.T) {
    handler := handlers.NewFetchHandler()
    reconCtx := &service.ReconciliationContext{
        Request: ctrl.Request{...},
        Client:  mockClient,
        Logger:  &logger,
    }

    err := handler.execute(reconCtx)
    // ... assertions ...
}
```

### Testing Handler Chain

Test full chain execution:

```go
func TestHandlerChain(t *testing.T) {
    // Build chain
    fetchHandler := handlers.NewFetchHandler()
    // ... link handlers ...

    // Execute chain
    err := fetchHandler.execute(reconCtx)
    // ... assertions ...
}
```

## Best Practices

1. **Dependency Injection**: Handlers should receive dependencies via constructor, not global state
2. **Error Categorization**: Always categorize errors as critical or recoverable
3. **Context Validation**: Check required context fields before using them
4. **Chain Ordering**: Maintain fixed handler order (fetch → finalizer → auth → period → scaling → status)
5. **Logging**: Use structured logging with appropriate levels (debug, info, warn, error)
6. **Testing**: Write unit tests for each handler with mocked dependencies

## Migration from Monolithic Implementation

The refactored controller maintains 100% backward compatibility:

- Existing K8s scaler resources continue to work without changes
- API contracts remain unchanged
- Status updates are identical to previous implementation
- All existing tests pass with refactored implementation

No migration steps are required - the refactoring is transparent to users.
