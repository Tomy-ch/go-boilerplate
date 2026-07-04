---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [idempotency, gc, ops]
---

# ADR-0053: Run idempotency key garbage collection as a separate one-shot CLI job

## Status

accepted

## Context

Idempotency keys expire after 24 hours but are not deleted automatically at that point —
they remain in the `idempotency_keys` table until something removes them. Without active
cleanup, the table grows without bound. There are two common approaches for managing this
growth: inline cleanup on the request path (delete a batch of expired rows before or after
each write) or a dedicated background job that runs on a schedule.

Inline cleanup couples table maintenance to request latency, adds unpredictable spikes to
write-path latency, and distributes the deletion load across all incoming requests —
including periods of low traffic when the backlog grows. It also ties cleanup throughput to
request throughput.

A separate GC job decouples cleanup entirely from the request path. It can be scheduled at
a fixed cadence, sized independently (batch size is configurable), and monitored or
restarted without touching the API serving path.

The scaffold's CLI job mechanism (`internal/controller/job/`) provides a natural home for
one-shot batch operations, and the idempotency subsystem already ships a `GCUsecase` with
a `SweepExpired` method designed for batch iteration.

## Decision

Idempotency key garbage collection is implemented as a **separate one-shot CLI job**
(`idempotencygc`). The job calls `GCUsecase.SweepExpired(batchSize)` in a loop, each
iteration deleting up to `batchSize` expired rows ordered by `expires_at`, until a short
batch signals completion. The default batch size is 10,000. The job is scheduled externally
(cron, k8s CronJob) and does not run on the request path.

## Consequences

### Positive Consequences

- The request path has no GC overhead; write latency is not affected by expired-row volume.
- Batch size is independently tunable via `--batch-size=N` without any API change.
- The job can be monitored, retried, and throttled independently of the API server.
- Hourly scheduling is sufficient for a 24-hour TTL — rows linger at most 25 hours in
  the worst case, which is acceptable.

### Negative Consequences

- Expired rows are not removed immediately; the table retains stale rows for up to the GC
  job interval (typically one hour).
- An external scheduler (cron or k8s CronJob) is required; the API server alone is not
  self-cleaning.
- If the GC job is not scheduled or fails repeatedly, the table grows unbounded.

## Alternatives Considered

### Inline cleanup on the request path

Delete a batch of expired rows as part of each write request. Rejected because it adds
unpredictable latency spikes to the write path and ties cleanup throughput to request
throughput.

### Background goroutine inside the API server

Run a goroutine in the server process that periodically sweeps expired rows. Rejected
because it couples the server's resource usage to GC activity, makes the GC harder to
observe and restart independently, and introduces goroutine lifecycle management concerns.

## Notes

- Source: [`docs/design/idempotency.md`](../design/idempotency.md) §1 (responsibility table,
  "GCUsecase + idempotencygc job") and §4 (operational notes, "Schedule the GC").
- `GCUsecase` is at `internal/usecase/idempotency/gc.go`; the job entry point is at
  `internal/controller/job/idempotencygc/`.
- The `expires_at` index in `idempotency_keys` keeps each `DeleteExpired` batch scan cheap
  regardless of total table size.
- Related: [ADR-0051](0051-idempotency-fixed-ttl.md) (fixed 24h TTL that the GC sweeps
  against).
