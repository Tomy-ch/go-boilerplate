---
status: accepted
date: 2026-07-24
deciders: [maintainers]
tags: [architecture, persistence]
---

# ADR-0101: Money is modeled in two scales — pricing (exact decimal) and settlement (integer minor unit)

## Status

accepted

## Context

Product pricing was stored as an integer number of USD cents (`products.price`,
`purchase_details.unit_price` as `INTEGER`). This cannot represent a **sub-cent unit price** —
a price below one minor unit — which real metered / wholesale / FX-derived goods routinely
carry. At the same time, a settlement figure (a subtotal, tax, or total that a customer is
actually charged) *must not* be expressible below one minor unit: a sub-cent charge is an
invalid state that the type system should make unrepresentable.

These two needs pull in opposite directions. A single representation cannot both admit sub-cent
precision (pricing) and forbid it (settlement). A decision is needed on how each kind of money
amount is stored and what the unit contract for each is.

This **revises decision point 2 of [ADR-0100](0100-purchase-stock-lock-and-amount-contract.md)**
("money is integer USD cents, everywhere"): the *pricing-scale* columns (`products.price`,
`purchase_details.unit_price`) move to exact decimal (USD major unit), while the *settlement-scale*
amounts (`purchases.{subtotal,tax,shipping_fee,total}`) keep the integer-USD-cents contract
ADR-0100 established. ADR-0100's stock-locking, truncation-at-the-settlement-boundary, and
CommandService decisions are unchanged.

## Decision

Money is modeled in **two scales**:

- **Pricing scale** — a unit price (`products.price`, `purchase_details.unit_price`) is an
  **exact decimal** with sub-cent precision. The database column is an **unspecified `NUMERIC`**
  (no `precision`/`scale`): scale is a property of the value, not a design-time constant, so it
  is not baked into the schema. In Go the value is `pkg/decimal.Decimal` (see
  [ADR-0102](0102-exact-decimal-pkg-wrap.md)), wrapped by a domain `money.Price` value object
  that owns non-negativity and minor-unit conversion.

- **Settlement scale** — a charged amount (`purchases.subtotal_amount` / `tax_amount` /
  `shipping_fee` / `total_amount`) stays an **integer number of the currency minor unit**
  (`INTEGER`), unchanged. Sub-cent settlement is not representable by construction.

Rounding from pricing scale to settlement scale happens at **exactly one settlement boundary**
and follows the rounding policy recorded in [ADR-0103](0103-decimal-half-up-rounding.md). The
minor-unit digit count for that rounding (e.g. JPY = 0, USD = 2) is a **policy owned by the
usecase / money value object**, not by the generic decimal mechanism.

Settlement currency stays **USD only**; JPY appears solely as the advisory, non-persistent
`referenceAmount` display conversion (see ADR-0102). Multi-currency settlement is out of scope.

## Consequences

### Positive Consequences

- A sub-cent unit price is representable; a sub-cent *charge* is not — each invalid/valid state
  is enforced by the chosen representation rather than by convention.
- `NUMERIC` without a fixed scale keeps the schema honest: the boilerplate does not assert a
  currency-specific decimal exponent it cannot justify.
- The pricing/settlement split localizes rounding to one boundary, so the charge a customer
  sees cannot silently drift from the stored price.

### Negative Consequences

- Two representations for "money" raise the conceptual bar: a contributor must know which scale
  a given amount lives in. This is documented here and in the per-layer READMEs.
- An unspecified `NUMERIC` admits arbitrarily long fractions at the DB level; the domain
  `money.Price` VO and the minor-unit conversion are what actually bound precision at use sites.

## Alternatives Considered

### Keep everything as integer minor units

Rejected: this is the status quo that cannot express a sub-cent unit price at all, which is the
whole motivation.

### Make settlement amounts decimal too (single scale for all money)

Rejected: it would make a sub-cent charge representable, reintroducing exactly the invalid state
the settlement scale exists to forbid. Uniformity is not worth losing that guarantee.

### Fix `NUMERIC(precision, scale)` per column

Rejected: it bakes a currency-specific decimal exponent into the schema (a design-time constant
the boilerplate cannot justify for an arbitrary future currency), contradicting "scale is a
property of the value".

## Notes

- Decimal container and wire contract: [ADR-0102](0102-exact-decimal-pkg-wrap.md).
- Rounding at the settlement boundary: [ADR-0103](0103-decimal-half-up-rounding.md).
- Migration: the pricing-scale `NUMERIC` type is defined directly in the create migrations
  `database/migrations/000010_create_products.up.sql` (`products.price`) and
  `000013_create_purchase_details.up.sql` (`purchase_details.unit_price`).
- Domain VO: `internal/domain/kernel/money.Price`.
