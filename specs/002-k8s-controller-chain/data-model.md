# Data Model: Rewrite K8s Controller Using Chain of Responsibility Pattern

## Entities

### Handler Interface

**Purpose**: Defines the contract for all handlers in the Chain of Responsibility pattern.

**Interface Definition**:
```go
type Handler interface {
    execute(ctx *ReconciliationContext) error
    setNext(next Handler)
}
```

**Methods**:
- `execute(ctx *ReconciliationContext) error` - Processes a reconciliation step and passes control to next handler
- `setNext(next Handler)` - Establishes the next handler in the chain

**Relationships**:
- Each handler implementation maintains a reference to the next handler
- Handlers are linked together via `setNext()` calls during chain construction

**Validation Rules**:
- `execute()` must not be called with nil context
- `setNext()` can be called with nil to indicate end of chain
- Handlers should check if `next` is nil before calling `next.execute()`

### ReconciliationContext

**Purpose**: Shared state object passed through the handler chain containing all necessary data for reconciliation.

**Fields**:
- `Request` (ctrl.Request) - Controller request with NamespacedName
- `Client` (client.Client) - Kubernetes client for API operations
- `K8sClient` (kubernetes.Interface) - K8s client for resource operations (set by auth handler)
- `DynamicClient` (dynamic.Interface) - Dynamic client for resource operations (set by auth handler)
- `Logger` (*zerolog.Logger) - Structured logger
- `Scaler` (*kubecloudscalerv1alpha3.K8s) - K8s scaler resource (set by fetch handler)
- `Secret` (*corev1.Secret) - Authentication secret (set by auth handler, nil if not needed)
- `Period` (*period.Period) - Current time period (set by period handler)
- `ResourceConfig` (resources.Config) - Resource configuration (set by period handler)
- `SuccessResults` ([]common.ScalerStatusSuccess) - Successful scaling operations (set by scaling handler)
- `FailedResults` ([]common.ScalerStatusFailed) - Failed scaling operations (set by scaling handler)
- `ShouldFinalize` (bool) - Flag indicating finalizer cleanup needed (set by finalizer handler)
- `SkipRemaining` (bool) - Flag indicating chain should stop early (set by any handler)
- `RequeueAfter` (time.Duration) - Requeue delay duration (set by any handler, first handler wins)

**Relationships**:
- Passed by reference through handler chain
- Modified by handlers as reconciliation progresses
- Later handlers overwrite earlier changes (last write wins)

**Validation Rules**:
- Context must be initialized with at least `Request`, `Client`, and `Logger` before chain execution
- `Scaler` must be set by fetch handler before other handlers can use it
- `Period` must be set by period handler before scaling handler can use it

**State Transitions**:
- Initial: Only `Request`, `Client`, `Logger` set
- After Fetch: `Scaler` set
- After Finalizer: `ShouldFinalize` may be set
- After Auth: `K8sClient`, `DynamicClient`, `Secret` set
- After Period: `Period`, `ResourceConfig` set
- After Scaling: `SuccessResults`, `FailedResults` set
- After Status: Status updated in cluster

### Handler Implementations

#### FetchHandler

**Purpose**: Fetches the K8s scaler resource from the Kubernetes API.

**Dependencies**:
- Kubernetes client (from context)
- Logger (from context)

**Outputs**:
- Sets `ctx.Scaler` with fetched resource

**Error Handling**:
- Resource not found: Returns `CriticalError` (stops chain)
- Transient API errors: Returns `RecoverableError` (allows requeue)

#### FinalizerHandler

**Purpose**: Manages finalizer lifecycle for the K8s scaler resource.

**Dependencies**:
- Kubernetes client (from context)
- Logger (from context)
- Scaler resource (from context, set by fetch handler)

**Outputs**:
- Adds/removes finalizer on scaler resource
- Sets `ctx.ShouldFinalize` if deletion detected
- Sets `ctx.SkipRemaining` if finalizer already removed

**Error Handling**:
- Update failures: Returns `RecoverableError` (allows requeue)

#### AuthHandler

**Purpose**: Sets up K8s client with authentication.

**Dependencies**:
- Kubernetes client (from context)
- Logger (from context)
- Scaler resource (from context, set by fetch handler)

**Outputs**:
- Sets `ctx.K8sClient` and `ctx.DynamicClient` with authenticated clients
- Sets `ctx.Secret` if auth secret specified

**Error Handling**:
- Secret not found: Returns `CriticalError` (stops chain)
- Client creation failure: Returns `CriticalError` (stops chain)

#### PeriodHandler

**Purpose**: Validates and determines the current time period for scaling operations.

**Dependencies**:
- Kubernetes client (from context)
- Logger (from context)
- Scaler resource (from context, set by fetch handler)
- K8s clients (from context, set by auth handler)

**Outputs**:
- Sets `ctx.Period` with current time period
- Sets `ctx.ResourceConfig` with resource configuration
- Sets `ctx.SkipRemaining` if "no action" period detected
- Sets `ctx.RequeueAfter` for run-once periods

**Error Handling**:
- Invalid period configuration: Returns `CriticalError` (stops chain)
- Run-once period: Sets requeue delay and returns (stops chain with requeue)

#### ScalingHandler

**Purpose**: Scales K8s resources based on the determined period.

**Dependencies**:
- Kubernetes client (from context)
- Logger (from context)
- Scaler resource (from context, set by fetch handler)
- K8s clients (from context, set by auth handler)
- Period (from context, set by period handler)
- ResourceConfig (from context, set by period handler)

**Outputs**:
- Sets `ctx.SuccessResults` with successful scaling operations
- Sets `ctx.FailedResults` with failed scaling operations

**Error Handling**:
- Resource handler creation failures: Logs error, continues with other resources
- SetState failures: Logs error, continues with other resources
- Errors are collected in `FailedResults`, not returned (allows chain to continue)

#### StatusHandler

**Purpose**: Updates the scaler status with operation results.

**Dependencies**:
- Kubernetes client (from context)
- Logger (from context)
- Scaler resource (from context, set by fetch handler)
- SuccessResults (from context, set by scaling handler)
- FailedResults (from context, set by scaling handler)
- ShouldFinalize (from context, set by finalizer handler)

**Outputs**:
- Updates scaler status in cluster
- Removes finalizer if `ShouldFinalize` is true

**Error Handling**:
- Status update failures: Returns `RecoverableError` (allows requeue)
- Finalizer removal failures: Returns `RecoverableError` (allows requeue)

## Error Types

### CriticalError

**Purpose**: Indicates an error that prevents further reconciliation and requires immediate stop.

**Usage**:
- Authentication failures
- Invalid configuration
- Resource not found (when required)

**Behavior**:
- Handler returns `CriticalError` and does not call `next.execute()`
- Chain execution stops immediately
- Controller returns error to controller-runtime (no requeue)

### RecoverableError

**Purpose**: Indicates an error that may be resolved with a retry/requeue.

**Usage**:
- Temporary rate limits
- Transient network issues
- API update conflicts

**Behavior**:
- Handler returns `RecoverableError` and does not call `next.execute()`
- Chain execution stops
- Controller returns `ctrl.Result` with requeue delay

## Handler Chain Construction

**Purpose**: Links handlers together in fixed order via `setNext()` calls.

**Order**:
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

**Validation Rules**:
- Handlers must be linked in correct order
- Last handler's `next` should be nil
- First handler is used to start chain execution
