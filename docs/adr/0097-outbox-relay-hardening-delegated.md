---
status: accepted
date: 2026-07-10
deciders: [maintainers]
tags: [exclusion, outbox, messaging, reliability, setup-review]
---

# ADR-0097: Delegate outbox-relay duplicate-window hardening to production copies

## Status

accepted

## Context

The outbox relay claims pending rows, publishes them over HTTP, and marks the outcome inside a
single transaction ([ADR-0046], [ADR-0047]). Any such design is bounded by an impossibility
result: an external side effect (the HTTP POST) and its record (the DB row) cannot be made atomic
(Two Generals). Duplicate, loss, and availability *windows* — bounded time intervals during which
an invariant can be violated — cannot all be zero; a design only chooses which window remains.
A window admits exactly three treatments: **close** (make the interleaving structurally
impossible), **narrow** (shrink the interval), or **absorb** (make the event harmless downstream).

The shipped single-transaction relay carries these windows:

- **Transaction dwell**: publish runs inside the claim tx, so the worst-case tx duration is
  `BatchSize (100) x per-attempt timeout (~3s) ≈ 300s`, pinning the vacuum horizon and holding a
  pool connection. Consequently a pool-wide `idle_in_transaction_session_timeout` backstop is
  deliberately omitted (`driver.applyDBTimeouts` sets only `statement_timeout` / `lock_timeout`):
  a blanket value short enough to backstop runaway transactions would kill the relay's own
  long-lived claim→publish→mark tx. It becomes safe to enable pool-wide once blueprint layer 2
  moves publish outside the transaction.
- **No per-message backoff**: max attempts is hard-coded (`DefaultMaxAttempts = 10`) and failed
  rows are re-claimed on the next poll (default 1s), so a downstream outage of only tens of
  seconds drives the whole pending backlog to `dead`.
- **Tx-retry republish**: a serialization-failure/deadlock retry ([ADR-0030]) re-runs the whole
  claim → publish → mark function, re-sending already-delivered messages within one poll. The
  relay is the sole sanctioned exception to ADR-0030's rule that external side effects must live
  in outbox rows — the relay *is* the drain and has no outbox to defer to.
- **Attempts as a lower bound**: a batch rollback erases the `attempts` increment but not the
  HTTP POST that already happened.
- **Lag SLI ambiguity**: rows being published still count as `pending`, so the lag gauge cannot
  distinguish in-flight from stalled.

A hardened design that closes every closable window exists (the blueprint below). The pressure to
ship it in the template is real. However, it adds schema (two columns and a status value), a
leader-election dependency on runtime topology, and tuning knobs (lease length, deadline margin,
backoff curve) whose correct values are deployment-specific — policy the template cannot know,
mirroring the reasoning of [ADR-0094] and [ADR-0095].

## Decision

We deliberately do NOT harden the outbox relay in this template. The template keeps the simple
single-transaction relay together with its shipped absorption contract: `message_id` propagates as
`Idempotency-Key` ([ADR-0048]) and the bundled idempotency middleware deduplicates on the receiving
side, so duplicates collapse to exactly-once *effect* whenever the receiver is built from this
template (third-party receivers inherit the dedup obligation as an integration requirement).

Production copies that operate the outbox at scale SHOULD redesign the relay with the following
multi-layer structure. It has no functional trade-off; its price is implementation cost plus an
availability window.

### Hardening blueprint

1. **Lease-based claim**: add `claiming` status, `claimed_until`, and `next_attempt_at`; fold
   expired-lease reclaim into the claim predicate (`pending AND next_attempt_at <= now()` OR
   `claiming AND claimed_until < now()`) so no separate reaper process exists.
2. **Publish outside the transaction**: claim and mark become short txs; tx dwell, vacuum pinning,
   idle-in-transaction concerns, and the tx-retry republish path disappear. With no external side
   effect inside any transaction, the relay satisfies ADR-0030's general rule and its sanctioned
   exception ceases to exist.
3. **Per-message exponential backoff** via `next_attempt_at`, with max attempts made configurable —
   a downstream outage then costs minutes-to-hours before dead-lettering instead of seconds.
4. **Singleton topology** via a session-scoped Postgres advisory lock — closes inter-instance
   concurrent publish at the topology level; the lock auto-releases when the holder's session dies.
5. **Self-deadline fence**: bound each batch's publishing by `claimed_until − margin`, anchored to
   the DB clock captured at claim time — a live instance structurally cannot act outside its own
   lease (no clock-skew dependence between app instances).
6. **Fenced mark**: `WHERE status = 'claiming' AND claimed_until = <own lease>` — double-mark is
   closed entirely inside the DB.
7. **Absorption as the last layer**: the only remaining duplicates are crash-class, temporal
   (never concurrent), bounded in count, and keyed by the same `message_id` — exactly the input the
   bundled idempotency middleware collapses deterministically.

Under this structure, normal-operation duplicates are structurally zero; the lease length tunes
only crash-recovery latency (no longer a duplicate-probability trade-off); and the residual price
is an availability window (failover takeover, hung-leader detection), which belongs to deployment
runbooks.

### Balanced delegation

