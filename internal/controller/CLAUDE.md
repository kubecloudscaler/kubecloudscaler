# Controllers - Reconciliation Logic

## Chain of Responsibility Pattern (MANDATORY)

`Reconcile()` MUST NOT contain business logic — it MUST delegate to a handler chain.

```go
type Handler interface {
    Execute(ctx *ReconciliationContext) error
    SetNext(next Handler)
}
```

**Standard chain**: `FetchHandler -> FinalizerHandler -> AuthHandler -> PeriodHandler -> ScalingHandler -> StatusHandler`

### Handler Rules

- One handler = one reconciliation step. Share state via `ReconciliationContext`.
- Call `next.Execute(ctx)` to continue (check `next != nil`)
- `CriticalError` -> stop, no requeue | `RecoverableError` -> stop, requeue | `nil` -> continue
- Chain order MUST NOT change without review. New steps = new handlers.

### ReconciliationContext Contract

- **Controller**: `Ctx`, `Request`, `Client`, `Logger`
- **FetchHandler**: `Scaler` | **AuthHandler**: `K8sClient`, `DynamicClient`, `Secret`
- **PeriodHandler**: `Period`, `ResourceConfig`; may set `SkipRemaining = true`
- **ScalingHandler**: `SuccessResults`, `FailedResults`
- Any handler: `RequeueAfter` (first write wins), `SkipRemaining`

### Error Handling

- `service.NewCriticalError(err)` / `service.NewRecoverableError(err)` / `nil`
- Check: `service.IsCriticalError(err)` / `service.IsRecoverableError(err)`
- HTTP 409 on status: `retry.RetryOnConflict(retry.DefaultRetry, ...)`
- Default requeue: `utils.ReconcileErrorDuration` (10m)

## Reconciliation Best Practices

- **Idempotent** and **level-triggered** (react to state, not events)
- `ctrl.Result{RequeueAfter: duration}`, not `Requeue: true`
- `client.IgnoreNotFound(err)` for deletable resources
- Status subresource only; NEVER update status and spec together
- Prefer SSA (`client.Apply`) over Update; unique field manager per controller

## Finalizers

- Add on first reconciliation, remove after cleanup. Check: `!obj.DeletionTimestamp.IsZero()`
- Cleanup MUST be idempotent

## Predicates

- `GenerationChangedPredicate` (spec-only), `IgnoreDeletionPredicate`
- `Owns()` for child resources, `EnqueueRequestForOwner` for custom refs

## Adding a New Handler

1. Create in `service/handlers/`, implement `Handler` interface
2. TDD: tests first (success, critical error, recoverable error)
3. Register in `buildChain()`. Update RBAC markers -> `make manifests`
