# money

English | [日本語](README.ja.md)

Provides the money-settlement **policy** for the Usecase tier. The exact arithmetic, rounding,
and minor-unit scaling mechanism lives in [`pkg/decimal`](../../../../pkg/decimal/README.md);
this package only chooses *which* minor-unit digit count and *which* rounding mode a settlement
amount is reduced to.

## Public API

- `ApplyRateHalfUp(amount, rate decimal.Decimal, minorUnitDigits int32) (int64, error)` — computes
  `amount × rate` exactly, then rounds to `minorUnitDigits` **half-away-from-zero** and scales to
  the settlement-scale `int64` (`decimal.ToScaledInt64`). Rounding happens exactly once, at this
  single settlement boundary; there is no `float` on the path, so no accumulation error. Returns
  an error when the minor-unit integer exceeds `int64`.

## Design policy

- Rounding is centralized here so the policy cannot drift across call sites; callers never round.
- The half-up method and the single-rounding-point rule are recorded in
  [ADR-0036 (two-scale-quantity-model)](../../../../docs/adr/0036-two-scale-quantity-model.md); the concrete policy is stated with the feature that applies it (`docs/spec/exchange-rate/`, sample content).
- The generic decimal mechanism is [`pkg/decimal`](../../../../pkg/decimal/README.md); this
  package holds only the policy (`minorUnitDigits`, half-up), no infrastructure dependency.
