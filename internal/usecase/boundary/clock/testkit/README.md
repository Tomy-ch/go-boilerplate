# clock/testkit

English | [日本語](README.ja.md)

Test doubles for the `clock` boundary (`Clock` / `Sleeper`), so time-dependent
logic (TTL, deadlines, retry / backoff) can be verified **deterministically**
without real time or real sleeping.

## Mock helpers

Thin constructors over the generated gomock mocks (`clock/mock`):

- `NewMockClock(t, now) clock.Clock` — always returns `now` from `Now()` (call
  count unconstrained).
- `NewMockClockOnce(t, now) clock.Clock` — returns `now`, asserting `Now()` is
  called exactly once.
- `NewNoopSleeper(t) clock.Sleeper` — `Sleep` returns `nil` immediately without
  waiting (call count unconstrained).

## `StepClock`

`NewStepClock(start, step) *StepClock` is a deterministic implementation of both
`clock.Clock` and `clock.Sleeper` that advances the fake clock by a fixed `step`
on every `Sleep`, regardless of the requested wait duration `d`. This decouples
retry / deadline discipline from wall-clock time and from backoff jitter — even
when jitter varies `d`, the clock advances by a constant amount.

- `Now() time.Time` — returns the current fake time.
- `Sleep(ctx, d) error` — never actually waits: if `ctx` is already
  cancelled / expired it returns that error **without** advancing the clock;
  otherwise it advances by `step` and returns `nil`.

It is safe for concurrent use (guarded by a mutex).
