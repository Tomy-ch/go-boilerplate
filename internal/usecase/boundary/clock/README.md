# clock

English | [日本語](README.ja.md)

Provides a `Clock` interface for retrieving the current time.

```go
type Clock interface {
    Now() time.Time
}
```

## Why Abstract?

- Ensure testability of time-dependent logic (TTL, expiration, scheduling)
- Prevent Domain / Usecase from depending directly on `time.Now()`
- Allow mock substitution in tests for deterministic behavior

## Implementation

`internal/infrastructure/system/` provides the concrete implementation that calls `time.Now()`.
