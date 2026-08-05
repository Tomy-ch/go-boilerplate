---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [worker, async, adapter]
---

# ADR-0045: Broker-agnostic pull-ack worker scaffold

## Status

accepted

## Context

Queue messages need to reach the same Usecase layer as HTTP requests, as **another driving
adapter** (message-in) on par with the HTTP handler, without inventing a new architectural
layer and without binding the engine to any specific broker.

## Decision

Provide a **broker-agnostic, pull-ack worker scaffold**:

- The engine (`internal/controller/worker`) depends only on a minimal seam
  (`Consumer` / `Handler` / `FailureHandler`) defined in `internal/usecase/boundary/worker`,
  and is **completed against an in-memory fake** — every engine invariant is tested without a
  real broker.
- The seam is **scoped to the pull-ack class**, designed first against AWS SQS and GCP
  Pub/Sub (pull). Other pull-ack platforms fit by writing an adapter; only fundamentally
  different models require changing the interface.
- Permanent failures route through a `FailureHandler` (dead-letter) seam; broker-specific
  redrive (e.g. SQS `maxReceiveCount` → DLQ) is IaC configuration, not application code.
- Backpressure is a 3-state **circuit breaker on the intake side** (stop pulling on
  continued downstream failure, self-heal via half-open), distinct from per-message `Nack`
  delay and from `Fatal` (which stops the engine).

## Consequences

### Positive Consequences

- The same Usecase / domain code is reachable from HTTP and from queues without duplication.
- Broker independence: switching or adding a pull-ack broker is an adapter change; the
  engine and its tests do not change.
- A fake-first engine keeps the behavioral contract (ack discipline, ordering, drain,
  backpressure) verifiable in fast unit tests.

### Negative Consequences

- The port is deliberately narrow (pull-ack only); push and streaming-log models do not fit
  and are excluded (recorded separately as ADR-0046, materialized in Phase 4).
- Each new broker requires an adapter implementing the seam.

## Alternatives Considered

### A general multi-broker abstraction (incl. push / streaming)

Rejected: a lowest-common-denominator port would leak or weaken guarantees; Kafka-style
consumers (offset commit / consumer-group / partition) belong to a separate engine, not an
extension of this port.

### Build tags for dependency isolation of adapters

Rejected: no precedent in this repo, and with a single binary, module separation is
insufficient. Not importing an adapter from `cmd` achieves isolation without tags (see
[ADR-0047](0047-sqs-adapter-opt-in.md)).

## Notes

- Push / streaming exclusion: ADR-0046 (Phase 4). SQS adapter opt-in isolation: [ADR-0047](0047-sqs-adapter-opt-in.md).
- Design reference: `docs/design/worker.md`.
- Migrated from `docs/decisions.md` (§ "Why a broker-agnostic worker scaffold").
