# money

English | [日本語](README.ja.md)

Provides money arithmetic helpers for the Usecase tier. Money is held as **minor-unit
integers** (e.g. USD cents, JPY yen) to avoid `float` accumulation error; the rounding method
is an application convention fixed by ADR.

## Public API

- `ApplyRateHalfUp(amountMinor int64, rate float64, scale int64) int64` — applies `rate` to a
  minor-unit integer and divides by `scale` (the fixed-point scale used to derive `amountMinor`;
  100 when the amount was cent-scaled), rounding **half-away-from-zero** at the final division.
  The rate is mapped to a 10^6 fixed-point integer and the intermediate product is computed with
  `math/big`, so the `int64` multiplication cannot overflow; rounding happens exactly once.
  `amountMinor` / `rate` may be any sign (negatives round away from zero); `scale` must be
  a positive integer (`<= 0` panics via division by zero).

## Design policy

- Rounding is centralized here so the policy cannot drift across call sites; callers never round.
- The half-up method and the single-rounding-point rule are recorded in
  [ADR-0099](../../../../docs/adr/0099-reference-amount-half-up-rounding.md).
- No infrastructure dependencies; a mechanical transformation only.
