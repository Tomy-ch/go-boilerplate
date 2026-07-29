---
status: accepted
date: 2026-07-23
deciders: [maintainers]
tags: [persistence, cqrs, architecture, money]
---

# ADR-0100: Purchase creation locks stock with SELECT FOR UPDATE and computes money as integer USD cents

## Status

accepted

## Context

`POST /v1/purchases` ([#571]) is this repository's first CommandService implementation
(the atomicity criterion itself is already settled by [ADR-0029]). Several residual contracts
that ADR-0029 does not fix need pinning so later purchase-domain PBIs
(#569 / #570 / #588-#591 / #593) and the stock-replenishment API (#592, which shares the same
rows) stay consistent:

- how concurrent stock decrements are serialized (purchase vs. `PATCH stock`),
- the monetary unit and rounding rule for stored amounts,
- where the tax rate and shipping fee live,
- and what a CommandService method receives, since this call sets the precedent.

`referenceAmount` rounding is deliberately **out of scope** — it is an advisory, non-persistent
display figure already governed by [ADR-0099] (half-up, computed once in `money.ApplyRateHalfUp`).
This ADR covers only the authoritative, stored money.

## Decision

1. **Row locking — `SELECT ... FOR UPDATE`, ordered by `id`.** Stock rows are locked pessimistically
   in ascending `product_id` order (`lock_products_for_update.sql`), fixing lock order to structurally
   avoid deadlocks between concurrent multi-product purchases. Duplicate `product_id` in a request is
   rejected as `ErrValidation` (422) before locking so the ordering premise holds. The decrement
   `UPDATE` additionally carries a defensive `WHERE quantity >= :qty`; a 0-row result is treated as
   oversell → `ErrConflict` (409). #592 uses the same lock-then-verify-then-update shape on the same rows.

2. **Money is integer USD cents, everywhere.** `products.price`, `purchase_details.unit_price`, and
   `purchases.{subtotal,tax,shipping_fee,total}_amount` are all `INTEGER` USD cents; no `float` touches
   the stored money path. `unit_price` is a snapshot of the locked `products.price` taken at purchase
   time and is immutable to later price changes (the essence of the CommandService positive example).

   > **Revised in part by [ADR-0101](0101-two-scale-money-model.md):** the *pricing-scale* columns
   > (`products.price`, `purchase_details.unit_price`) became exact decimal (USD major unit, sub-cent
   > capable). The *settlement-scale* amounts (`purchases.{subtotal,tax,shipping_fee,total}`) keep the
   > integer-USD-cents contract decided here. This ADR's stock-locking, truncation-at-the-settlement-
   > boundary (point 3), and CommandService (point 6) decisions are unchanged.

3. **Domain money rounding is truncation, in one place.** `subtotal = Σ unit_price × quantity`,
   `tax = subtotal × taxRate` (truncated), `shipping_fee = constant`, `total = subtotal + tax + shipping`.
   Rounding happens once inside the domain money computation and is **truncation** for stored amounts.
   This is distinct from `referenceAmount`, which is half-up per [ADR-0099]; the two differ because they
   serve different purposes (authoritative/charged vs. advisory/display), and both boundaries are explicit.

4. **Tax rate and shipping fee are domain constants (placeholders).** `taxRatePercent = 10` and
   `shippingFeeCents = 500` live in `internal/domain/purchase/constant.go` as sample placeholders. A fixed
   non-zero shipping fee (not 0) is chosen so the `shipping_fee` column, the `total` computation path, and
   the response field are actually exercised. When a real requirement appears they move to config / master.

5. **Initial status is resolved by code, not a baked UUID.** The domain carries
   `StatusCodeUnprocessed = 1`; the `purchase_statuses` UUID is resolved in SQL via a sub-SELECT on `code`
   at insert time, keeping the seed UUID out of application code (no two-place drift).

6. **A CommandService method receives the decided aggregate.** As the write-side counterpart of the
   Repository ([ADR-0027]), `CreatePurchase(ctx, *purchase.Purchase)` takes the decided domain aggregate —
   symmetric to how a Repository returns one — rather than a decomposed parameter bag. Infra legitimately
   handles domain entities (repositories already map rows↔entities); the DTO-boundary rule targets
   controller exposure, not infra. `LockProducts(ctx, ids) ([]LockedProduct, error)` stays a separate first
   phase because locking precedes aggregate construction. This shape is the precedent for future CommandServices.

## Consequences

### Positive Consequences

- Oversell cannot occur: it is caught both by the in-domain stock check (under the lock) and by the
  defensive `WHERE quantity >= :qty` (fail-closed), and mapped to a single 409.
- No `float` on the stored-money path; amounts are deterministic and the rounding rule is singular.
- Purchase and replenishment (#592) contend for the same rows under one consistent locking discipline.
- The write-side interface mirrors the read-side Repository, so CommandService placement and signatures
  are predictable for the rest of the purchase domain.

### Negative Consequences

- Pessimistic row locks serialize concurrent purchases of the same product; acceptable because stock is a
  real contended resource, but it bounds write throughput per hot product.
- `taxRatePercent` / `shippingFeeCents` as constants are not production tax logic; they are placeholders and
  must be revisited (config / master) before real commerce use.

## Alternatives Considered

### Advisory locks (`pg_advisory_xact_lock`)

Rejected: advisory locks fit logical keys with no backing row. Stock is a real row resource, so a row lock
is the direct expression. The outbox relay's SKIP LOCKED choice ([ADR-0047]) is a different contention
profile (compete-to-claim queue rows) and does not transfer to contended stock rows.

### Decompose the aggregate into a parameter bag for CreatePurchase

Rejected: it scatters the write intent and breaks the Repository-symmetry framing of [ADR-0027]. Passing the
aggregate keeps the "decided write unit" intact and is not a boundary violation (infra handles entities).

## Notes

- Related: [ADR-0027] (CommandService placement), [ADR-0029] (atomicity criterion; purchase is its positive
  example), [ADR-0099] (referenceAmount half-up), [ADR-0031] (UUIDv7 for `id` / `code`), [ADR-0039]
  (`ErrConflict` → 409 / `ErrValidation` → 422).
- Implementation: `internal/domain/purchase`, `internal/usecase/purchase`,
  `internal/infrastructure/rdb/command_service/purchase`, `database/dml/command_service/purchase`.

[#571]: https://github.com/Tomy-ch/go-boilerplate/issues/571
[ADR-0027]: 0027-lightweight-cqrs.md
[ADR-0029]: 0029-commandservice-atomicity-criterion.md
[ADR-0031]: 0031-uuidv7-identifiers.md
[ADR-0039]: 0039-apperror-protocol-agnostic-errors.md
[ADR-0047]: 0047-skip-locked-outbox-relay.md
[ADR-0099]: 0099-reference-amount-half-up-rounding.md
