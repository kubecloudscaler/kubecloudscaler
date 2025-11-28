# Change: Fix All Linter Issues

## Why

The codebase has multiple golangci-lint issues that need to be fixed to ensure code quality and compliance with project specifications. These issues include import shadowing, code duplication, unchecked errors, cognitive complexity, and other code quality concerns.

## What Changes

This change will:
- Fix all golangci-lint issues identified in `internal/` and `pkg/` directories
- Address import shadowing by renaming shadowed variables
- Refactor duplicate code to reduce duplication
- Fix unchecked error returns
- Reduce cognitive complexity where appropriate
- Extract magic strings to constants
- Name return values for better readability
- Optimize parameter passing for large structs
- Ensure all code passes golangci-lint validation

**Note**: This change focuses on fixing linter issues without changing functionality.

## Impact

- **Affected specs**: Code quality requirements
- **Affected code**: Multiple files in `internal/` and `pkg/` directories
- **Linter issues to fix**:
  - importShadow (multiple files)
  - dupl (code duplication)
  - errcheck (unchecked errors)
  - gocognit (high cognitive complexity)
  - goconst (repeated strings)
  - unnamedResult (unnamed return values)
  - paramTypeCombine (parameter type combination)
  - hugeParam (large parameters)
