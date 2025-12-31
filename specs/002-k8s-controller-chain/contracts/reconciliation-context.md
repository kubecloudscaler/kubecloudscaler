# ReconciliationContext Contract

## Overview

The `ReconciliationContext` is a shared state object passed through the handler chain containing all necessary data for reconciliation.

## Structure Definition

```go
type ReconciliationContext struct {
    Request        ctrl.Request
    Client         client.Client
    K8sClient      kubernetes.Interface
    DynamicClient  dynamic.Interface
    Logger         *zerolog.Logger
    Scaler         *kubecloudscalerv1alpha3.K8s
    Secret         *corev1.Secret
    Period         *period.Period
    ResourceConfig resources.Config
    SuccessResults []common.ScalerStatusSuccess
    FailedResults  []common.ScalerStatusFailed
    ShouldFinalize bool
    SkipRemaining  bool
    RequeueAfter   time.Duration
}
```

## Field Specifications

### Request (ctrl.Request)

**Purpose**: Controller request with NamespacedName identifying the resource to reconcile.

**Set By**: Controller (before chain execution)

**Used By**: All handlers

**Modifications**: None (read-only)

### Client (client.Client)

**Purpose**: Kubernetes client for API operations (fetching/updating resources).

**Set By**: Controller (before chain execution)

**Used By**: All handlers

**Modifications**: None (read-only)

### K8sClient (kubernetes.Interface)

**Purpose**: K8s client for resource operations (scaling resources).

**Set By**: AuthHandler

**Used By**: PeriodHandler, ScalingHandler

**Modifications**: Set once by AuthHandler, read by subsequent handlers

### DynamicClient (dynamic.Interface)

**Purpose**: Dynamic client for resource operations (scaling resources).

**Set By**: AuthHandler

**Used By**: PeriodHandler, ScalingHandler

**Modifications**: Set once by AuthHandler, read by subsequent handlers

### Logger (*zerolog.Logger)

**Purpose**: Structured logger for handler execution logging.

**Set By**: Controller (before chain execution)

**Used By**: All handlers

**Modifications**: None (read-only)

### Scaler (*kubecloudscalerv1alpha3.K8s)

**Purpose**: K8s scaler resource being reconciled.

**Set By**: FetchHandler

**Used By**: All subsequent handlers

**Modifications**:
- Set by FetchHandler
- Updated by FinalizerHandler (finalizer changes)
- Updated by StatusHandler (status updates)

### Secret (*corev1.Secret)

**Purpose**: Authentication secret for remote cluster access (nil if not needed).

**Set By**: AuthHandler

**Used By**: AuthHandler only

**Modifications**: Set once by AuthHandler, read by AuthHandler

### Period (*period.Period)

**Purpose**: Current time period for scaling operations.

**Set By**: PeriodHandler

**Used By**: ScalingHandler, StatusHandler

**Modifications**: Set once by PeriodHandler, read by subsequent handlers

### ResourceConfig (resources.Config)

**Purpose**: Resource configuration for scaling operations.

**Set By**: PeriodHandler

**Used By**: ScalingHandler

**Modifications**: Set once by PeriodHandler, read by ScalingHandler

### SuccessResults ([]common.ScalerStatusSuccess)

**Purpose**: Successful scaling operations.

**Set By**: ScalingHandler

**Used By**: StatusHandler

**Modifications**: Set by ScalingHandler, read by StatusHandler

### FailedResults ([]common.ScalerStatusFailed)

**Purpose**: Failed scaling operations.

**Set By**: ScalingHandler

**Used By**: StatusHandler

**Modifications**: Set by ScalingHandler, read by StatusHandler

### ShouldFinalize (bool)

**Purpose**: Flag indicating finalizer cleanup is needed.

**Set By**: FinalizerHandler

**Used By**: StatusHandler

**Modifications**: Set by FinalizerHandler, read by StatusHandler

### SkipRemaining (bool)

**Purpose**: Flag indicating chain should stop early (no remaining handlers should execute).

**Set By**: Any handler (e.g., FinalizerHandler, PeriodHandler)

**Used By**: Handlers check this flag before calling `next.execute()`

**Modifications**: Set by any handler, checked by handlers before continuing

### RequeueAfter (time.Duration)

**Purpose**: Requeue delay duration (first handler to set wins).

**Set By**: Any handler (e.g., PeriodHandler for run-once periods)

**Used By**: Controller (uses this value in `ctrl.Result`)

**Modifications**:
- First handler to set non-zero value wins
- Subsequent handlers should check if already set before setting their own value

## Context Lifecycle

### Initialization

Context is initialized by the controller before chain execution:

```go
reconCtx := &ReconciliationContext{
    Request: req,
    Client:  r.Client,
    Logger:  r.Logger,
}
```

### Handler Execution Flow

1. **FetchHandler**: Sets `Scaler`
2. **FinalizerHandler**: May set `ShouldFinalize` or `SkipRemaining`
3. **AuthHandler**: Sets `K8sClient`, `DynamicClient`, `Secret`
4. **PeriodHandler**: Sets `Period`, `ResourceConfig`, may set `SkipRemaining` or `RequeueAfter`
5. **ScalingHandler**: Sets `SuccessResults`, `FailedResults`
6. **StatusHandler**: Uses all context fields to update status

## Context Modification Rules

- **Last Write Wins**: When multiple handlers modify the same context field, later handlers overwrite earlier changes
- **Read-Only Fields**: `Request`, `Client`, `Logger` are set by controller and should not be modified by handlers
- **Single Assignment**: Some fields are set once and never modified (e.g., `K8sClient`, `Period`)
- **Accumulation**: Some fields accumulate values (e.g., `SuccessResults`, `FailedResults`)

## Validation Rules

- Context must be initialized with at least `Request`, `Client`, and `Logger` before chain execution
- `Scaler` must be set by FetchHandler before other handlers can use it
- `Period` must be set by PeriodHandler before ScalingHandler can use it
- Handlers should check for required fields before using them (defensive programming)
