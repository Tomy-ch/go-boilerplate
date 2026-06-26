# backoff

English | [日本語](README.ja.md)

Computes exponential backoff wait durations as a pure function of the attempt count, free of clock or randomness dependencies.

## Notes

- `Duration` computes `Initial * Multiplier^attempt`, capping the result at `Max` when `Max` is positive.
- `Initial <= 0` yields `0`; a negative `attempt` is treated as `0`.
- A `Multiplier` below `1` is treated as `1`.
- With no cap (`Max <= 0`), the result is clamped to `math.MaxInt64` to avoid overflow producing a negative duration.

## Wraps

Standard library `time` and `math` packages.
