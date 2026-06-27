# clock

English | [日本語](README.ja.md)

Provides a `Clock` interface for retrieving the current time and a `Sleeper` interface for waiting.

```go
type Clock interface {
    Now() time.Time
}

type Sleeper interface {
    Sleep(ctx context.Context, d time.Duration) error
}
```

## Why Abstract?

- Ensure testability of time-dependent logic (TTL, expiration, scheduling)
- Prevent Domain / Usecase from depending directly on `time.Now()`
- Allow mock substitution in tests for deterministic behavior
- `Sleeper` makes backoff waits injectable so retries can be tested without real sleeping. Consumers: the resilient HTTP client (`internal/infrastructure/httpclient`) and the transaction manager's retry (`internal/infrastructure/rdb/driver`).

## Implementation

`internal/infrastructure/system/` provides the concrete implementations that call `time.Now()` / `time.After` (ctx-aware).
