---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [idempotency, reliability]
---

# ADR-0060: Run claim, business function, and complete in a single transaction for at-most-once semantics

## Status

accepted

## Context

A database transaction guarantees atomicity within one request, but it does not deduplicate
client retries. When a write has no natural unique key — a `POST` that allocates its own id,
a balance increment, a charge, an email send — a network timeout or double-submit causes the
side effect to run again.

The idempotency subsystem prevents this by tracking each `(scope, key)` pair in a
`idempotency_keys` table. The central design question is whether the claim (reserving the
key), the business function, and the completion step (storing the response) run in the same
database transaction or in separate ones.

Using separate transactions would open a window between a successful business operation and
a failed `Complete` write: the effect would be applied but the response never stored, so a
retry would attempt the business function a second time. Conversely, a claim in a separate
transaction that is never followed by a `Complete` would require a separate explicit
"release" step on business failure, adding complexity and a failure mode.

## Decision

The `Run[T]` orchestrator executes `Claim`, `businessFn`, and `Complete` inside **one
shared database transaction**. If the business function fails or `Complete` fails, the
transaction rolls back and the claim is released — the key becomes free for a clean retry.
A committed transaction guarantees that exactly one response is stored and every retry of a
completed key replays that stored response without calling the business function again.

This is the at-most-once guarantee: the side effect runs at most once per `(scope, key)`.

## Consequences

### Positive Consequences

- Business failure automatically releases the key via rollback — no explicit release path
  is needed.
- A stored response is always paired with a committed business effect; there is no
  intermediate "effect applied, response lost" state.
- The orchestrator (`Run`) is infrastructure-agnostic: it delegates the transaction to
  `tx.Manager` and persistence to the `Store` seam.

### Negative Consequences

- The database transaction spans the full duration of the business function, which may hold
  locks longer than a write-only transaction would.
- All three steps must participate in the same `tx.Manager`-controlled transaction; a
  business function that opens its own separate transaction cannot participate in this
  guarantee.

## Alternatives Considered

### Two-phase: separate claim then complete

Claim in one transaction, run the business function, complete in a second transaction.
Rejected because a crash between the two phases leaves the key claimed but incomplete, and
a retry would skip the business function (key exists) while no response is stored — making
the operation silently lost.

### Idempotency table in a separate database

Store idempotency state in a dedicated database decoupled from the business database.
Rejected because atomic coordination between two databases requires distributed transactions
or a two-phase commit protocol, both of which add substantial complexity without benefit for
this use case.

## Notes

- Source: [`docs/design/idempotency.md`](../design/idempotency.md) §1 (design principles) and §2.2
  (per-request decision diagram).
- Related: [ADR-0002](0002-onion-architecture.md) (onion architecture — `Run` depends only
  on `Store` seam and `tx.Manager`, not on infrastructure).
- The `Store` seam is defined at `internal/usecase/boundary/idempotency`; the RDB
  implementation lives in `internal/infrastructure/rdb/system_cqrs/idempotency`.
