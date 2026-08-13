---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [outbox, async, ops]
---

# ADR-0059: The relay is a resident process; GC is a one-shot cron job

## Status

accepted

## Context

The relay and the GC operation have fundamentally different runtime characteristics:

- The **relay** must minimise delivery lag. It must be continuously responsive to new
  `pending` rows, sleeping only when there is nothing to claim, and resuming immediately
  when new rows appear. A restart-on-each-run model introduces latency spikes proportional
  to the cron interval.

- **GC** is periodic maintenance — it sweeps `published` rows older than the retention
  window. It has no latency requirement and needs to run only often enough to prevent
  unbounded table growth. Running it continuously wastes resources and complicates the
  relay process unnecessarily.

Both operations share the same binary (`cmd outbox-relay` for the relay; `cmd job
outbox-gc` via the main binary's `job` subcommand).

## Decision

- **Relay** — runs as a **resident process** (`cmd outbox-relay`). It starts, enters the
  poll loop, and stays up until it receives `SIGTERM`. On `SIGTERM` it drains in-flight
  batches gracefully before exiting. Lifecycle is managed by `SupervisedRunner` wired in
  `OutboxRelayModule`.

- **GC** — runs as a **one-shot job** (`cmd job outbox-gc`). It executes one sweep of
  `published` rows past the retention window and then exits. Scheduling cadence is
  delegated to an external scheduler (Kubernetes CronJob or system cron).

## Consequences

### Positive Consequences

- The relay process is simple: a single poll loop with a well-defined shutdown path.
- GC scheduling cadence is controlled by the external scheduler without code changes.
- Failure isolation: a GC crash or misconfiguration does not affect the relay, and a relay
  restart does not disrupt a running GC job.
- Resource consumption is separated: the relay runs continuously at low CPU; GC runs
  briefly and then exits.

### Negative Consequences

- An external scheduler must be provisioned and monitored independently. A missing or
  misconfigured cron leaves `published` rows accumulating until the scheduler is corrected.
- Two operational concerns — relay uptime and GC schedule — must be managed separately,
  adding to deployment complexity.

## Alternatives Considered

### GC embedded in the relay process on a timer

Reduces deployment units. But couples two unrelated concerns in one process: a GC bug or
panic could affect relay availability, and GC downtime equals relay downtime.

### Relay as a cron job (run-to-drain then exit)

Simpler deployment: no long-lived process to monitor. However, this introduces a latency
spike on every cron interval; continuous delivery with sub-second `PollInterval` is not
achievable without a resident process.

## Notes

- Responsibility split: `docs/design/outbox.md` (§ "Role theory", table rows for
  `outbox-gc job` and relay `Engine`).
- Integrator checklist ⑤–⑥: `docs/design/outbox.md` §4.
- Related ADRs: [ADR-0052](0052-transactional-outbox.md),
  [ADR-0057](0057-outbox-retention-gc.md).
