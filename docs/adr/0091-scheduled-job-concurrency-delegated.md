---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [exclusion, setup-review]
---

# ADR-0091: Do not control scheduled-job concurrency in-app; delegate to the scheduler

## Status

accepted

## Context

Scheduled jobs running in a multi-instance environment can overlap: a new run starts before
the previous one finishes, or two instances start the same job at the same wall-clock tick.
The conventional remedy is an application-level mutual exclusion mechanism — typically an
advisory database lock (e.g. `pg_try_advisory_lock`) or a distributed lock (e.g. Redis
`SET NX`). These mechanisms add infrastructure dependencies and coordination logic that must
be tested, operated, and kept in sync with the job lifecycle.

The template bundles four scheduled jobs: `outbox-gc`, `idempotency-gc`, `usercount`, and
the outbox relay. Each is designed to be concurrency-safe without application-level locking:

- `outbox-gc` and `idempotency-gc` are age-predicate, idempotent batch deletes — running
  them concurrently produces the same result as running them once.
- `usercount` is read-only.
- The outbox relay claims rows with `SELECT … FOR UPDATE SKIP LOCKED`, so concurrent
  relays process disjoint sets of rows.

Given this design, application-level mutual exclusion would add complexity without providing
correctness benefits for the bundled jobs.

## Decision

We deliberately do NOT control scheduled-job concurrency inside the application. Overlap
prevention and multi-instance guarding are delegated to the scheduler (e.g. Kubernetes
`CronJob` with `concurrencyPolicy: Forbid` or `Replace`, or an equivalent advisory-lock
mechanism at the scheduler layer).

Adopters who require strict single-run semantics must configure `concurrencyPolicy: Forbid`
at the Kubernetes `CronJob` (or the equivalent for their scheduler). For jobs added by
adopters that are not concurrency-safe by design, the same scheduler-level policy applies;
if the adopter's job requires fine-grained locking beyond scheduler control, they should
implement it within that specific job.

## Consequences

### Positive Consequences

- No advisory-lock infrastructure dependency in the application.
- Bundled jobs remain concurrency-safe by design, not by lock — simpler and easier to test.
- Concurrency policy is expressed declaratively in scheduler configuration, making it
  visible to operators without reading application code.

### Negative Consequences

- Adopters are responsible for setting `concurrencyPolicy` correctly; the application
  provides no guard if they omit it.
- Adopters who add non-idempotent jobs must implement their own locking or rely on the
  scheduler policy — neither is provided out of the box.
- The concurrency guarantee is only as strong as the scheduler's enforcement, which may
  vary (e.g. clock skew on multi-master Kubernetes setups).

## Alternatives Considered

### Advisory database lock per job (`pg_try_advisory_lock`)

Prevents concurrent runs at the database level, independent of the scheduler. Rejected
because it couples the job runner to the database for coordination (not data), complicates
failure recovery (lock not released on crash), and is unnecessary for the bundled jobs which
are already safe by design.

### Distributed lock via Redis

Robust across multiple database replicas, but introduces a Redis dependency solely for
coordination. Rejected as over-engineering for the template; scheduler-level policy achieves
the same outcome without an extra infrastructure component.

## Notes

- Source: [`docs/project/out-of-scope.md`](../project/out-of-scope.md) lines 17–26.
- Full ADR set and ordering: [the ADR log](README.md).
