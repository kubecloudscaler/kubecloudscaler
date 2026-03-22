# K8s Controller

Scales Kubernetes workloads based on time periods.

## Supported Resources

- Deployments (`common.ResourceDeployments`)
- StatefulSets (`common.ResourceStatefulSets`)
- CronJobs (`common.ResourceCronJobs`)
- HPAs (`common.ResourceHPA`)
- GitHub AutoScalingRunnerSets

## Handler Chain

```
FetchHandler -> FinalizerHandler -> AuthHandler -> PeriodHandler -> ScalingHandler -> StatusHandler
```

- **AuthHandler**: Sets up K8s authentication via secrets; populates `K8sClient`, `DynamicClient`
- **PeriodHandler**: Uses `utils.SetActivePeriod()`; handles `ErrRunOncePeriod` sentinel
- **ScalingHandler**: Delegates to `pkg/resources/` factory for resource-specific scaling

## Key Patterns

- Resource factory: `resources.NewResource(ctx, name, config, logger)` creates typed handlers
- Each resource type implements `SetState(ctx)` for scaling actions
- Period evaluation via `pkg/period/` package (`period.New`, `IsActive`, `Type`, `Hash`, `Once`)

## Tests

- Handler tests: `service/handlers/*_test.go`
- Controller integration: `*_test.go`
- Service layer: `service/*_test.go`
