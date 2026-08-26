---
status: accepted
date: 2026-07-10
deciders: [maintainers]
tags: [exclusion, outbox, messaging, reliability, setup-review]
---

# ADR-0107: Ship a balanced outbox relay; delegate hardening to operational evidence

## Status

accepted

## Context

The outbox relay claims pending rows, publishes them over HTTP, and marks the outcome inside a
single transaction ([ADR-0055], [ADR-0056]). Any such design is bounded by an impossibility
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
- **Tx-retry republish**: a serialization-failure/deadlock retry ([ADR-0035]) re-runs the whole
  claim → publish → mark function, re-sending already-delivered messages within one poll. The
  relay is the sole sanctioned exception to ADR-0035's rule that external side effects must live
  in outbox rows — the relay *is* the drain and has no outbox to defer to.
- **Attempts as a lower bound**: a batch rollback erases the `attempts` increment but not the
  HTTP POST that already happened.
- **Lag SLI ambiguity**: rows being published still count as `pending`, so the lag gauge cannot
  distinguish in-flight from stalled.

A hardened design that closes every closable window exists (the blueprint below). Whether to run it
is not a correctness question — it is a trade-off between three things:

- **Duplicate suppression** — how far duplicates are pushed from "absorbed harmlessly downstream"
  toward "structurally impossible".
- **Availability** — a lease- and leader-based relay stalls for the takeover interval when the
  holder dies, where a stateless poll loop simply continues on whichever instance polls next.
- **Implementation and operational cost** — extra schema (two columns and a status value), a
  dependency on runtime topology for leader election, and tuning knobs (lease length, deadline
  margin, backoff curve) that only a specific deployment can set.

Which of the three binds is rarely knowable before the system runs. Message volume, instance count,
how long a receiver outage actually lasts, and whether receivers deduplicate are operational facts,
not design-time ones. Fixing a point on this trade-off in advance therefore risks paying
availability and cost for a duplicate rate that never materialises — the same reasoning as
[ADR-0104] and [ADR-0105].

## Decision

We ship the **balanced** point on that trade-off and move from it on operational evidence.

The relay stays the simple single-transaction one, and duplicates are handled by absorption rather
than by structural exclusion: `message_id` propagates as `Idempotency-Key` ([ADR-0057]) and the
bundled idempotency middleware deduplicates on the receiving side, so duplicates collapse to
exactly-once *effect* wherever that middleware runs (third-party receivers inherit the dedup
obligation as an integration requirement). This point costs no availability and no extra schema,
never loses a message, and leaves duplicates possible but harmless.

Once operation shows the duplicate axis binding — a multi-instance relay, volume that makes
normal-operation duplicates routine, or a receiver that cannot deduplicate — the relay SHOULD be
redesigned with the following multi-layer structure. It has no functional trade-off; its price is
implementation cost plus an availability window.

### Hardening blueprint

1. **Lease-based claim**: add `claiming` status, `claimed_until`, and `next_attempt_at`; fold
   expired-lease reclaim into the claim predicate (`pending AND next_attempt_at <= now()` OR
   `claiming AND claimed_until < now()`) so no separate reaper process exists.
2. **Publish outside the transaction**: claim and mark become short txs; tx dwell, vacuum pinning,
   idle-in-transaction concerns, and the tx-retry republish path disappear. With no external side
   effect inside any transaction, the relay satisfies ADR-0035's general rule and its sanctioned
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

### Keeping the later move cheap

Deferring a decision is only safe while taking it later stays cheap. Four things keep it so:

- **Blueprint, not just a verdict**: this ADR carries the full mechanical design — schema, claim
  predicate, fences, and the residual-window ledger — so the later move pays for implementation and
  tuning, never for re-deriving the analysis.
- **Extension seams, not a rewrite**: the hardened design fits the existing seams, so it arrives as
  an extension of specific interfaces and tables rather than a rewrite:
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
  the one-line DI provide swap. The schema deliberately does NOT pre-admit `claiming` in its CHECK,
  and no unused strategy switch is shipped — schema and wiring declare only what the code
  exercises.
- **Escalation trigger**: if hardened relays become the norm rather than the exception, promote
  this blueprint to an implementation guide under `docs/design/` (or a reference implementation)
  and revisit this ADR — the balanced point is where a relay starts, not where it must stay.

## Consequences

### Positive Consequences

- The shipped relay stays small and readable, and states the at-least-once contract honestly
  instead of implying exactly-once.
- Hardening starts from a fully-analyzed, mechanically-applicable blueprint (schema, predicates,
  fences, residual-window ledger) rather than from a half-hardened default whose policy values were
  fixed before any operational evidence existed.
- Responsibility is explicit: this decision owns the absorption contract and this analysis; the
  deployment that hardens owns the implementation and its operational tuning.

### Negative Consequences

- The shipped relay retains the enumerated windows: under multi-instance operation duplicates
  can occur in normal operation, and a short downstream outage dead-letters the backlog. The
  shipped defaults therefore suit low-volume or single-instance relays.
- Hardening is real implementation work (migration, claim/mark rewrite, lock, fence, tests);
  divergence from the shipped default grows once it is done — mitigated, not eliminated, by
  the blueprint and the extension seams above, which bound the divergence to an enumerated surface.
- Third-party receivers remain protected only by the documented dedup obligation — nothing in this
  repository can enforce it.

## Alternatives Considered

### Implement the multi-layer redesign up front

Closes every closable window out of the box, but fixes a point on the trade-off before operation
can say which axis binds: it commits deployment-specific policy (lease, margin, backoff, singleton
topology) to values chosen without that evidence, and pays availability and reading cost on the
core path from day one. Rejected as the shipped default; recorded here so the work is mechanical
once the evidence arrives.

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
- Related ADRs: [ADR-0035] (tx-retry idempotency contract; the shipped relay is its sole
  sanctioned exception, which blueprint layer 2 removes), [ADR-0055] (at-least-once poll),
  [ADR-0056] (SKIP LOCKED claim), [ADR-0057] (message-id / Idempotency-Key propagation),
  [ADR-0058] (dead-lettering after max attempts).
- Full ADR set and ordering: [the ADR log](README.md).

[ADR-0035]: 0035-transaction-retry-idempotent-callers.md
[ADR-0055]: 0055-at-least-once-outbox-poll.md
[ADR-0056]: 0056-skip-locked-outbox-relay.md
[ADR-0057]: 0057-message-id-idempotency-propagation.md
[ADR-0058]: 0058-outbox-dead-after-max-attempts.md
[ADR-0104]: 0104-no-in-app-rate-limiter.md
[ADR-0105]: 0105-scheduled-job-concurrency-delegated.md
