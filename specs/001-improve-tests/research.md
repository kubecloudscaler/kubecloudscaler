# Research: Improve Tests

## Coverage Tooling & Reporting

### Decision: Use Go's built-in coverage tools with custom reporting

**Rationale**:
- Go's `go test -cover` provides reliable coverage analysis
- `go tool cover` can generate HTML reports with per-package breakdown
- Standard tooling ensures compatibility and maintainability
- No external dependencies required

**Alternatives considered**:
- `gocov` / `gocovmerge`: Additional tooling, not standard
- Third-party coverage services: External dependency, potential cost
- Custom coverage parser: Unnecessary complexity

**Implementation approach**:
- Use `go test -coverprofile=coverage.out ./...` for overall coverage
- Use `go tool cover -func=coverage.out` for per-package breakdown
- Parse output to extract overall percentage and per-package percentages
- Generate coverage report in CI and local command

## CI Integration for Coverage Enforcement

### Decision: GitHub Actions workflow with coverage check step

**Rationale**:
- Project uses GitHub (evident from repository structure)
- GitHub Actions provides native CI/CD integration
- Can block merges via required status checks
- Supports PR comments for coverage reporting

**Alternatives considered**:
- GitLab CI: Not applicable (project uses GitHub)
- Jenkins: More complex setup, overkill for this need
- External CI services: Additional dependency

**Implementation approach**:
- Create `.github/workflows/test-coverage.yml`
- Run coverage analysis on PR open and every update
- Parse coverage percentage from `go tool cover -func` output
- Fail workflow if overall coverage < 80%
- Post coverage report as PR comment (optional, for visibility)
- Set as required status check to block merges

**Coverage reporting format**:
```
Overall coverage: 82.5%
Package breakdown:
  github.com/kubecloudscaler/kubecloudscaler/internal/service: 95.2%
  github.com/kubecloudscaler/kubecloudscaler/pkg/k8s: 78.1%
  ...
```

## Local Coverage Command

### Decision: Makefile target `make test-coverage`

**Rationale**:
- Consistent with existing Makefile-based workflow
- Easy to discover and use
- Can be integrated into developer workflow
- Fast execution (< 30 seconds per SC-007)

**Implementation approach**:
- Add `test-coverage` target to Makefile
- Run `go test -coverprofile=coverage.out ./...`
- Generate coverage report with `go tool cover -func=coverage.out`
- Display overall percentage and per-package breakdown
- Optionally generate HTML report: `go tool cover -html=coverage.out`

## Test Reorganization Strategy

### Decision: Big-bang migration in single PR

**Rationale**:
- Spec clarification confirmed big-bang approach
- Minimizes transition period and confusion
- Single review cycle for all test moves
- Clear before/after state

**Migration steps**:
1. Create `test/unit/` and `test/integration/` directories
2. Identify test types (unit vs integration):
   - Unit: Fast, isolated, no external dependencies, use mocks
   - Integration: Require external services (K8s API, GCP API), use test environment
3. Move all `*_test.go` files from source directories to appropriate test directories
4. Update import paths in moved test files
5. Update package declarations to match new locations
6. Verify all tests still pass after move
7. Update Makefile test targets if needed

**Package structure in test/unit/**:
- Mirror source structure: `test/unit/internal/service/`, `test/unit/pkg/k8s/`, etc.
- Maintains package organization and import clarity

## Benchmark Implementation

### Decision: Use Go standard `testing.B` benchmarks

**Rationale**:
- Built into Go standard library
- No additional dependencies
- Standard format: `func BenchmarkXxx(b *testing.B)`
- Results can be parsed and reported in CI

**Alternatives considered**:
- Custom benchmarking framework: Unnecessary complexity
- External benchmarking tools: Additional dependencies

**Implementation approach**:
- Add `*_bench_test.go` files alongside source or in test/unit/
- Benchmark critical functions:
  - Kubernetes API calls (client operations, resource updates)
  - GCP API calls (VM instance operations, Cloud SQL operations)
  - Heavy computation (period calculations, resource mapping)
- Run benchmarks with `go test -bench=. -benchmem`
- Report in CI output (advisory, doesn't block)
- Track performance trends over time

**Benchmark naming**:
- `BenchmarkK8sClient_UpdateDeployment`
- `BenchmarkGCPClient_ScaleVMInstance`
- `BenchmarkPeriodCalculator_CalculateNextPeriod`

## Table-Driven Test Patterns

### Decision: Standard Go table-driven test pattern

**Rationale**:
- Well-established Go testing pattern
- Reduces code duplication
- Easy to add new test cases
- Clear test structure

**Implementation approach**:
- Identify functions with 3+ input variants
- Create test table structure:
  ```go
  tests := []struct {
    name string
    input interface{}
    expected interface{}
    expectError bool
  }{
    // test cases
  }
  ```
- Use Ginkgo's `DescribeTable` for BDD-style table tests
- Ensure all branches are covered

## Coverage Gap Analysis

### Decision: Use `go tool cover -html` for visual gap identification

**Rationale**:
- Built-in Go tooling
- Visual representation helps identify uncovered lines
- Can be generated locally and in CI
- No additional dependencies

**Implementation approach**:
- Generate HTML coverage report: `go tool cover -html=coverage.out -o coverage.html`
- Focus on `internal/service/` layer for 100% coverage requirement
- Identify uncovered functions and branches
- Prioritize business logic coverage gaps

## CI Workflow Timing

### Decision: Run coverage check on PR open and every update

**Rationale**:
- Spec clarification confirmed this approach
- Provides early feedback to developers
- Catches regressions on each push
- Balances feedback speed with CI resource usage

**Implementation approach**:
- GitHub Actions workflow triggers on:
  - `pull_request` events (opened, synchronize, reopened)
- Run coverage analysis as part of test workflow
- Fail fast if coverage drops below threshold
- Report results in workflow summary
