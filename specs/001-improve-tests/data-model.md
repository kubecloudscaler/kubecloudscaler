# Data Model: Test Organization Structure

## Overview

This feature improves test infrastructure rather than adding application data models. The "data model" here represents the test file organization structure and coverage data structures.

## Test Directory Structure

### Entity: Test Organization

**Structure**:
```
test/
├── unit/                    # All unit tests
│   ├── internal/
│   │   ├── controller/
│   │   │   ├── k8s/
│   │   │   ├── gcp/
│   │   │   └── flow/
│   │   ├── service/         # Business logic tests (100% coverage required)
│   │   └── webhook/
│   └── pkg/
│       ├── k8s/
│       ├── gcp/
│       └── period/
├── integration/             # All integration tests
│   ├── k8s/                 # Kubernetes API integration tests
│   ├── gcp/                 # GCP API integration tests
│   └── flow/                # Flow controller integration tests
└── e2e/                     # End-to-end tests (unchanged)
    ├── e2e_suite_test.go
    └── e2e_test.go
```

**Rules**:
- Unit tests: Fast, isolated, use mocks, no external dependencies
- Integration tests: Require external services, use test environment
- Package structure mirrors source code structure for clarity
- All tests use Ginkgo/Gomega (constitution requirement)

## Coverage Data Structure

### Entity: Coverage Report

**Fields**:
- `OverallCoverage`: float64 - Overall codebase coverage percentage (must be >= 80%)
- `PackageCoverage`: map[string]float64 - Per-package coverage percentages
- `ServiceLayerCoverage`: float64 - Coverage for `internal/service/` layer (must be 100%)
- `Timestamp`: time.Time - When coverage was calculated

**Validation Rules**:
- OverallCoverage >= 80.0 (enforced in CI)
- ServiceLayerCoverage == 100.0 (all exported functions covered)
- PackageCoverage entries help identify gaps

## Benchmark Data Structure

### Entity: Benchmark Result

**Fields**:
- `Name`: string - Benchmark function name
- `Operations`: int64 - Number of operations performed
- `Duration`: time.Duration - Total time taken
- `NsPerOp`: int64 - Nanoseconds per operation
- `AllocBytes`: int64 - Bytes allocated per operation
- `AllocOps`: int64 - Allocations per operation

**Usage**:
- Reported in CI output (advisory only)
- Tracked over time for regression detection
- Not used to block merges

## Test File Naming Conventions

### Rules:
- Unit tests: `*_test.go` in `test/unit/` matching source package structure
- Integration tests: `*_test.go` in `test/integration/` organized by feature
- Benchmarks: `*_bench_test.go` (can be alongside source or in test/unit/)
- Test suites: `*_suite_test.go` for Ginkgo suite setup

## Migration Mapping

### Source → Destination

**Current state** (tests alongside source):
- `internal/controller/k8s/scaler_controller_test.go` → `test/unit/internal/controller/k8s/scaler_controller_test.go`
- `pkg/k8s/resources/deployments/deployments_test.go` → `test/unit/pkg/k8s/resources/deployments/deployments_test.go`
- `test/e2e/e2e_test.go` → `test/e2e/e2e_test.go` (unchanged)

**Classification logic**:
- Tests using mocks only → `test/unit/`
- Tests requiring K8s API server or GCP APIs → `test/integration/`
- Tests using envtest or real clusters → `test/integration/` or `test/e2e/`
