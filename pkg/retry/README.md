# retry

English | [日本語](README.ja.md)

A bounded-retry behavior layer that *consumes* a failure classification: `classify → bounded attempts → backoff + full jitter → deadline-aware`. Implemented once and shared by callers such as transaction retries and resilient outbound HTTP.

## Why this package

`pkg/backoff` computes wait durations as a pure function of the attempt count, free of clock or randomness. The full-jitter step needs `math/rand/v2`, so it lives here instead — keeping `backoff` pure while the randomness dependency is confined to `retry`.

`pkg/` packages are mutually independent (no pkg → pkg imports), so `Policy.Backoff` is taken as a **function value** (`attempt → base duration`) rather than a `backoff.Exponential`; the caller (in `internal/`) wires `backoff.Exponential.Duration` or any equivalent.

## API

|Symbol|Description|
|---|---|
|`Do(ctx, sleeper, policy, isRetryable, fn)`|Run `fn` with bounded retries; retry while `isRetryable(err)` holds, sleeping `policy.Backoff` (+ full jitter) between attempts|
|`Full(d)`|Return a uniform random duration in `[0, d]` (full jitter); `0` when `d <= 0`|
|`Policy` (struct)|`MaxAttempts` + `Backoff` (a `func(attempt int) time.Duration`; `nil` means no wait)|
|`Sleeper` (interface)|`Sleep(ctx, d) error` — wait abstraction; structurally satisfied by `internal/usecase/boundary/clock.Sleeper`|

## Notes

- `Do` runs `fn` at least once (`MaxAttempts < 1` is treated as `1`) and returns the last observed error (`nil` on success).
- `isRetryable` is only consulted for a non-nil `fn` error.
- When `sleeper.Sleep` returns an error (ctx canceled / deadline), `Do` returns the **preceding `fn` error**, not the sleep error — the original retryable failure is surfaced to the caller.
- `Sleeper` is defined locally so `pkg/` stays free of `internal/` dependencies; the boundary `clock.Sleeper` satisfies it structurally.

## Wraps

Standard library `context`, `time`, and `math/rand/v2`. No other package dependencies.
