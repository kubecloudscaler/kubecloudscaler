# Design: Fix All Linter Issues

## Context

The codebase has been reviewed and golangci-lint has identified multiple code quality issues that need to be addressed. This change focuses on fixing these issues while maintaining functionality and code behavior.

## Goals

- Fix all golangci-lint issues without changing functionality
- Improve code quality and maintainability
- Ensure compliance with project specifications
- Maintain backward compatibility

## Issues and Solutions

### 1. Import Shadowing

**Issue**: Parameter names shadow imported package names (e.g., `client client.Client`)

**Solution**: Rename parameters to avoid shadowing:
- `client client.Client` → `k8sClient client.Client` or `c client.Client`
- Apply consistently across all affected files

**Files Affected**:
- `internal/controller/flow/flow_controller.go`
- `internal/utils/reconciler.go`
- `internal/utils/secret.go`

### 2. Code Duplication

**Issue**: `CreateK8sResource` and `CreateGcpResource` have duplicate code (dupl linter)

**Solution**: Extract common logic into a shared helper function:
- Create `createResource` helper that handles common operations
- Both methods call the helper with type-specific parameters
- Reduces duplication while maintaining type safety

**Files Affected**:
- `internal/controller/flow/service/resource_creator.go`

### 3. Unchecked Errors

**Issue**: Error return value not checked in `DeepCopyObject()`

**Solution**: Add proper error checking:
- Check error return value
- Handle error appropriately (log and return)

**Files Affected**:
- `internal/controller/flow/service/resource_creator.go`

### 4. Cognitive Complexity

**Issue**: `NewResource` function has cognitive complexity 21 (> 20)

**Solution**: Refactor to reduce complexity:
- Extract helper functions for different resource types
- Use early returns where appropriate
- Break down complex conditionals

**Files Affected**:
- `pkg/resources/resources.go`

### 5. Magic Strings

**Issue**: String `"down"` appears 3 times and should be a constant

**Solution**: Extract to constant:
- Define `const periodTypeDown = "down"` or similar
- Use constant throughout the file

**Files Affected**:
- `pkg/k8s/resources/base/strategies.go`

### 6. Unnamed Return Values

**Issue**: Functions with multiple return values should have named returns for clarity

**Solution**: Add named return values:
- `func ExtractFlowData(...) (resourceNames map[string]bool, periodNames map[string]bool, err error)`
- Improves readability and documentation

**Files Affected**:
- `internal/controller/flow/service/flow_validator.go`
- `internal/controller/flow/service/mocks.go`

### 7. Parameter Optimization

**Issue**: Large parameters should be passed by pointer, or parameter types should be combined

**Solution**:
- Combine parameter types: `func(currentPeriodName, statusPeriodName string)`
- For large structs, consider passing by pointer (but note: Kubernetes API types are typically passed by value for immutability)

**Files Affected**:
- `internal/utils/reconciler.go`
- `pkg/k8s/resources/cronjobs/adapter.go`
- `pkg/k8s/resources/deployments/adapter.go`

## Decisions

### Decision: Maintain Kubernetes API Conventions
- **Rationale**: Kubernetes API types (`metaV1.ListOptions`, `metaV1.UpdateOptions`) are typically passed by value for immutability
- **Approach**: For `hugeParam` warnings on Kubernetes API types, we may need to suppress with `//nolint` if changing would violate API conventions
- **Alternative**: Pass by pointer - Rejected: Would violate Kubernetes API conventions

### Decision: Preserve Functionality
- **Rationale**: All fixes must maintain existing behavior
- **Approach**: Run tests after each fix to ensure no regressions
- **Validation**: Full test suite must pass

## Risks / Trade-offs

### Risk: Breaking Changes
- **Risk**: Refactoring might introduce bugs
- **Mitigation**: Run full test suite after each change, fix incrementally

### Risk: Performance Impact
- **Risk**: Some optimizations might affect performance
- **Mitigation**: Profile if needed, but most fixes are code quality improvements

## Implementation Strategy

1. Fix issues incrementally, one category at a time
2. Run tests after each fix
3. Run golangci-lint after each fix to verify resolution
4. Commit fixes in logical groups
