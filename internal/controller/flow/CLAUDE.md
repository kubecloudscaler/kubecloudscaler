# Flow Controller

Orchestrates multi-resource scaling workflows across K8s and GCP resources.

## Architecture

Same Chain of Responsibility pattern as the k8s/gcp controllers. One handler per
reconciliation step; services are injected into handlers for the business logic.

```
FetchHandler -> FinalizerHandler -> ProcessingHandler -> StatusHandler
```

- **FetchHandler** - Loads the Flow from the API server
- **FinalizerHandler** - Adds/removes `kubecloudscaler.cloud/flow-finalizer` via
  optimistic-locked Patch. NotFound is treated as a no-op (the Flow was deleted between
  Fetch and Patch); on success `ctx.Flow` is refreshed so later handlers see the current
  ResourceVersion.
- **ProcessingHandler** - Delegates to the `FlowProcessor` service. Always populates
  `ctx.Condition` (success or failure); populates `ctx.ProcessingError` only on failure.
  Never short-circuits so StatusHandler can still persist the outcome. The controller then
  classifies `ctx.ProcessingError` as CriticalError (ValidationError) or RecoverableError
  independently of the chain's returned error — StatusHandler failing to persist must not
  downgrade a ValidationError to a hot-requeuing RecoverableError.
- **StatusHandler** - Writes the condition from `ctx.Condition` (falls back to a default
  success when the chain ended before ProcessingHandler). Arms the success requeue
  (`ReconcileSuccessDuration`) only when processing actually succeeded; transient
  processing failures fall through to the controller's `ReconcileErrorDuration` default.

Injected services:

- **FlowProcessor** - Core workflow processing (validate → map → create)
- **FlowValidator** - Validates flow configuration; returns `*ValidationError` for
  user-config mistakes. See `service/errors.go` for the full closed set of reason
  constants.
- **ResourceCreator** - Creates child K8s/Gcp resources; builds deterministic
  collision-safe names via `childResourceName`
- **ResourceMapper** - Maps flow definitions to resource specs; `*ValidationError` for
  ambiguous / unknown / duplicate resources
- **StatusUpdater** - Writes conditions to Flow status with retry-on-conflict
- **TimeCalculator** - Computes timing delays. Recurring periods are allowed to cross
  midnight for duration math; zero-duration periods (start == end) are rejected. Note:
  `pkg/period` currently refuses cross-midnight recurring windows at activation time, so
  a 22:00→02:00 Flow will pass validation but fail activation until pkg/period is aligned.

## Key Patterns

- Orchestrates creation of child `K8s` and `Gcp` CRD resources
- Cascading scaling with configurable delays between resources
- Status aggregation from all child resources
- Flow validation ensures referenced resources and configurations are valid

## Tests

- Service layer tests: `service/*_test.go`
