# Quickstart: Improved Test Infrastructure

## Overview

This guide explains how to use the improved test infrastructure after implementation.

## Running Tests

### Unit Tests
```bash
# Run all unit tests
go test ./test/unit/...

# Run tests for specific package
go test ./test/unit/internal/service/...

# Run with verbose output
go test -v ./test/unit/...
```

### Integration Tests
```bash
# Run all integration tests
go test ./test/integration/...

# Run specific integration test
go test -v ./test/integration/k8s/...
```

### End-to-End Tests
```bash
# Run e2e tests (unchanged)
make test-e2e
```

## Checking Coverage

### Local Coverage Analysis
```bash
# Run coverage analysis
make test-coverage

# View HTML coverage report
open coverage.html  # or xdg-open on Linux
```

**Expected output**:
- Overall coverage percentage (must be >= 80%)
- Per-package breakdown
- Service layer coverage (should be 100%)
- HTML report for visual gap identification

### Coverage in CI
- Coverage is automatically checked on every PR
- PR will be blocked if coverage < 80%
- Coverage report visible in GitHub Actions workflow summary
- Per-package breakdown helps identify gaps

## Writing Tests

### Unit Test Structure
```go
// test/unit/internal/service/my_service_test.go
package service_test

import (
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

var _ = Describe("MyService", func() {
    Describe("ProcessData", func() {
        It("should process valid data", func() {
            // test implementation
        })
    })
})
```

### Integration Test Structure
```go
// test/integration/k8s/scaler_test.go
package k8s_test

import (
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

var _ = Describe("K8s Scaler Integration", func() {
    It("should scale deployment", func() {
        // integration test with real K8s API
    })
})
```

### Table-Driven Tests
```go
var _ = Describe("ValidateInput", func() {
    DescribeTable("should validate input correctly",
        func(input string, expected bool) {
            result := ValidateInput(input)
            Expect(result).To(Equal(expected))
        },
        Entry("valid input", "valid", true),
        Entry("invalid input", "", false),
        Entry("edge case", "   ", false),
    )
})
```

## Adding Benchmarks

### Benchmark Structure
```go
// test/unit/pkg/k8s/bench_test.go or alongside source
func BenchmarkK8sClient_UpdateDeployment(b *testing.B) {
    // setup
    for i := 0; i < b.N; i++ {
        // operation to benchmark
    }
}
```

### Running Benchmarks
```bash
# Run all benchmarks
make test-bench

# Run specific benchmark
go test -bench=BenchmarkK8sClient -benchmem ./test/unit/pkg/k8s/
```

## Test Organization

### Where to Place Tests

**Unit Tests** (`test/unit/`):
- Fast, isolated tests
- Use mocks for external dependencies
- No external services required
- Mirror source package structure

**Integration Tests** (`test/integration/`):
- Require external services (K8s API, GCP API)
- Use test environment or envtest
- Organized by feature/package

**E2E Tests** (`test/e2e/`):
- Full system tests
- Require complete cluster setup
- Use `make test-e2e`

## Coverage Requirements

### Overall Coverage
- **Minimum**: 80% (enforced in CI)
- Check locally: `make test-coverage`
- CI blocks merge if below threshold

### Service Layer Coverage
- **Required**: 100% for all exported functions in `internal/service/`
- Critical business logic must be fully covered
- Check specific layer: `go tool cover -func=coverage.out | grep internal/service`

## Troubleshooting

### Coverage Below 80%
1. Run `make test-coverage` to see gaps
2. Open `coverage.html` to identify uncovered lines
3. Focus on `internal/service/` layer first
4. Add tests for uncovered functions

### Tests Fail After Migration
1. Check import paths in moved test files
2. Verify package declarations match new locations
3. Ensure test utilities are accessible
4. Run `go mod tidy` to update dependencies

### Benchmark Not Running
1. Ensure benchmark function name starts with `Benchmark`
2. Use `-bench=.` flag to run all benchmarks
3. Check benchmark file is included in package

## Next Steps

After implementation:
1. All tests organized in `test/unit/` and `test/integration/`
2. Coverage enforced at 80% minimum
3. Benchmarks added for critical functions
4. CI automatically checks coverage on PRs
5. Local `make test-coverage` command available
