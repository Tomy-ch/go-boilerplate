---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [persistence, transaction, concurrency]
---

# ADR-0029: Retry transactions on serialization conflict; require callers to be idempotent

## Status

accepted

## Context

PostgreSQL's REPEATABLE READ and SERIALIZABLE isolation levels can abort transactions with:

- `serialization_failure` (SQLSTATE 40001) — concurrent transactions produced a
  serialization anomaly
- `deadlock_detected` (SQLSTATE 40P01) — a circular wait was broken by aborting one
  transaction

Both are expected conditions under concurrent write load, not application bugs. PostgreSQL's
own documentation recommends retrying these transactions at the application layer.

Without automatic retry, these errors propagate as transient failures that every usecase
must handle individually. The resulting boilerplate is inconsistent, error-prone, and easy
to omit.

However, automatic retry introduces a subtle safety constraint: the transaction body
function `fn` may be executed multiple times if the transaction is retried. Any non-database
side effect inside `fn` — sending an email, publishing a message to a broker, calling an
external API — would also execute multiple times, producing duplicates. An unconditional
retry guarantee is unsound without a corresponding constraint on what `fn` may do.

## Decision

`tx.Manager.Do` (implemented in `internal/infrastructure/rdb/driver/transaction.go`)
**automatically retries the full transaction body** a bounded number of times when
`serialization_failure` (40001) or `deadlock_detected` (40P01) is detected, using
exponential backoff with full jitter via `pkg/retry` and `pkg/backoff`.

Retry parameters are configurable without code changes:

| Config key | Default |
| --- | --- |
| `DB_TX_MAX_RETRIES` | 3 attempts |
| `DB_TX_RETRY_BASE_BACKOFF` | 5 ms initial |
| `DB_TX_RETRY_MAX_BACKOFF` | 100 ms cap |

**Callers must ensure `fn` is idempotent for all non-database side effects**, because `fn`
may run up to `maxAttempts` times. The canonical pattern for external side effects is to
write them as **outbox rows within the same transaction**: a rolled-back attempt discards
the outbox rows, and only the committed attempt's rows are delivered downstream, making
retries safe by construction.

Nested `Manager.Do` calls (where a transaction already exists in context) reuse the outer
transaction and run exactly once — they are not retried. Only the outermost `Do` call
retries.

The retry predicate inspects the raw error chain using `errors.As` to find
`*pgconn.PgError`, which remains accessible even after `pgerror.NormalizeError` because
the project uses `xerrors.Join` rather than string-flattening wraps (see
[`docs/rules.md`](../rules.md) § "Error Handling Rules").

## Consequences

### Positive Consequences

- Serialization and deadlock errors are handled automatically and consistently. Usecases do
  not need per-call retry boilerplate.
- Retry parameters are tunable per deployment without code changes.
- The outbox pattern (write side effects transactionally) makes the idempotency constraint
  easy to satisfy and auditable: retry safety is visible in the schema, not hidden in
  application code.
- Nested transactions are safe: only the outermost `Do` retries; inner calls inherit the
  outer transaction and run once.

### Negative Consequences

- Callers must actively understand and satisfy the idempotency constraint. A violation (e.g.
  publishing to a broker directly inside `fn` rather than via an outbox row) produces
  silent duplicates that are hard to detect in testing.
- Under high contention, retry adds latency and can hold connections for longer than a
  single attempt. A high `maxAttempts` without backoff cap could worsen contention.
- The idempotency contract is enforced by convention and code review, not by a type-system
  guarantee.

## Alternatives Considered

### No automatic retry — let callers handle it

Each usecase or handler would implement its own retry loop on detecting 40001 or 40P01.
Error-prone (easy to forget), inconsistent (different retry counts and backoff strategies),
and verbose. Serialization failures are the expected mechanism for conflict resolution under
SERIALIZABLE isolation — surfacing them directly to callers is architectural noise that
belongs in the infrastructure layer.

### Unlimited retries

Without a bound, a high-contention scenario can starve other transactions indefinitely via
repeated failed attempts. A bounded retry with exponential backoff is the standard safe
pattern and matches PostgreSQL's own recommendation to retry a finite number of times before
returning an error.

### Opt-in retry (separate DoWithRetry method)

Callers explicitly choose `Manager.DoWithRetry` when they want retry behavior.
More explicit but adds friction to the common case. All production transactions should be
retry-safe by default (side effects via outbox), so opt-in creates pressure to use the
non-retrying path as a shortcut rather than a deliberate choice. An opt-out model (retry is
the default, nested path is opt-out automatically) is more appropriate.

### Savepoint-based nested retries

Retry nested `Do` calls independently using PostgreSQL savepoints. Adds significant
complexity for the nested case; the outer transaction's retry is already sufficient because
the whole transaction (including nested calls) is retried atomically.

## Notes

- Source: [`internal/usecase/boundary/tx/README.md`](../../internal/usecase/boundary/tx/README.md)
  § "Notes" (retry and idempotency contract).
- Source: [`internal/infrastructure/rdb/driver/transaction.go`](../../internal/infrastructure/rdb/driver/transaction.go)
  — `Do` (retry loop) and `doOnce` (single attempt, returns raw error for retry predicate).
- Related: retry and backoff are implemented in `pkg/retry` and `pkg/backoff`.
