---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [outbox, async, idempotency]
---

# ADR-0052: Propagate the outbox message_id as the receiver's Idempotency-Key

## Status

accepted

## Context

The relay delivers messages with at-least-once semantics
(see [ADR-0050](0050-at-least-once-outbox-poll.md)), meaning a receiver may observe the
same message on more than one HTTP call — due to transient delivery failures, process
restarts, or edge-case duplicate claims. To allow the receiver to de-duplicate without
application-level coordination, every delivery must carry a stable, globally unique key
that identifies the logical event, not the transport attempt.

## Decision

Each outbox row is assigned a `message_id` UUID **once, at INSERT time** (assigned by
`EmitUsecase.Emit`). The HTTP publisher propagates this UUID as the
`Idempotency-Key` request header on every POST to the receiver endpoint. The receiver
is expected to treat this key as the deduplication token and return `2xx` only after the
event is durably accepted.

## Consequences

### Positive Consequences

- The key is stable across retries: the same UUID is sent on every delivery attempt for
  a given outbox row, regardless of how many poll cycles have elapsed.
- Receiver-side de-duplication is straightforward: one indexed column keyed on
  `Idempotency-Key` converts at-least-once delivery into exactly-once effect.
- The key is assigned synchronously at emit time, so it is available for tracing and
  debugging before the row is ever delivered.

### Negative Consequences

- The receiver must implement de-duplication; the subsystem provides the key but not the
  receiver-side storage or logic.
- The `message_id` must never change after INSERT: any mutation that regenerates the UUID
  (e.g. a replay that assigns a new UUID) would break receiver de-duplication.

## Alternatives Considered

### Receiver-side de-duplication by payload hash

Brittle if the payload contains timestamps or other non-deterministic fields; does not
survive payload evolution across schema versions.

### No de-duplication mechanism

Pushes the full burden onto the receiver without a stable, subsystem-provided reference
key. Impractical for at-least-once delivery.

## Notes

- Receiver idempotency design invariant: `docs/design/outbox.md` (§ "Design invariants",
  integrator checklist ③).
- `message_id` propagation implemented in
  `internal/infrastructure/publisher/http_publisher.go` (`Publish`,
  `httpclient.WithIdempotencyKey`).
- Related ADRs: [ADR-0049](0049-transactional-outbox.md), [ADR-0050](0050-at-least-once-outbox-poll.md).
