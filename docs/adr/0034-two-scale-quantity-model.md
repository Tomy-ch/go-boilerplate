---
status: accepted
date: 2026-07-24
deciders: [maintainers]
tags: [architecture, persistence, http]
---

# ADR-0034: Hold a quantity in two scales — exact decimal for precision, integer minor unit for settlement

## Status

accepted

## Context

A domain routinely carries two quantities that look like one concept — "money" — but whose
precision requirements point in opposite directions.

- One of them **must** admit values finer than the smallest indivisible unit. A unit price below
  one cent is ordinary for metered, wholesale, and FX-derived goods, and a representation that
  cannot hold it is simply wrong for that value.
- The other **must not**. A figure that is actually settled — charged, invoiced, transferred —
  cannot be expressible below the minor unit, because such a figure is an invalid state rather
  than a merely unusual one.

A single representation cannot both admit and forbid sub-minor-unit precision. Making everything
fine-grained loses the guarantee; making everything coarse loses the value.

Underneath both sits a second problem: `float64` cannot represent decimal fractions such as `0.1`
or `19.99` exactly, so any such quantity carried as a float is already corrupted at parse time,
before any arithmetic runs. Deciding the two scales therefore also means deciding what the exact
container is, how it is stored, and how it crosses the wire without being re-corrupted at the
boundary.

## Decision

**Two quantities with different precision requirements are held as different types.** One is an
exact base-10 decimal; the other is an integer count of the currency minor unit. The choice of
type *is* the guarantee: a settlement figure is not representable below the minor unit **by
construction**, so the invalid state cannot be built at all rather than being built and then
rejected. Which concrete quantities live in which scale is feature content and is specified with
the feature, not here.

**The lossy conversion between the scales happens at exactly one point.** Rounding is applied once,
at the boundary where an exact quantity becomes a settled figure, in a single function reused
across endpoints. Callers never round, so the policy cannot drift between two call sites and a
settled figure cannot silently disagree with the stored quantity it derives from.

**The rounding mode and the digit count are a policy owned by the usecase / value object, never
baked into the generic decimal mechanism.** `pkg/decimal` offers arithmetic, rounding modes, and
scale conversion (`ToScaledInt64`) and holds no opinion about which currency has how many minor-unit
digits or which mode a given figure needs; both are supplied by the layer that knows the business
meaning. A mechanism that decided them would force every future policy change through a change to a
generic `pkg/` package.

**The database column for the exact scale is `NUMERIC` with no precision or scale.** Scale is a
property of the value, not a design-time constant, so the schema does not assert a decimal exponent
that cannot be justified at design time: for money it would have to be the minor-unit digit count of
whichever currency is in play, which the column cannot know. The settlement scale is a plain integer
column.

**On the wire, an exact-decimal value is a JSON string** (`"19.99"`), never a JSON number, and the
OpenAPI schema types those fields as `type: string` with a decimal `pattern`. A JSON number is
decoded by typical parsers as an IEEE754 `double`, which silently undoes the exactness that the rest
of the decision buys; a string is the only lossless JSON representation. On decode the container
also accepts a bare JSON number, so an external payload that emits one is ingested without digit
loss.

**At the persistence boundary the same container carries itself.** It implements `sql.Scanner` /
`driver.Valuer` and the sqlc override maps `NUMERIC` columns to it, so generated infrastructure code
never names the underlying vendor type either. That the vendor sits behind a `pkg/` seam at all is
the lock-in-avoidance pattern recorded in [ADR-0001](0001-avoid-lock-in.md); what is specific here is
that the seam must also cover the DB and wire boundaries, or the vendor type reappears in generated
code.

## Consequences

### Positive Consequences

- A sub-minor-unit quantity is representable where it is legitimate, and a sub-minor-unit *settled
  figure* is not representable at all — each state is enforced by the chosen representation rather
  than by convention or by a validation that someone must remember to call.
- Rounding is localized to one boundary, so a settled figure cannot drift from the quantity it was
  derived from, and a change of policy is a change in one place.
- `NUMERIC` without a fixed scale keeps the schema honest: no currency-specific decimal exponent is
  asserted at design time.
- The value is lossless end to end — string on the wire, exact decimal in memory, `NUMERIC` at rest —
  with no float anywhere on the path.
- The generic mechanism stays free of business meaning, so `pkg/` keeps satisfying its own
  independence bar.

### Negative Consequences

- Two representations for one apparent concept raise the conceptual bar: a contributor must know
  which scale a given quantity lives in before touching it. This is what the per-feature spec and
  the per-layer READMEs have to state.
- An unspecified `NUMERIC` admits arbitrarily long fractions at the database level; what actually
  bounds precision at use sites is the domain value object and the single conversion point, not the
  column.
- Clients must treat the field as a string rather than a number — a visible shape in the API
  contract, and a migration cost for any existing client.
- A `NUMERIC` → string → decimal round trip is marginally more work than a native numeric type,
  accepted as the price of exactness.

## Alternatives Considered

### One scale — integer minor units everywhere

Rejected: it cannot express a value below the minor unit at all, which is the motivating
requirement. This is the status quo the two-scale model replaces.

### One scale — exact decimal everywhere

Rejected: it makes a sub-minor-unit *settled figure* representable again, reintroducing exactly the
invalid state the second scale exists to forbid. Uniformity is not worth losing that guarantee.

### `float64` for both

Rejected outright: `float64` cannot hold `0.1` or `19.99` exactly, so the value is corrupted at
parse time and every subsequent operation accumulates the error. This is the defect the decision
removes, not a trade-off to weigh.

### Fix `NUMERIC(precision, scale)` per column

Rejected: it bakes a unit-specific decimal exponent into the schema as a design-time constant,
contradicting "scale is a property of the value" and committing the column to an assumption it
cannot justify for a currency it has never seen.

### Keep the wire as a JSON number (`format: decimal` / `double`)

Rejected: common tooling decodes JSON numbers as IEEE754 doubles, reintroducing the float corruption
being removed at the one boundary the application does not control. A string is the only lossless
JSON representation of an exact decimal.

### `big.Rat` (true rationals)

Rejected as over-modeling: prices, rates, and taxes are decimal, not arbitrary rationals. `big.Rat`
would add complexity and non-terminating representations with no benefit for these quantities.

### Bake the rounding mode and minor-unit digit count into the decimal container

Rejected: it would give a generic `pkg/` container business meaning, breaking the rule that `pkg/`
holds no feature-specific logic, and would route every future policy change through the shared
mechanism instead of through the layer that owns the policy.

## Notes

- Container: `pkg/decimal`; the vendor-behind-`pkg/` pattern it follows is
  [ADR-0001](0001-avoid-lock-in.md), and the `pkg/` independence rule it must satisfy is in
  [`docs/rules.md`](../rules.md).
- Where a shared, business-semantic money value object lives:
  [ADR-0035](0035-domain-lexicon.md).
- Which quantities occupy which scale, the concrete minor-unit digit counts, the rounding mode
  chosen for each figure, and the settlement currency are feature content, specified with the
  feature. In this repository that means the removable sample set (`docs/spec/purchase/`,
  `docs/spec/exchange-rate/`) — referenced by path rather than linked, because those files are
  deleted by `make setup-remove-sample-api` while this ADR stays.
