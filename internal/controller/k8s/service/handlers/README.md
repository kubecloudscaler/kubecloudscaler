# K8s Controller Handlers: Development and Testing Guide

This document provides guidelines for developing, testing, and extending handlers within the `internal/controller/k8s/service/handlers/` directory. These handlers implement the Chain of Responsibility pattern for the K8s Scaler reconciliation logic.

## 1. Handler Architecture Overview

The K8s controller's `Reconcile` function uses a **Chain of Responsibility** pattern following the classic refactoring.guru Go pattern. Reconciliation is broken down into a sequence of discrete, single-responsibility handlers linked via `SetNext()` calls.

### Key Components

- **`service.Handler` Interface**: Defines the contract for all handlers with `Execute()` and `SetNext()` methods.
- **`service.ReconciliationContext`**: A shared mutable struct passed through the chain, containing all necessary state.
- **Handler Implementations**: Individual handlers for each reconciliation step (fetch, finalizer, auth, period, scaling, status).

### Handler Execution Flow

The handler chain executes in a fixed, predefined order:

1. **`FetchHandler`**: Fetches the `K8s` Scaler resource from the Kubernetes API
2. **`FinalizerHandler`**: Manages the `kubecloudscaler.cloud/finalizer` on the `K8s` resource
3. **`AuthHandler`**: Sets up the K8s client, handling authentication secrets
4. **`PeriodHandler`**: Validates configured periods and determines the active scaling period
5. **`ScalingHandler`**: Performs the actual scaling operations on K8s resources
6. **`StatusHandler`**: Updates the `K8s` resource's status in Kubernetes

### Error Handling Strategy

Errors are categorized to provide fine-grained control over reconciliation flow:

- **`service.CriticalError`**: Indicates a non-recoverable error. Handler stops chain execution immediately.
- **`service.RecoverableError`**: Indicates a transient error. Chain may stop with requeue for retry.

### Chain Construction (refactoring.guru pattern)

Handlers are linked via `SetNext()` calls during chain construction:

```go
fetchHandler := handlers.NewFetchHandler()
finalizerHandler := handlers.NewFinalizerHandler()
authHandler := handlers.NewAuthHandler()
periodHandler := handlers.NewPeriodHandler()
scalingHandler := handlers.NewScalingHandler()
statusHandler := handlers.NewStatusHandler()

fetchHandler.SetNext(finalizerHandler)
finalizerHandler.SetNext(authHandler)
authHandler.SetNext(periodHandler)
periodHandler.SetNext(scalingHandler)
scalingHandler.SetNext(statusHandler)
// statusHandler.next is nil (end of chain)

// Start chain execution
return fetchHandler.Execute(ctx)
```

## 2. Developing a New Handler

To add a new reconciliation step:

1. **Create a new Go file**: `internal/controller/k8s/service/handlers/your_handler.go`

2. **Define your handler struct**:

```go
package handlers

import (
    "github.com/kubecloudscaler/kubecloudscaler/internal/controller/k8s/service"
)

type YourHandler struct {
    next service.Handler
}

func NewYourHandler() service.Handler {
    return &YourHandler{}
}

func (h *YourHandler) Execute(ctx *service.ReconciliationContext) error {
    ctx.Logger.Debug().Msg("executing YourHandler")

    // Your business logic here
    // Access shared state via ctx (e.g., ctx.Scaler, ctx.K8sClient)
    // Modify ctx as needed for subsequent handlers

    // Example: Check a condition and stop chain early
    if someCondition {
        ctx.SkipRemaining = true
        return nil
    }

    // Example: Return a critical error
    if someCriticalCondition {
        return service.NewCriticalError(errors.New("critical condition met"))
    }

    // Call next handler in chain
    if h.next != nil && !ctx.SkipRemaining {
        return h.next.Execute(ctx)
    }
    return nil
}

func (h *YourHandler) SetNext(next service.Handler) {
    h.next = next
}
```

3. **Register your handler**: Add an instance of `YourHandler` to the chain in `internal/controller/k8s/scaler_controller.go`'s `initializeChain()` method.

## 3. Testing Handlers (TDD Approach)

All handlers MUST be unit tested independently with mocked dependencies. Follow TDD:

1. **Write a failing test**: Create a test file (`your_handler_test.go`)
2. **Implement the handler**: Write code to make the test pass
3. **Refactor**: Improve code quality while maintaining passing tests

### Test Structure

- Use `Ginkgo` and `Gomega` for BDD-style tests
- Each handler should have its own `_test.go` file
- Use `zerolog.Nop()` for the logger in tests
- Mock Kubernetes API using `sigs.k8s.io/controller-runtime/pkg/client/fake`

### Example Handler Test

```go
var _ = Describe("FetchHandler", func() {
    var (
        handler  service.Handler
        reconCtx *service.ReconciliationContext
        logger   zerolog.Logger
        scheme   *runtime.Scheme
    )

    BeforeEach(func() {
        handler = handlers.NewFetchHandler()
        logger = zerolog.Nop()
        scheme = runtime.NewScheme()
        Expect(kubecloudscalerv1alpha3.AddToScheme(scheme)).To(Succeed())

        reconCtx = &service.ReconciliationContext{
            Request: ctrl.Request{
                NamespacedName: types.NamespacedName{
                    Name:      "test-scaler",
                    Namespace: "default",
                },
            },
            Logger: &logger,
        }
    })

    Context("When the Scaler resource exists", func() {
        It("should fetch the scaler and add it to the context", func() {
            scaler := &kubecloudscalerv1alpha3.K8s{...}
            reconCtx.Client = fake.NewClientBuilder().WithScheme(scheme).WithObjects(scaler).Build()

            err := handler.Execute(reconCtx)

            Expect(err).ToNot(HaveOccurred())
            Expect(reconCtx.Scaler).ToNot(BeNil())
        })

        It("should complete in under 100ms", func() {
            // ... performance test
        })
    })
})
```

### Running Tests

```bash
# Run all handler tests
go test ./internal/controller/k8s/service/handlers/...

# Run with coverage
go test -coverprofile=coverage.out ./internal/controller/k8s/service/handlers/...
go tool cover -html=coverage.out
```

### Metrics

- **Test Coverage**: 69.2% (target: 80%)
- **Average Handler Execution Time**: ~3ms
- **Pass Rate**: 100%

## 4. Extending the Chain

The Chain of Responsibility pattern makes it easy to extend the reconciliation logic:

1. **Create a new handler**: Follow the "Developing a New Handler" guide
2. **Add to `initializeChain()`**: Insert your handler into the chain at the appropriate position
3. **Link handlers**: Use `SetNext()` to link your handler into the chain
4. **No modification to existing handlers**: Existing handlers don't need to change

This modular design ensures the controller remains maintainable and adaptable.
