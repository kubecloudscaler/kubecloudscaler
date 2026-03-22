# Internal Code

## Go Conventions (Go 1.25+)

- Idiomatic Go: Effective Go, Google's Go Style Guide, Go Proverbs
- Tabs for indentation, 140 char line limit (`lll`), `gofmt` + `goimports`
- Named functions over long anonymous ones; small composable single-responsibility functions
- Modern features: `range` over ints/functions, `slices`/`maps`/`cmp`, generics, `context.AfterFunc`
- Prefer `zerolog` over `log/slog`

### Naming

- CamelCase exports, camelCase private; acronyms uppercase (`HTTP`, `URL`, `ID`, `GCP`, `CRD`, `RBAC`)
- Receiver: short (1-2 chars), consistent. Package: short, lowercase, no underscores
- Test helpers: prefix `setup` or `new` (e.g., `newFakeClient`)

### Error Handling

- Wrap with context: `fmt.Errorf("context: %w", err)`
- Sentinel errors for known conditions; `CriticalError`/`RecoverableError` for classification
- No panics; `defer` for cleanup; `errors.Is`/`errors.As` for inspection; `errors.Join` for aggregation

### Logging

- `zerolog` structured JSON; levels: `debug` (dev), `info` (operational), `warn` (recoverable), `error` (unrecoverable)
- **Info** = business events + one final structured summary; **Debug** = troubleshooting only
- NEVER log secrets, tokens, or credentials

### Linting Thresholds

- Cyclomatic complexity >12 (`gocyclo`), cognitive >20 (`gocognit`)
- Functions >200 lines or >50 statements (`funlen`); duplicates >100 tokens (`dupl`)
- Magic numbers flagged (`mnd`) — use constants; test files exempt
- `//nolint:linter // reason` required for overrides

### Security

- Validate external input (webhooks, CRD fields). Use CEL for declarative validation.
- Context for cancellation/timeouts. No secrets in logs or errors.
- HTTP/2 disabled (CVE). Metrics endpoint secured. RBAC least privilege (`+kubebuilder:rbac`).