The delegation splits the cost of hardening between the template and its copies:

- **Blueprint, not just a verdict**: this ADR carries the full mechanical design — schema, claim
  predicate, fences, and the residual-window ledger — so a copy pays for implementation and
  tuning, never for re-deriving the analysis.
- **Extension seams, not a rewrite**: the hardened design fits the template's existing seams, so a
  copy builds it as an extension of specific interfaces and tables rather than a fork-and-rewrite:
  - *Strategy seam*: the relay engine owns only the poll loop and its waits, and reaches the
    claim → publish → mark business solely through the `RelayUsecase` interface; the hardened
    orchestration is a new implementation of that interface, selected in DI.
  - *Persistence seam*: `boundary/outbox.Store` is **signature-compatible** with the lease design —
    `ClaimPending(ctx, limit)` is implementable by the lease-claim `UPDATE … RETURNING`, and the
    mark methods keep their shapes. The contracts are stated behaviorally (concurrent callers never
    claim the same row), never mechanically (no `FOR UPDATE` wording), so a lease implementation
    *conforms* instead of deviating.
  - *Query seam*: one query per SQL file — new predicates arrive as new files under
    `database/dml/system_cqrs/outbox/`.
  - *Schema seam*: new columns (`next_attempt_at`, `claimed_until`) arrive as additive migrations.
- **What is not additive, declared honestly**: swapping the `outbox_status_check` CHECK to admit
  `claiming` and replacing `outbox_pending_idx` for the new claim predicate (both via new
  migration files — the standard schema-evolution mechanism; a CHECK cannot be "extended"), plus
  the one-line DI provide swap. The template deliberately does NOT pre-admit `claiming` in its
  CHECK nor ship an unused strategy switch — its schema and wiring declare only what its own code
  exercises.
- **Escalation trigger**: if hardened copies become the norm rather than the exception, promote
  this blueprint to an implementation guide under `docs/design/` (or a reference implementation)
  and revisit this ADR — the exclusion is a default for the scaffold's audience, not a ceiling.

## Consequences

### Positive Consequences

- The template's relay stays small and readable, and teaches the at-least-once contract honestly
  instead of implying exactly-once.
- Copies get a fully-analyzed, mechanically-applicable blueprint (schema, predicates, fences,
  residual-window ledger) rather than a half-hardened default with template-chosen policy values.
- Responsibility is explicit: the template owns the absorption contract and this analysis; the
  copy owns hardening and its operational tuning.

### Negative Consequences

- The template's relay retains the enumerated windows: under multi-instance operation duplicates
  can occur in normal operation, and a short downstream outage dead-letters the backlog. The
  shipped defaults therefore suit low-volume or single-instance relays.
- Copies must do real implementation work (migration, claim/mark rewrite, lock, fence, tests) to
  productionize; divergence from the template grows once they do — mitigated, not eliminated, by
  the blueprint and the extension seams above, which bound the divergence to an enumerated surface.
- Third-party receivers remain protected only by the documented dedup obligation — nothing in this
  repository can enforce it.

## Alternatives Considered

### Implement the multi-layer redesign in the template

Closes every closable window out of the box, but bakes deployment-specific policy (lease, margin,
backoff, singleton topology) into a scaffold that cannot know the right values, and grows the
teaching surface of the core sample. Rejected for the template; recorded here so the copy-side
work is mechanical.

### Narrow-only tuning (smaller batches, shorter timeouts, lease values)

Keeps every window open and only reduces width/frequency — probabilistic, not structural,
protection. Rejected as a primary strategy; narrowing is useful only as a complement to closing.

### At-most-once relay (mark published before sending; never retry ambiguous outcomes)

Closes all duplicate windows by opening a loss window. Rejected outright: guaranteed delivery is
the outbox's reason to exist.

### Receiver-verified fencing tokens

Closes end-to-end duplication but requires receiver cooperation beyond the shipped contract, i.e.
the same third-party dependence as absorption with strictly more coupling. Rejected as a sole
mechanism.

## Notes

- Design reference: [`docs/design/outbox.md`](../design/outbox.md) — §1 invariants and the §4
  integrator checklist carry the receiver-dedup obligation this decision relies on.
- Related ADRs: [ADR-0030] (tx-retry idempotency contract; the shipped relay is its sole
  sanctioned exception, which blueprint layer 2 removes), [ADR-0046] (at-least-once poll),
  [ADR-0047] (SKIP LOCKED claim), [ADR-0048] (message-id / Idempotency-Key propagation),
  [ADR-0049] (dead-lettering after max attempts).
- Full ADR set and ordering: [the ADR log](README.md).

[ADR-0030]: 0030-transaction-retry-idempotent-callers.md
[ADR-0046]: 0046-at-least-once-outbox-poll.md
[ADR-0047]: 0047-skip-locked-outbox-relay.md
[ADR-0048]: 0048-message-id-idempotency-propagation.md
[ADR-0049]: 0049-outbox-dead-after-max-attempts.md
[ADR-0094]: 0094-no-in-app-rate-limiter.md
[ADR-0095]: 0095-scheduled-job-concurrency-delegated.md
