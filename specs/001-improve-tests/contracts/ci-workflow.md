# CI Workflow Contract: Test Coverage Enforcement

## Workflow: `.github/workflows/test-coverage.yml`

### Purpose
Enforce 80% test coverage requirement on all pull requests. Run coverage analysis on PR open and every update.

### Triggers
- `pull_request` events:
  - `opened` - When PR is created
  - `synchronize` - When PR is updated with new commits
  - `reopened` - When closed PR is reopened

### Steps

1. **Checkout Code**
   - Checkout PR branch and base branch for comparison

2. **Setup Go**
   - Use Go version from `go.mod` (currently 1.25.1)
   - Cache Go modules for faster builds

3. **Run Tests with Coverage**
   ```bash
   go test -coverprofile=coverage.out ./...
   ```

4. **Generate Coverage Report**
   ```bash
   go tool cover -func=coverage.out > coverage.txt
   ```

5. **Parse Coverage Percentage**
   - Extract overall coverage from last line: `total: (statements) XX.X%`
   - Extract per-package breakdown from coverage.txt
   - Extract `internal/service/` layer coverage

6. **Check Coverage Threshold**
   - If overall coverage < 80%: FAIL workflow
   - If `internal/service/` coverage < 100%: WARN (not blocking)
   - If overall coverage >= 80%: PASS workflow

7. **Report Results**
   - Post coverage summary as workflow summary
   - Optionally comment on PR with coverage breakdown
   - Include per-package breakdown for gap identification

### Output Format

**Workflow Summary**:
```
Test Coverage Report
====================
Overall Coverage: 82.5% ✅ (Required: 80%)
Service Layer Coverage: 95.2% ⚠️ (Required: 100%)

Package Breakdown:
  internal/service: 95.2%
  pkg/k8s: 78.1%
  internal/controller: 85.3%
  ...
```

**Failure Message**:
```
❌ Coverage check failed
Overall coverage: 78.5% (Required: 80%)
Please add tests to increase coverage above 80%
```

### Status Check
- Workflow must be set as required status check
- Blocks merge if coverage < 80%
- Does not block for service layer coverage (warning only)
