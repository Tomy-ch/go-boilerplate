---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [outbox, async, gc]
---

# ADR-0046: 7-day retention GC of published rows (batches of 10,000)

## Status

accepted

## Context

Once an outbox row is marked `published`, it serves no further delivery purpose. Without
a cleanup policy the `outbox` table grows without bound, increasing storage costs and
gradually slowing the `ClaimPending` query as the table size grows — even with a partial
index on `status = 'pending'`, table bloat degrades vacuuming and page cache efficiency.

## Decision

`GCUsecase.SweepPublished` deletes `published` rows whose `published_at` timestamp is
older than `DefaultRetention = 7 days`, processing at most `DefaultGCBatchSize = 10 000`
rows per invocation. The underlying SQL selects candidates ordered by `published_at` and
deletes by `id IN (subquery)`, bounding the lock duration per statement. GC is invoked
via `cmd job outbox-gc` — a one-shot command scheduled by an external cron (see
[ADR-0048](0048-relay-resident-gc-oneshot.md)).

The SQL is:

```sql
DELETE FROM outbox
WHERE id IN (
    SELECT o.id
    FROM outbox AS o
    WHERE o.status = 'published'
      AND o.published_at < $1
    ORDER BY o.published_at
    LIMIT $2
);
```

## Consequences

### Positive Consequences

- The `outbox` table is bounded; it does not grow monotonically.
- Batch-delete limits per-statement lock duration and I/O pressure, reducing impact on
  concurrent relay activity.
- The 7-day retention window preserves recent delivery history for debugging without
  accumulating data indefinitely.

### Negative Consequences

- Published rows occupy space for up to 7 days after delivery; this is acceptable but
  means the table always contains some historical rows.
- Batch size and retention period are fixed defaults; changing them requires a code change.
- An external scheduler (cron or Kubernetes CronJob) must be provisioned and monitored;
  a misconfigured or absent cron leaves published rows accumulating.

## Alternatives Considered

### Delete inside MarkPublished

Immediate cleanup, but adds latency to every mark call and runs inside the relay
transaction, lengthening row-lock hold time.

### Continuous background sweeper embedded in the relay

Avoids an external scheduler dependency, but couples two unrelated concerns in one
process and adds concurrency to an already stateful loop.

### No GC (retain forever)

Operationally simple but unsustainable at any non-trivial event volume.

## Notes

- `DefaultRetention` (7 days) and `DefaultGCBatchSize` (10 000) described in
  `docs/design/outbox.md` (Glossary entry "GC (SweepPublished)").
- SQL source: `database/dml/system_query/outbox/delete_published_outbox.sql`.
- Related ADRs: [ADR-0045](0045-outbox-dead-after-max-attempts.md),
  [ADR-0048](0048-relay-resident-gc-oneshot.md).
