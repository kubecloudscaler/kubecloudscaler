## ADDED Requirements

### Requirement: Code Compliance Review
The codebase SHALL be reviewed against the project specification to ensure compliance with established conventions, patterns, and best practices.

#### Scenario: Comprehensive code review
- **WHEN** reviewing code in `internal/` and `pkg/` directories
- **THEN** all code SHALL follow the patterns defined in `openspec/project.md`
- **AND** findings SHALL be documented for remediation

#### Scenario: Error handling compliance
- **WHEN** errors occur in library code
- **THEN** errors SHALL be returned (not panicked)
- **AND** errors SHALL be wrapped using `fmt.Errorf("context: %w", err)`
- **AND** no `panic()` or `os.Exit()` calls SHALL be present

#### Scenario: Dependency injection compliance
- **WHEN** creating components
- **THEN** dependencies SHALL be injected via constructors
- **AND** no global state variables SHALL be used
- **AND** all dependencies SHALL use interfaces where possible

#### Scenario: Context propagation compliance
- **WHEN** functions perform operations that can be cancelled or have timeouts
- **THEN** `context.Context` SHALL be the first parameter
- **AND** context SHALL be propagated to all child operations

#### Scenario: Interface-driven development compliance
- **WHEN** defining dependencies
- **THEN** interfaces SHALL be defined for external dependencies
- **AND** public functions SHALL interact with interfaces, not concrete types
- **AND** interfaces SHALL be small and purpose-specific

#### Scenario: Testing strategy compliance
- **WHEN** writing tests
- **THEN** tests SHALL use Ginkgo (BDD framework) and Gomega (matcher library)
- **AND** unit tests SHALL be separated from integration tests
- **AND** external services SHALL be mocked using interfaces
- **AND** all exported functions SHALL have test coverage

#### Scenario: Architecture pattern compliance
- **WHEN** organizing code
- **THEN** code SHALL follow Clean Architecture principles
- **AND** controllers SHALL delegate to service layers
- **AND** data access SHALL be separated via Repository Pattern
- **AND** business logic SHALL be decoupled from framework code

#### Scenario: Logging compliance
- **WHEN** logging events
- **THEN** structured JSON logs SHALL be used via zerolog
- **AND** log levels SHALL be appropriate (info, warn, error)
- **AND** secrets SHALL NOT be exposed in logs

#### Scenario: golangci-lint compliance
- **WHEN** code is written or modified
- **THEN** code SHALL pass all enabled golangci-lint checks without errors
- **AND** golangci-lint configuration SHALL be defined in `.golangci.yml`
- **AND** golangci-lint version SHALL match the configuration file version (v1 or v2)
- **AND** CI/CD pipelines SHALL run golangci-lint validation
- **AND** all linter warnings SHALL be addressed or explicitly suppressed with proper `//nolint` directives
- **AND** `//nolint` directives SHALL include explanations when required by configuration
- **AND** golangci-lint SHALL be run before code is merged

#### Scenario: Code formatting compliance
- **WHEN** code is formatted
- **THEN** code SHALL be formatted using `gofmt` or `goimports`
- **AND** formatting SHALL be enforced by golangci-lint formatters
- **AND** CI/CD pipelines SHALL verify code formatting

## ADDED Requirements

### Requirement: Project Specification Clarity
The project specification SHALL clearly distinguish between Go-specific conventions and general development practices.

#### Scenario: Go naming conventions
- **WHEN** defining naming conventions in project specification
- **THEN** Go-specific conventions SHALL be clearly documented:
  - PascalCase for exported types, functions, constants
  - camelCase for unexported/internal identifiers
  - lowercase for package names
  - No JavaScript/React-specific conventions SHALL be included

#### Scenario: Code style guidelines
- **WHEN** documenting code style
- **THEN** Go-specific guidelines SHALL be emphasized:
  - Use tabs for indentation (Go standard)
  - Use `gofmt` or `goimports` for formatting
  - Follow idiomatic Go conventions from Effective Go
  - Remove references to JavaScript/TypeScript conventions
  - Enforce code quality with `golangci-lint` as specified in project conventions
  - All code SHALL pass golangci-lint validation before merging
