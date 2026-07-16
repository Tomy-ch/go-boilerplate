---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [outbox, async, concurrency]
---

# ADR-0046: Single-transaction relay using SELECT FOR UPDATE SKIP LOCKED (safe across instances)

## Status

accepted

## Context

Multiple relay instances may run simultaneously for availability or throughput scaling.
Without row-level coordination, two instances could claim the same `pending` row and each
deliver it to the receiver endpoint, violating the goal of minimising duplicates. The
locking strategy must ensure that only one instance processes each row at a time, while
not blocking other instances from making progress on different rows.

## Decision

`ClaimPending` issues `SELECT ... FOR UPDATE SKIP LOCKED` against the `outbox` table.
The entire **claim → publish → mark** sequence runs inside a single transaction. A claimed
row's lock is held until the surrounding transaction commits or rolls back, so a second
relay instance that runs `ClaimPending` concurrently will skip the locked row rather than
blocking on it.

The SQL is:

```sql
SELECT id, message_id, aggregate_type, aggregate_id,
       event_type, payload, headers, attempts
FROM outbox
WHERE status = 'pending'
ORDER BY id
LIMIT $1
FOR UPDATE SKIP LOCKED;
```

## Consequences

### Positive Consequences

- Multi-instance safety is enforced at the database level with no application-side
  coordination or distributed lock manager.
- `SKIP LOCKED` avoids head-of-line blocking: if one instance holds a row, other instances
  skip it and continue processing other rows.
- Rows are processed in `id` order within each batch (best-effort; strict global ordering
  is sacrificed for lock availability).

### Negative Consequences

- Requires a database that supports `FOR UPDATE SKIP LOCKED` (PostgreSQL 9.5+). Portability
  to other engines is not guaranteed.
- A long relay transaction holding many row locks may increase contention; batch size must
  be tuned to limit lock hold time.

## Alternatives Considered

### Optimistic compare-and-set on status

`UPDATE outbox SET status = 'claimed' WHERE status = 'pending'` before processing. Avoids
locking but risks races at high contention without careful isolation levels, and introduces
a separate `claimed` status that must be handled in failure/crash scenarios.

### Application-level advisory locks

PostgreSQL advisory locks managed by the application. More flexible but complex to
implement correctly and requires application-side coordination for which rows are locked.

### Exclusive relay (single instance)

Simpler: no concurrency issue. But introduces a single point of failure for the entire
delivery path.

## Notes

- Multi-instance safety invariant: `docs/design/outbox.md` (§ "Design invariants").
- SQL source: `database/dml/system_cqrs/outbox/claim_pending_outbox.sql`.
- Related ADRs: [ADR-0044](0044-transactional-outbox.md), [ADR-0045](0045-at-least-once-outbox-poll.md).
