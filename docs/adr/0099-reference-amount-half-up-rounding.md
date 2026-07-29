---
status: superseded
date: 2026-07-22
deciders: [maintainers]
superseded-by: 0103
tags: [architecture]
---

# ADR-0099: referenceAmount is computed in integers with half-up rounding at a single point

## Status

superseded by [ADR-0103](0103-decimal-half-up-rounding.md)

## Context

`GET /v1/exchange-rates` returns, when `displayCurrency` is requested, a `referenceAmount`
— a display-currency (JPY) figure derived from the base amount and the fetched rate. Money
must not accumulate `float` error, so the internal computation for `referenceAmount` must be
integer-based. Two things need deciding: the rounding method for the final integer, and where
the single float-to-integer normalization is allowed to happen (the query `amount` is a
`float64` multiplier, so *some* normalization point exists).

## Decision

The `referenceAmount.amount` field (a JPY-yen integer) is computed with **half-up rounding
applied exactly once**, at the final division. The computation is centralized in
`internal/usecase/tools/money.ApplyRateHalfUp(amountMinor int64, rate float64, scale int64) int64`,
which converts the rate to a 10^6 fixed-point integer and then performs integer-only
arithmetic; callers never round.

The **single input-normalization point** is in the usecase: `amountMinor =
int64(math.Round(amount * baseMinorUnitScale))` (`baseMinorUnitScale = 100` for USD cents),
performed once before `ApplyRateHalfUp`. Downstream of this point the reference computation is
pure integer arithmetic with no `float` money value.

The query `amount` remains `type: number, format: double` (OpenAPI): it is a conversion input
multiplier, not a stored money value. The "no float money" rule applies to **stored values and
the referenceAmount computation path only**, not to this input multiplier or to the generic
`converted` output.

Half-up is chosen because `referenceAmount` is an advisory, non-persistent display figure not
subject to accounting/tax constraints. If such a requirement later arises, this ADR is revised.

## Consequences

### Positive Consequences

- No `float` accumulation on the money path; the reference figure is deterministic.
- Rounding lives in one function reused by later APIs (`POST /v1/purchases` etc.), so the
  policy cannot drift across endpoints.
- The float exception (query `amount`, `converted`) is explicit and bounded.

### Negative Consequences

- `amount` is quantized to 2 decimal places (`amount * 100`) before rounding, so a base amount
  carrying more than 2 significant decimal places loses sub-cent precision in the reference figure.
  The conversion's order of magnitude is unaffected — the `×100` and the `÷100` inside
  `ApplyRateHalfUp` cancel, so the result is `≈ round(amount * rate)` regardless of the base
  currency's decimal exponent; this is a minor precision artifact, not a scale error.
- Half-up differs from banker's rounding; a future accounting requirement would need a revision.

## Alternatives Considered

### Integerize the query `amount` (USD cents) at the API boundary

Rejected: the endpoint is a generic any-pair conversion demo, and a minor-unit integer query
would require a per-currency exponent table (JPY=0 / USD=2 / BHD=3 …). That is over-built for
referenceAmount integerization, and later `purchases` bring their own stored USD-cent value
rather than using this query. The non-negotiable "stored values and reference path are integer"
is fully met without it.

### Banker's rounding (round half to even)

Weighed for accounting neutrality, but `referenceAmount` is advisory and non-persistent, so
the bias argument does not apply. Half-up is simpler to reason about for a display figure.

## Notes

- Helper: `internal/usecase/tools/money` (`ApplyRateHalfUp`).
- Assembly of the reference DTO: `exchangerate.BuildReferenceAmount` (single place, reused by
  later APIs).
- Cache that supplies the rate: [ADR-0098](0098-exchange-rate-cache-gateway-decorator.md).
