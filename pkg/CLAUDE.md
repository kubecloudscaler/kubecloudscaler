# Shared Packages

Exposed externally via `pkg/`. Contains reusable logic shared across controllers.

## period/

Period activation logic:
- `period.New` - Create a period from spec
- `IsActive` - Check if period is currently active
- `Type` - Period type (recurring, fixed)
- `Hash` - Period hash for change detection
- `Once` - One-shot period support
- Supports recurring and fixed datetime periods

## resources/

Resource scaling factory:
- `resources.NewResource(ctx, name, config, logger)` - Factory creating typed resource handlers
- Each handler implements `SetState(ctx)` for scaling operations
- Types: deployments, statefulsets, cronjobs, github-ars, hpa, vm-instances, scaledobjects

## Guidelines

- Packages here are shared externally; maintain backward compatibility
- Keep interfaces small and purpose-specific
- Test with `suite_test.go` pattern
