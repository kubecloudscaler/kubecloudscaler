# GCP Controller

Scales GCP resources based on time periods.

## Supported Resources

- Compute Engine VM instances (`common.ResourceVMInstances` = `"vm-instances"`)
- Managed Instance Groups (`common.ResourceInstanceGroupManagers` = `"instance-group-managers"`)
  - Uses `instanceGroupManagers.stopInstances`/`startInstances` (MIG-aware API)
  - Selection by MIG name (`resources.names`); label selectors do not apply to MIGs
  - Do NOT use plain `instances.stop` on MIG-managed VMs — the autohealer will immediately recreate them
- Do NOT use `"instance"`, `"disk"`, or other arbitrary strings

## Handler Chain

```
FetchHandler -> FinalizerHandler -> AuthHandler -> PeriodHandler -> ScalingHandler -> StatusHandler
```

- **AuthHandler**: Sets up GCP authentication via service account secrets
- **PeriodHandler**: Uses `utils.SetActivePeriod()`; handles `ErrRunOncePeriod` sentinel
- **ScalingHandler**: Delegates to `pkg/resources/` factory for VM instance operations

## Key Patterns

- GCP Compute client: `cloud.google.com/go/compute`
- Resource factory: `resources.NewResource(ctx, name, config, logger)`
- Period evaluation via `pkg/period/`

## Tests

- Handler tests: `service/handlers/*_test.go`
- Controller integration: `*_test.go`
- Service layer: `service/*_test.go`
