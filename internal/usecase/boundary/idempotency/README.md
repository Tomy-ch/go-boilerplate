# idempotency

English | [日本語](README.ja.md)

Defines the `Store` persistence boundary for idempotency keys (claim / replay /
409 / 422 decisions). Every method requires a `scope` (no id-only lookup, to
prevent cross-boundary access).

```go
type Store interface {
    Claim(ctx context.Context, p ClaimParams) (claimed bool, err error)
    Get(ctx context.Context, scope, key string) (*Record, error)
    Complete(ctx context.Context, p CompleteParams) error
    DeleteExpired(ctx context.Context, cutoff time.Time, limit int32) (int64, error)
}
```

## Why Abstract?

- Ensure testability of replay / conflict logic without a real database
- Keep Usecase depending on a persistence port, not on sqlc or SQL details
- Allow mock substitution in tests for deterministic behavior
- Express concurrency failures as a boundary sentinel (`ErrLockTimeout`, mapped to 409 by the usecase)

## Implementation

`internal/infrastructure/rdb/system_query/idempotency/` provides the concrete RDB implementation backed by sqlc-generated queries.
