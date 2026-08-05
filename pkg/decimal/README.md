# decimal

English | [日本語](README.ja.md)

Exact-decimal type wrapping `github.com/shopspring/decimal`. Represents money, rates, and other precise decimal quantities without `float64` error.

## Role

`float64` cannot represent decimal fractions such as `0.1` or `19.99` exactly, so any money or rate carried as a float is corrupted the moment it is parsed. This package provides a base-10 exact quantity (big-integer coefficient + exponent) that every layer reaches for instead, and hides the vendor dependency behind a seam so application code never imports `shopspring/decimal` directly (the `pkg/uuid` precedent).

It carries **no** business semantics — currency, non-negativity, and minor-unit choice live in the domain layer's value objects (`internal/domain/lexicon/money`). This package is pure decimal arithmetic, rounding, scaling, and the DB / wire boundary.

## Wraps

`github.com/shopspring/decimal`

## Notes

- Wire representation is a **JSON string** (`"19.99"`): a JSON number is decoded as an IEEE754 double and loses precision. `UnmarshalJSON` also accepts a JSON number so external payloads that emit bare numbers are ingested without digit loss.
- `ToScaledInt64(n)` rounds to `n` places half-away-from-zero, multiplies by `10^n`, and returns the minor-unit integer, or `ErrOverflow` if the result exceeds `int64`. This is the generic mechanism; *which* `n` (the minor-unit digit count) is a policy owned by the caller.
- Implements `sql.Scanner` / `driver.Valuer` for the `NUMERIC` boundary; the sqlc override aligns `NUMERIC` columns with this type.
- `MustParse` is for testing only — do not use in production.
- Depends on `pkg/xerrors` for error wrapping — the sole permitted `pkg/` → `pkg/` dependency (enforced by depguard `independent_pkg`).
