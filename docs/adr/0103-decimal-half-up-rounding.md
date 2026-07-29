---
status: accepted
date: 2026-07-24
deciders: [maintainers]
supersedes: 0099
tags: [architecture]
---

# ADR-0103: referenceAmount and rate application round half-up at a single point, on exact decimals

## Status

accepted (supersedes [ADR-0099](0099-reference-amount-half-up-rounding.md))

## Context

[ADR-0099](0099-reference-amount-half-up-rounding.md) fixed the rounding policy for
`GET /v1/exchange-rates`' `referenceAmount`: half-up, applied exactly once, on integer-based
arithmetic. Its *decision content* — half-up, single point — is still correct. But its
*mechanism* has been overtaken by the exact-decimal work ([ADR-0101](0101-two-scale-money-model.md),
[ADR-0102](0102-exact-decimal-pkg-wrap.md)): the multiplier rate was still a `float64`
(`boundary.Rate.Value`, the gateway's parsed response, and `ApplyRateHalfUp(rate float64)`), and
the query `amount` / `converted` output were still JSON numbers. Float remained on the money/rate
path — the exact hole this ADR closes. ADR-0099 is therefore superseded rather than edited.

## Decision

`referenceAmount.amount` is still computed with **half-up rounding applied exactly once**, and
that rounding still lives in **one function reused across endpoints**
(`internal/usecase/tools/money.ApplyRateHalfUp`). What changes is that the whole money/rate path
is now **exact decimal, with `float64` fully removed**:

- The rate (`boundary.Rate.Value`), the gateway's parsed provider response, and the multiplier
  input carried by `ApplyRateHalfUp` are all `pkg/decimal.Decimal`.
- `ApplyRateHalfUp(amount, rate decimal.Decimal, minorUnitDigits int32) (int64, error)` computes
  `amount × rate` exactly, then rounds to `minorUnitDigits` half-away-from-zero and scales to the
  minor-unit `int64` in one step (`decimal.ToScaledInt64`), returning an error on overflow.
  There is no float-to-integer normalization anymore, so the ADR-0099 sub-cent quantization
  artifact (`amount × 100`) is gone.
- `minorUnitDigits` is a **policy the usecase owns** (JPY = 0 today), not a constant baked into
  the mechanism.
- On the wire, the query `amount`, the `converted` output, and the reference `rate` are **decimal
  strings** ([ADR-0102](0102-exact-decimal-pkg-wrap.md)), not JSON numbers. The ADR-0099 float
  exception for `amount` / `converted` no longer exists.

Half-up remains the choice because `referenceAmount` is an advisory, non-persistent display
figure not subject to accounting/tax constraints; a future accounting requirement would revise
this ADR.

## Consequences

### Positive Consequences

- No `float64` anywhere on the money or rate path — the exactness guarantee is end-to-end, from
  the provider response through to the stored / displayed value.
- Rounding still lives in one reusable function, so the policy cannot drift across endpoints
  (`exchangerate.BuildReferenceAmount`, later `purchases`).
- The former sub-cent input-quantization artifact is eliminated; the reference figure is the
  exact `round(amount × rate)` to the display minor unit.

### Negative Consequences

- `converted` and the query `amount` are now strings, a breaking change for
  `GET /v1/exchange-rates` clients (shared with [ADR-0102](0102-exact-decimal-pkg-wrap.md)).
- Half-up still differs from banker's rounding; a future accounting requirement would need a
  revision, as under ADR-0099.

## Alternatives Considered

### Keep ADR-0099's float mechanism and only change storage

Rejected: it leaves the multiplier rate and the wire figures as floats, so the money path keeps
corrupting values at the exact points the two-scale work set out to fix.

### Banker's rounding (round half to even)

Weighed as under ADR-0099 and rejected for the same reason: `referenceAmount` is advisory and
non-persistent, so the bias argument does not apply; half-up is simpler to reason about.

## Notes

- Supersedes [ADR-0099](0099-reference-amount-half-up-rounding.md).
- Helper: `internal/usecase/tools/money.ApplyRateHalfUp`; generic scaling:
  `pkg/decimal.ToScaledInt64`.
- Reference DTO assembly: `exchangerate.BuildReferenceAmount` (single place).
- Decimal container / wire contract: [ADR-0102](0102-exact-decimal-pkg-wrap.md).
- Two-scale model: [ADR-0101](0101-two-scale-money-model.md).
