# Quickstart: GCP Controller Chain of Responsibility Refactoring

## Overview

This guide explains the refactored GCP controller architecture using the Chain of Responsibility pattern and how to work with handlers.

## Architecture

The GCP controller reconciliation flow is now implemented as a chain of handlers:

```
Reconcile() → Chain.Execute() → [Fetch → Finalizer → Auth → Period → Scaling → Status]
```

Each handler processes a specific reconciliation step and passes control to the next handler.

## Handler Chain

### Handler Execution Order

1. **Fetch Handler**: Retrieves the GCP scaler resource from Kubernetes
2. **Finalizer Handler**: Manages finalizer lifecycle (add/remove)
3. **Authentication Handler**: Sets up GCP client with authentication
4. **Period Validation Handler**: Validates and determines current time period
5. **Resource Scaling Handler**: Scales GCP resources based on period
6. **Status Update Handler**: Updates scaler status with operation results

### Working with Handlers

#### Creating a New Handler

```go
// internal/controller/gcp/service/handlers/my_handler.go
package handlers

import (
    "context"
    "github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service"
)

type MyHandler struct {
    // Dependencies injected via constructor
}

func (h *MyHandler) Handle(ctx context.Context, req *service.ReconciliationContext) (*service.ReconciliationResult, error) {
    // 1. Process reconciliation step
    // 2. Modify context for subsequent handlers
    // 3. Return result indicating next action

    return &service.ReconciliationResult{
        Continue: true,
        Requeue: false,
    }, nil
}
```

#### Adding Handler to Chain

```go
// internal/controller/gcp/service/chain.go
func NewChain(logger *zerolog.Logger) *Chain {
    return &Chain{
        handlers: []Handler{
            NewFetchHandler(client),
            NewFinalizerHandler(client),
            NewAuthHandler(client),
            NewPeriodHandler(),
            NewScalingHandler(),
            NewStatusHandler(client),
            // Add your handler here in correct order
        },
        logger: logger,
    }
}
```

## Reconciliation Context

The context is shared between all handlers and contains:

- `Scaler`: The GCP scaler resource
- `GCPClient`: GCP API client
- `Period`: Current time period configuration
- `SuccessResults`, `FailedResults`: Scaling operation results
- `SkipRemaining`: Flag to skip remaining handlers

### Modifying Context

```go
// In your handler
req.Scaler.Status.Comments = ptr.To("processed by my handler")
req.SuccessResults = append(req.SuccessResults, result)
```

**Note**: Later handlers overwrite earlier changes (last write wins).

## Error Handling

### Critical Errors

Stop chain execution immediately:

```go
return nil, service.NewCriticalError("authentication failed: %w", err)
```

### Recoverable Errors

Allow chain to continue with requeue:

```go
result := &service.ReconciliationResult{
    Continue: true,
    Requeue: true,
    RequeueAfter: 10 * time.Second,
}
return result, service.NewRecoverableError("rate limit: %w", err)
```

## Testing Handlers

### Unit Test Example

```go
// internal/controller/gcp/service/handlers/my_handler_test.go
package handlers

import (
    "context"
    "github.com/onsi/ginkgo/v2"
    "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("MyHandler", func() {
    var handler *MyHandler
    var mockClient *mocks.MockClient

    ginkgo.BeforeEach(func() {
        mockClient = mocks.NewMockClient()
        handler = NewMyHandler(mockClient)
    })

    ginkgo.It("should process reconciliation step", func() {
        req := &service.ReconciliationContext{
            Scaler: &gcpv1alpha3.Gcp{},
        }

        result, err := handler.Handle(context.Background(), req)

        gomega.Expect(err).To(gomega.BeNil())
        gomega.Expect(result.Continue).To(gomega.BeTrue())
    })
})
```

## Chain Execution Flow

### Normal Flow

1. Chain creates ReconciliationContext
2. Executes handlers in order
3. Each handler modifies context
4. Final handler updates status
5. Returns success result

### Error Flow

1. Handler encounters critical error
2. Chain stops immediately
3. Returns error to caller
4. Context modifications preserved up to error point

### Requeue Flow

1. Handler requests requeue (recoverable error or normal condition)
2. Chain continues execution
3. First requeue delay is tracked
4. Returns result with requeue delay

### Skip Flow

1. Handler sets `SkipRemaining = true` in context
2. Chain stops execution
3. Returns current result
4. Remaining handlers not executed

## Migration from Current Implementation

The refactored controller maintains 100% backward compatibility:

- Same API contracts
- Same reconciliation behavior
- Same error handling
- Same status updates

**No changes required** to existing GCP scaler resources or configurations.

## Debugging

### Enable Handler Logging

Handlers log execution with structured logging:

```go
h.logger.Info().
    Str("handler", "MyHandler").
    Str("scaler", req.Scaler.Name).
    Msg("processing reconciliation step")
```

### Check Handler Execution

Chain logs handler execution:

```
INFO handler execution started handler=FetchHandler
INFO handler execution completed handler=FetchHandler duration=10ms
INFO handler execution started handler=FinalizerHandler
...
```

## Best Practices

1. **Single Responsibility**: Each handler should handle one reconciliation step
2. **Context Modification**: Only modify context fields your handler owns
3. **Error Categorization**: Use appropriate error category (critical vs recoverable)
4. **Testing**: Write unit tests for each handler with mocked dependencies
5. **Logging**: Use structured logging with appropriate levels

## Common Patterns

### Early Return (No Action)

```go
if req.Period.Name == "noaction" {
    req.SkipRemaining = true
    return &service.ReconciliationResult{
        Continue: false,
        Requeue: true,
        RequeueAfter: utils.ReconcileSuccessDuration,
    }, nil
}
```

### Conditional Processing

```go
if req.ShouldFinalize {
    // Handle finalizer cleanup
    return &service.ReconciliationResult{Continue: true}, nil
}
// Continue normal flow
```

### Resource Processing Loop

```go
for _, resource := range resourceList {
    result, err := processResource(resource)
    if err != nil {
        req.FailedResults = append(req.FailedResults, result)
        continue
    }
    req.SuccessResults = append(req.SuccessResults, result)
}
```
