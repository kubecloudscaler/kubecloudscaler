# Makefile Command Contracts

## Command: `make test-coverage`

### Purpose
Run test coverage analysis locally and display results.

### Behavior
1. Run all tests with coverage: `go test -coverprofile=coverage.out ./...`
2. Generate functional coverage report: `go tool cover -func=coverage.out`
3. Display overall coverage percentage
4. Display per-package breakdown
5. Display `internal/service/` layer coverage
6. Optionally generate HTML report: `go tool cover -html=coverage.out -o coverage.html`

### Output Format
```
Test Coverage Report
====================
Overall Coverage: 82.5%
Service Layer Coverage: 95.2%

Package Breakdown:
  github.com/kubecloudscaler/kubecloudscaler/internal/service: 95.2%
  github.com/kubecloudscaler/kubecloudscaler/pkg/k8s: 78.1%
  github.com/kubecloudscaler/kubecloudscaler/internal/controller: 85.3%
  ...

HTML report generated: coverage.html
```

### Performance
- Must complete within 30 seconds (SC-007)
- Uses existing test infrastructure
- No additional dependencies

## Command: `make test-bench`

### Purpose
Run performance benchmarks for critical functions.

### Behavior
1. Run all benchmarks: `go test -bench=. -benchmem ./...`
2. Display benchmark results
3. Report in standard Go benchmark format

### Output Format
```
goos: linux
goarch: amd64
pkg: github.com/kubecloudscaler/kubecloudscaler/pkg/k8s
BenchmarkK8sClient_UpdateDeployment-8    1000    1234567 ns/op    12345 B/op    123 allocs/op
BenchmarkGCPClient_ScaleVMInstance-8      500     2345678 ns/op    23456 B/op    234 allocs/op
...
```

### Usage
- Run locally for performance testing
- Run in CI for advisory reporting (doesn't block)
- Track trends over time

## Command: `make test` (Updated)

### Purpose
Run all tests (unchanged functionality, may update paths after migration).

### Behavior
- Run unit and integration tests
- Exclude e2e tests (use `make test-e2e` separately)
- Generate coverage profile (for compatibility)

### Changes After Migration
- Test paths updated to reflect new `test/unit/` and `test/integration/` structure
- Functionality remains the same
