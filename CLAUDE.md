# CLAUDE.md

KubeCloudScaler — Kubernetes operator scaling cloud resources (K8s workloads, GCP VMs) based on time periods via custom CRDs. Subdirectory `CLAUDE.md` files provide context-specific rules.

## Constitutional Principles

From [.specify/memory/constitution.md](.specify/memory/constitution.md) — MANDATORY, supersede all other practices:

1. **Clean Architecture** — Layer separation: `cmd/`, `internal/controller/`, `internal/controller/*/service/handlers/`, `internal/repository/`, `internal/model/`, `api/`, `pkg/`. Repository Pattern for data access. Interfaces, not concrete types.
2. **Interface-Driven DI** — All deps as interfaces, injected via constructors. ISP + DIP. No global state.
3. **TDD** — Mandatory for `internal/controller/*/service/`. Red-Green-Refactor. 80% coverage (CI enforced).
4. **Observability** — `zerolog` (app) / `zap` (controller-runtime). Prometheus metrics (`kubecloudscaler_`). Never log secrets. See [docs/logging-strategy.md](docs/logging-strategy.md), [docs/metrics.md](docs/metrics.md).
5. **Go 1.25+ Style** — Idiomatic Go, 140 char lines, `golangci-lint` v2. Modern features: `range` over ints, `slices`/`maps`/`cmp`, generics, `errors.Join`.
6. **Error Handling** — Wrapped errors (`%w`), sentinels, `CriticalError`/`RecoverableError`. No panics. `errors.Is`/`errors.As`.
7. **Security** — No secrets in logs. K8s security best practices. RBAC least privilege. HTTP/2 disabled (CVE). TLS hot-reload.
8. **Performance** — Memory <100MB, CPU <100m. Informer caches. `prealloc`. `pprof`/benchmarks.

## Commands

```bash
make build              # Build binary
make test               # Unit tests (envtest)
make test-coverage      # HTML coverage report
make test-e2e           # E2E (Kind cluster)
make lint               # golangci-lint v2
make lint-fix           # Auto-fix
make generate           # DeepCopy code
make manifests          # CRDs/RBAC/webhooks
make run                # Run locally
make install            # Install CRDs
make deploy             # Deploy controller
```

## Workflow

- Conventional commits: `feat:`, `fix:`, `chore:`, `docs:`, `refactor:`, `test:`
- PRs require: constitutional compliance, tests, 80%+ coverage, lint pass, generated code up to date
- After changes: `make test` -> `make lint` -> `make generate && make manifests` (if CRDs changed)

## Agent Rules

- Read code before proposing changes. Use LSP tools for symbol queries.
- No over-engineering, no unrequested features, no premature abstractions
- No business logic in reconcilers — use handler chain (see `internal/controller/CLAUDE.md`)
- No `Update` when `Patch`/SSA is safer. No `init()` except scheme registration.
