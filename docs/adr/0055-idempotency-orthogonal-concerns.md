---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [idempotency, exclusion, setup-review]
---

# ADR-0055: Keep idempotency orthogonal to optimistic locking and rate limiting

## Status

accepted

## Context

Idempotency (deduplicating client retries via `Idempotency-Key`), optimistic locking
(detecting lost updates via version/ETag checks), and rate limiting (bounding request
frequency at the edge) address three distinct failure modes. Because all three concern
request safety and are applied in middleware or usecase layers, there is pressure to
integrate them — for example, automatically enabling optimistic locking when idempotency
is active, or co-locating rate-limiting counters with idempotency key storage.

Coupling these concerns would reduce integration surface for the simple case, but would
also make it harder to enable each independently, complicate the mental model, and
introduce policy decisions about how conflicts between the mechanisms should be resolved
(e.g., should a rate-limited request consume an idempotency key?).

## Decision

We deliberately do NOT couple idempotency with optimistic locking or rate limiting. The
idempotency subsystem is **opt-in per handler** and **orthogonal** to both:

- **Optimistic locking** (lost-update prevention via version/ETag checks) is a separate
  concern applied at the domain or usecase layer; enabling idempotency on an endpoint does
  not automatically add optimistic locking, and vice versa.
- **Rate limiting** is an edge concern; it is applied independently of whether an endpoint
  uses the `Idempotency-Key` mechanism.

Each concern may be adopted independently. There are no shared configuration flags,
combined middleware, or cross-cutting state between them.

## Consequences

### Positive Consequences

- Each mechanism can be adopted, reasoned about, and tested in isolation.
- Integrators understand the scope of each opt-in: adding idempotency does not implicitly
  add other behaviors.
- There is no policy ambiguity about interaction semantics (e.g., rate-limited retries,
  version conflicts on replayed operations).

### Negative Consequences

- An endpoint that needs all three must wire each independently, with no combined shortcut.
- The orthogonality is a design convention, not enforced by the type system or linter; it
  relies on integrators understanding the separation.

## Alternatives Considered

### Bundle optimistic locking into the idempotency middleware

Automatically perform an ETag/version check when an idempotency key is present. Rejected
because optimistic locking is a domain-level concern (it requires knowledge of the
resource's current version), while idempotency is a cross-cutting infrastructure concern.
Coupling them would violate the onion architecture's layer separation.

### Co-locate rate-limiting counters with idempotency state

Store request counts in the `idempotency_keys` table to share a single DB round-trip.
Rejected because rate limiting is an edge concern (typically enforced before the request
reaches application code) and operates on different time windows and eviction semantics
than idempotency TTLs.

## Notes

- Source: [`docs/design/idempotency.md`](../design/idempotency.md) §1 ("orthogonal to optimistic
  locking (lost-update prevention) and rate limiting (edge concern)").
- This is a **setup-review ADR**: when adopting this scaffold, confirm that the boundary
  between idempotency, optimistic locking, and rate limiting is appropriate for the target
  system. If the target requires combined enforcement, each mechanism must be extended
  explicitly rather than relying on the scaffold's separated defaults.
- Related: [ADR-0002](0002-onion-architecture.md) (layer separation — optimistic locking
  belongs in the domain/usecase layer, rate limiting at the edge).
