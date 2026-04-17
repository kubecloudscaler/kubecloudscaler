# Flow Controller

Orchestrates multi-resource scaling workflows across K8s and GCP resources.

## Architecture

Same Chain of Responsibility pattern as the k8s/gcp controllers. One handler per
reconciliation step; services are injected into handlers for the business logic.

```
FetchHandler -> FinalizerHandler -> ProcessingHandler -> StatusHandler
```

- **FetchHandler** - Loads the Flow from the API server
- **FinalizerHandler** - Adds/removes `kubecloudscaler.cloud/flow-finalizer` via optimistic-locked Patch
- **ProcessingHandler** - Delegates to the `FlowProcessor` service. Populates
  `ctx.Condition` (success or failure) and `ctx.ProcessingError`, never short-circuits so
  StatusHandler can still persist the outcome. The controller classifies
  ProcessingError as CriticalError (ValidationError) or RecoverableError afterwards.
- **StatusHandler** - Writes the condition from `ctx.Condition` (falls back to a default
  success) via the `StatusUpdater` service.

Injected services:

- **FlowProcessor** - Core workflow processing (validate → map → create)
- **FlowValidator** - Validates flow configuration; returns `*ValidationError` for
  user-config mistakes (unknown period, invalid delay, inverted window, …)
- **ResourceCreator** - Creates child K8s/Gcp resources; builds deterministic
  collision-safe names via `childResourceName`
- **ResourceMapper** - Maps flow definitions to resource specs; `*ValidationError` for
  ambiguous / unknown / duplicate resources
- **StatusUpdater** - Writes conditions to Flow status with retry-on-conflict
- **TimeCalculator** - Computes timing delays; recurring periods are allowed to cross midnight

## Key Patterns

- Orchestrates creation of child `K8s` and `Gcp` CRD resources
- Cascading scaling with configurable delays between resources
- Status aggregation from all child resources
- Flow validation ensures referenced resources and configurations are valid

## Tests

- Service layer tests: `service/*_test.go`
