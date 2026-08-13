---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [outbox, async, reliability]
---

# ADR-0051: Transactional outbox: emit events within the business transaction

## Status

accepted

## Context

Services frequently need to mutate domain state and notify an external endpoint as a single
logical operation. When the domain write (DB) and the publish (HTTP POST) are two separate
network calls, either can fail independently: publishing before the DB commit produces a
phantom event if the transaction rolls back; publishing after creates a lost event if the
process crashes or the network call fails. This dual-write anomaly is a fundamental
reliability risk in any service that fans out domain changes to downstream consumers.

## Decision

Adopt the transactional outbox pattern. `EmitUsecase.Emit` inserts a single row into the
`outbox` table **inside the same `tx.Manager.Do` as the domain change**. The "intent to
publish" is persisted as part of the business transaction, so a rolled-back business
transaction atomically discards its outbox row — no phantom events and no lost intent.
Actual delivery is deferred to a separate, asynchronous relay process.

## Consequences

### Positive Consequences

- The dual-write anomaly is eliminated: the DB is the single write point per business
  operation.
- A rolled-back business transaction cannot produce a phantom outbox row.
- The domain usecase remains decoupled from delivery details — transport, retry, and
  ordering belong to the relay.
- Schema-level atomicity is provided by the database; no distributed transaction or
  two-phase commit is required.

### Negative Consequences

- Delivery is asynchronous; downstream consumers observe eventual consistency.
- A dedicated relay process must be deployed and kept running.
- An `outbox` table and its associated lifecycle (migrations, GC) must be maintained.

## Alternatives Considered

### Synchronous publish inside the business tx

Publish the event within the same transaction and roll back on delivery failure. This
tightly couples the business operation to transport availability: a temporarily-down
receiver fails every business request that emits an event.

### Change Data Capture (CDC)

Capture events from the database WAL instead of inserting an explicit outbox row.
CDC is more transparent at the application level but requires additional infrastructure
(e.g. Debezium), complex operational setup, and is harder to reason about in application
code.

## Notes

- Dual-write avoidance is a design invariant; see `docs/design/outbox.md` (§ "Design invariants").
- Migrated from `docs/design/outbox.md` §1 (Role theory) and the invariants table.
- Related ADRs: [ADR-0002](0002-onion-architecture.md) (onion architecture, layer ownership).
- Related ADRs: [ADR-0052](0052-at-least-once-outbox-poll.md) (delivery guarantee), [ADR-0053](0053-skip-locked-outbox-relay.md) (relay concurrency).
