# K8s Scaler Controller: Architecture and Development Guide

This document provides a comprehensive overview of the K8s Scaler Controller, its architecture, the Chain of Responsibility pattern used for reconciliation, and guidelines for development, testing, and extension.

## 1. Controller Overview

The K8s Scaler Controller manages the lifecycle of Kubernetes resources (e.g., Deployments, StatefulSets) by scaling them up or down based on configured time periods. It watches `K8s` custom resources in Kubernetes and reconciles their desired state.

### Key Responsibilities

- Fetching `K8s` custom resources
- Managing Kubernetes finalizers for proper cleanup
- Authenticating with K8s API (local or remote clusters)
- Validating time periods and determining scaling actions
- Scaling K8s resources (Deployments, StatefulSets, HPAs)
- Updating resource status with operation results

## 2. Chain of Responsibility Architecture

The controller uses the **Chain of Responsibility pattern** from [refactoring.guru](https://refactoring.guru/design-patterns/chain-of-responsibility/go/example) where handlers have `Execute()` and `SetNext()` methods.

### Pattern Structure

```
Controller.Reconcile()
    │
    ▼
┌─────────────┐    ┌──────────────────┐    ┌─────────────┐
│FetchHandler │───▶│FinalizerHandler │───▶│ AuthHandler │
└─────────────┘    └──────────────────┘    └─────────────┘
                                                  │
                                                  ▼
                   ┌──────────────┐    ┌────────────────┐
                   │StatusHandler │◀───│ScalingHandler │
                   └──────────────┘    └────────────────┘
                          ▲                     ▲
                          │                     │
                   ┌──────────────┐             │
                   │PeriodHandler │─────────────┘
                   └──────────────┘
```

### Handler Execution Order

1. **FetchHandler**: Fetches the K8s Scaler resource from cluster
2. **FinalizerHandler**: Manages finalizer lifecycle (add/remove)
3. **AuthHandler**: Sets up K8s client with authentication
4. **PeriodHandler**: Validates periods and determines current period
5. **ScalingHandler**: Scales K8s resources based on period
6. **StatusHandler**: Updates scaler status with operation results

### Chain Linking

Handlers are linked via `SetNext()` calls in `initializeChain()`:

```go
func (r *ScalerReconciler) initializeChain() service.Handler {
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

    return fetchHandler
}
```

## 3. Error Handling

Errors are categorized for precise control:

| Error Type | Behavior | Example |
|------------|----------|---------|
| **CriticalError** | Stop chain, return error | Auth failure, resource not found |
| **RecoverableError** | Stop chain, requeue | Transient network error, API rate limit |
| **nil** | Continue to next handler | Successful operation |

### SkipRemaining Flag

Handlers can set `ctx.SkipRemaining = true` to stop the chain early without an error (e.g., "no action" period detected).

## 4. ReconciliationContext

The shared context passed through the chain:

```go
type ReconciliationContext struct {
    Ctx            context.Context        // Go context from Reconcile method
    Request        ctrl.Request           // Controller request
    Client         client.Client          // K8s API client
    K8sClient      kubernetes.Interface   // Typed K8s client
    DynamicClient  dynamic.Interface      // Dynamic K8s client
    Logger         *zerolog.Logger        // Structured logger
    Scaler         *K8s                   // Scaler resource
    Period         *period.Period         // Current period
    ResourceConfig resources.Config       // Resource config
    SuccessResults []ScalerStatusSuccess  // Scaling successes
    FailedResults  []ScalerStatusFailed   // Scaling failures
    ShouldFinalize bool                   // Finalizer cleanup needed
    SkipRemaining  bool                   // Stop chain early
    RequeueAfter   time.Duration          // Requeue delay
}
```

## 5. Directory Structure

```
internal/controller/k8s/
├── scaler_controller.go          # Controller with Reconcile and initializeChain
├── scaler_controller_test.go     # Controller tests
├── suite_test.go                 # Test suite setup
└── service/                      # Service layer (Chain of Responsibility)
    ├── interfaces.go             # Handler interface
    ├── context.go                # ReconciliationContext
    ├── errors.go                 # Error types
    ├── chain_test.go             # Chain integration tests
    └── handlers/                 # Handler implementations
        ├── fetch_handler.go
        ├── finalizer_handler.go
        ├── auth_handler.go
        ├── period_handler.go
        ├── scaling_handler.go
        ├── status_handler.go
        └── *_test.go             # Handler unit tests
```

## 6. Extending the Controller

To add a new reconciliation step:

1. Create a new handler in `service/handlers/`
2. Implement `Execute()` and `SetNext()` methods
3. Add the handler to `initializeChain()` in the appropriate order
4. Link the handler using `SetNext()` calls
5. Write tests following TDD approach

See `service/handlers/README.md` for detailed handler development guidelines.

## 7. Testing

```bash
# Run all controller tests
go test ./internal/controller/k8s/...

# Run with coverage
go test -coverprofile=coverage.out ./internal/controller/k8s/...
go tool cover -html=coverage.out

# Run handler tests only
go test ./internal/controller/k8s/service/handlers/...
```

## 8. Benefits of Chain Pattern

| Benefit | Description |
|---------|-------------|
| **Single Responsibility** | Each handler has one job |
| **Testability** | Handlers tested independently with mocks |
| **Extensibility** | Add handlers without modifying existing code |
| **Maintainability** | Clear separation of concerns |
| **Readability** | Reconciliation flow is explicit |
