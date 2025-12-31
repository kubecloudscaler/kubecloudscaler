# Data Model: Rewrite GCP Controller Using Chain of Responsibility Pattern

## Reconciliation Context

**Purpose**: Container for state shared between handlers during reconciliation

**Fields**:
- `Scaler` (*kubecloudscalerv1alpha3.Gcp) - The GCP scaler resource being reconciled
- `Request` (ctrl.Request) - The reconciliation request (namespaced name)
- `Client` (client.Client) - Kubernetes client for API operations
- `GCPClient` (gcpClient.Client) - GCP Compute Engine API client
- `Logger` (*zerolog.Logger) - Structured logger for observability
- `Secret` (*corev1.Secret) - Authentication secret for GCP access (nullable)
- `Period` (*common.ScalerPeriod) - Current time period configuration
- `ResourceConfig` (resources.Config) - Resource management configuration
- `SuccessResults` ([]common.ScalerStatusSuccess) - Successfully scaled resources
- `FailedResults` ([]common.ScalerStatusFailed) - Failed scaling operations
- `ShouldFinalize` (bool) - Flag indicating if finalizer cleanup is needed
- `SkipRemaining` (bool) - Flag indicating if remaining handlers should be skipped

**Validation Rules**:
- Scaler must be non-nil after fetch handler
- GCPClient must be non-nil after auth handler (if authentication succeeds)
- Period must be valid after period validation handler

**State Transitions**:
- Initial: Empty context with Request only
- After Fetch: Scaler populated
- After Finalizer: ShouldFinalize flag set if needed
- After Auth: GCPClient and Secret populated
- After Period: Period and ResourceConfig populated
- After Scaling: SuccessResults and FailedResults populated
- After Status: Context ready for next reconciliation cycle

## Reconciliation Result

**Purpose**: Encapsulates the outcome of handler execution

**Fields**:
- `Continue` (bool) - Whether chain should continue to next handler
- `Requeue` (bool) - Whether reconciliation should be requeued
- `RequeueAfter` (time.Duration) - Delay before requeue (if Requeue is true)
- `Error` (error) - Error encountered (nil if no error)
- `ErrorCategory` (ErrorCategory) - Category of error (Critical or Recoverable)

**Validation Rules**:
- If Error is non-nil and ErrorCategory is Critical, Continue must be false
- If Requeue is true, RequeueAfter must be > 0
- If SkipRemaining is true in context, Continue must be false

## Handler Interface

**Purpose**: Contract that all chain handlers must implement

**Methods**:
- `Handle(ctx context.Context, req *ReconciliationContext) (*ReconciliationResult, error)`

**Behavior**:
- Handler processes its reconciliation step
- Handler modifies context as needed for subsequent handlers
- Handler returns result indicating: continue, stop, requeue, or skip
- Handler returns error if critical failure occurs

**Error Handling**:
- Critical errors: Return error with CriticalError type, chain stops
- Recoverable errors: Return error with RecoverableError type, chain may continue with requeue
- No error: Return nil error, result indicates next action

## Handler Chain

**Purpose**: Manages ordered sequence of handlers and executes them

**Fields**:
- `handlers` ([]Handler) - Ordered list of handlers
- `logger` (*zerolog.Logger) - Logger for chain execution

**Methods**:
- `Execute(ctx context.Context, req *ReconciliationContext) (ctrl.Result, error)` - Execute all handlers in order

**Behavior**:
- Executes handlers in fixed order
- Stops on critical error or when handler requests stop
- Continues on recoverable error with requeue
- Respects skip flag from handlers
- Returns first requeue delay encountered
- Returns final reconciliation result

## Error Categories

**Purpose**: Categorize errors for appropriate handling

**Types**:
- `CriticalError` - Errors that indicate reconciliation cannot proceed (authentication failures, invalid configuration, resource not found)
- `RecoverableError` - Errors that may be resolved with retry (temporary rate limits, transient network issues, temporary API unavailability)

**Validation Rules**:
- Critical errors must stop chain execution
- Recoverable errors allow chain continuation with requeue
- Error category must be determinable from error type or wrapping

## Handler Implementations

### Fetch Handler
- **Responsibility**: Fetch scaler resource from Kubernetes API
- **Input**: Request (namespaced name)
- **Output**: Scaler resource in context
- **Errors**: Resource not found (Critical), API errors (Recoverable)

### Finalizer Handler
- **Responsibility**: Manage finalizer lifecycle (add/remove)
- **Input**: Scaler resource
- **Output**: ShouldFinalize flag set if deletion in progress
- **Errors**: Update failures (Recoverable)

### Authentication Handler
- **Responsibility**: Setup GCP client with authentication
- **Input**: Scaler spec (auth secret reference), Kubernetes client
- **Output**: GCPClient and Secret in context
- **Errors**: Secret not found (Critical), Client creation failure (Critical)

### Period Validation Handler
- **Responsibility**: Validate and determine current time period
- **Input**: Scaler spec periods, current status
- **Output**: Period and ResourceConfig in context
- **Errors**: Invalid period configuration (Critical), Run-once period (requeue, not error)

### Resource Scaling Handler
- **Responsibility**: Scale GCP resources based on period
- **Input**: Period, ResourceConfig, GCPClient
- **Output**: SuccessResults and FailedResults in context
- **Errors**: Scaling failures (Recoverable - individual resource failures don't stop chain)

### Status Update Handler
- **Responsibility**: Update scaler status with operation results
- **Input**: All context fields (results, period, etc.)
- **Output**: Status updated in Kubernetes
- **Errors**: Update failures (Recoverable)
