# Flow Controller

Orchestrates multi-resource scaling workflows across K8s and GCP resources.

## Architecture

The Flow controller uses a service-based pattern (not the handler chain):

- **FlowProcessor** - Core workflow processing
- **FlowValidator** - Validates flow configuration
- **ResourceCreator** - Creates child K8s/Gcp resources
- **ResourceMapper** - Maps flow definitions to resource specs
- **StatusUpdater** - Aggregates status from child resources
- **TimeCalculator** - Computes timing delays for cascade scaling

## Key Patterns

- Orchestrates creation of child `K8s` and `Gcp` CRD resources
- Cascading scaling with configurable delays between resources
- Status aggregation from all child resources
- Flow validation ensures referenced resources and configurations are valid

## Tests

- Service layer tests: `service/*_test.go`
