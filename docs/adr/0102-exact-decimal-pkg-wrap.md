---
status: accepted
date: 2026-07-24
deciders: [maintainers]
tags: [architecture, http, persistence]
---

# ADR-0102: Exact-decimal quantities use a `pkg/decimal` wrapper and a string wire contract

## Status

accepted

## Context

The two-scale money model ([ADR-0101](0101-two-scale-money-model.md)) needs an exact base-10
quantity type for pricing-scale values and for exchange rates. `float64` cannot represent
decimal fractions such as `0.1` or `19.99` exactly, so any money or rate carried as a float is
corrupted at parse time — the very defect this work removes. A decimal library
(`shopspring/decimal`) provides the arithmetic, but two questions remain: how the application
depends on it, and how such values cross the HTTP wire without being re-corrupted.

## Decision

**Container.** Exact-decimal quantities are represented by a new `pkg/decimal.Decimal` that
**wraps `github.com/shopspring/decimal`** and hides the vendor behind a seam, following the
`pkg/uuid` precedent. Domain, usecase, controller, and infrastructure code depend on
`pkg/decimal` only; a **direct dependency on `shopspring/decimal` from any layer other than the
`pkg/decimal` seam is forbidden**. The wrapper carries no business semantics — currency,
non-negativity, and minor-unit choice live in higher-layer value objects — it provides only
arithmetic, rounding modes, scale conversion (`ToScaledInt64`), and the DB / wire boundary. It
does not import `internal/**` and does not break the `pkg/` mutual-independence rule (depguard
`independent_pkg`); its sole `pkg/`→`pkg/` dependency is the permitted `pkg/xerrors`.

**Wire contract.** A decimal value is represented on the HTTP wire as a **JSON string**
(`"19.99"`), never a JSON number. A JSON number is decoded by typical parsers as an IEEE754
`double` and loses precision, which would silently undo the exactness. The OpenAPI schema types
these fields as `type: string` with a decimal `pattern`. On decode, `pkg/decimal` also accepts a
bare JSON number (for external payloads that emit one, e.g. the exchange-rate provider) and
ingests it without digit loss.

**Persistence boundary.** `pkg/decimal.Decimal` implements `sql.Scanner` / `driver.Valuer`, and
the sqlc override maps `NUMERIC` columns to this type — so the generated infrastructure code also
never names the raw `shopspring` type.

## Consequences

### Positive Consequences

- A single vendor seam: the decimal library can be swapped by editing `pkg/decimal` alone, and
  no layer accretes a `shopspring` import (verifiable by grep / depguard).
- The wire is lossless end-to-end: string in, exact decimal internally, `NUMERIC` at rest.
- The generic mechanism (`ToScaledInt64`) is separated from money policy (which minor-unit
  digit count), keeping `pkg/decimal` free of business meaning.

### Negative Consequences

- Clients must treat the field as a string, not a number — a visible API contract change for
  `price`, the exchange-rate `amount` / `converted`, and the reference `rate`.
- A `NUMERIC`→string→decimal round trip is marginally more work than a native numeric, accepted
  as the cost of exactness.

## Alternatives Considered

### Depend on `shopspring/decimal` directly across layers

Rejected: it spreads a concrete vendor type through domain and usecase, contradicting the
`pkg/uuid` precedent and making a future swap a repo-wide change.

### Keep the wire as a JSON number (`format: decimal` / `double`)

Rejected: JSON numbers are decoded as IEEE754 doubles by common tooling, reintroducing exactly
the float corruption being removed. A string is the only lossless JSON representation.

### `big.Rat` (true rationals)

Rejected as over-modeling: prices, rates, and taxes are all decimal, not arbitrary rationals;
`big.Rat` would add complexity (and non-terminating representations) with no benefit here.

## Notes

- Package: `pkg/decimal` (wraps `github.com/shopspring/decimal`).
- sqlc override: `sqlc.yaml` (`pg_catalog.numeric` → `pkg/decimal.Decimal`).
- Two-scale model this serves: [ADR-0101](0101-two-scale-money-model.md).
- pkg independence rule: `docs/rules.md`, depguard `independent_pkg` in `.golangci-full.yaml`.
