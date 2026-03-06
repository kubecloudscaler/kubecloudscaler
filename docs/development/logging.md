# Logging Strategy

This document defines how to write readable, relevant logs and keep volume low while preserving maximum information for debugging and operations.

## Principles

1. **One log per outcome** — Prefer one log at the end of an operation with result fields instead of "starting X" and "X completed".
2. **Structured fields** — Use zerolog fields (`.Str()`, `.Int()`, `.Err()`) so logs are queryable; avoid `.Msgf()` for multi-line or ad-hoc formatting.
3. **Info = one line per meaningful event** — At default level, operators see one line per reconcile, per handler result, or per scaling summary.
4. **Debug = detail for troubleshooting** — Use Debug for per-item iteration, internal state, or when Info already summarizes the outcome.

## Levels

| Level  | Use for |
|--------|--------|
| **Error** | Unrecoverable failure (auth, invalid config, persistent API error). Always include `.Err(err)`. |
| **Warn** | Recoverable condition (transient error, requeue). |
| **Info** | Meaningful outcome: reconcile started/done, period chosen, scaling summary, finalizer added/removed, status updated. Include key identifiers (name, namespace, period, counts). |
| **Debug** | Verbose detail: list of resources, each period checked, per-resource scale step. Omit if an Info at the end already carries the same information. |

## Where to log

- **Controllers** — One Info at reconcile start with `name`/`namespace` (or one at end with `result` + `requeue_after`). One Error on failure.
- **Handlers** — One Info per handler with outcome (e.g. "period determined" with `period`/`type`, "scaling completed" with `success_count`/`failed_count`). One Error on failure. No Debug "entering handler" unless it adds context not present in the outcome log.
- **Utils (e.g. SetActivePeriod)** — One Debug summary for period iteration (e.g. `active_period` when found); one Error on parse/load failure. No per-period Debug lines in tight loops.
- **pkg/resources (processor/strategies)** — One Debug per scaling run with kind and counts; avoid per-resource Debug (scale up/down/restore) unless Debug level is explicitly needed for that resource type.

## Format

- **Message**: Short, present tense or past tense for outcome. Examples: `"period determined"`, `"scaling completed"`, `"unable to fetch Scaler"`.
- **Fields**: Prefer consistent names: `name`, `namespace`, `period`, `type`, `success_count`, `failed_count`, `requeue_after`, `result`.
- **No secrets** — Never log credentials, tokens, or secret contents.

## Examples

```go
// Good: one Info with outcome
ctx.Logger.Info().
    Str("period", ctx.Period.Name).Str("type", ctx.Period.Type).
    Msg("period determined")

// Good: Error with context
ctx.Logger.Error().Err(err).Str("resource", name).Msg("unable to set resource state")

// Avoid: two logs for one operation
ctx.Logger.Debug().Msg("validating period")       // redundant
// ... work ...
ctx.Logger.Info().Msg("period validated")         // keep this one with fields

// Avoid: per-iteration Debug in hot path
for _, p := range periods {
    logger.Debug().Msgf("checking period: %s", p.Name)  // use one Debug with summary instead
}
```

## Reference

- [AGENTS.md § IV. Observability & Structured Logging](../../AGENTS.md) — Constitutional rules.
- Zerolog: [github.com/rs/zerolog](https://github.com/rs/zerolog).
