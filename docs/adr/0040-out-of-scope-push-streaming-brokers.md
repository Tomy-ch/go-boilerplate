---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [worker, async, exclusion, setup-review]
---

# ADR-0040: Push-type brokers and streaming-log platforms are out of scope for the worker port

## Status

accepted

## Context

The broker-agnostic worker scaffold ([ADR-0039](0039-broker-agnostic-worker-scaffold.md))
defines a pull-ack port (`Consumer` / `Handler` / `FailureHandler`) that allows different
queue brokers to be wired as adapters without changing the engine. This design creates
pressure to also cover:

- **Push-type brokers** (e.g. RabbitMQ, AMQP) — the broker pushes messages to the consumer
  rather than the consumer polling.
- **Streaming-log platforms** (e.g. Apache Kafka, Amazon Kinesis) — consumption involves
  offset management, consumer groups, and partition assignment; the protocol is fundamentally
  different from pull-ack.

There is a temptation to treat these as "just another adapter" and extend the pull-ack seam
to cover them, or to generalise the port to a lowest-common-denominator interface.

## Decision

We deliberately do NOT support push-type brokers or streaming-log consumers within the
worker port defined by [ADR-0039](0039-broker-agnostic-worker-scaffold.md). The exclusion
is not about pull-ack being the only conceivable model — it is about deliberately avoiding
the need to build and maintain multiple adapter variants. Pull-ack is the chosen concrete
instance; push and streaming are excluded because supporting them would require either a
diluted lowest-common-denominator interface or separate adapters with their own maintenance
burden.

- **Push delivery** (RabbitMQ-style) already has a natural home in this codebase: the HTTP
  controller layer receives pushed requests. A webhook endpoint is the correct adapter for
  push-type brokers.
- **Streaming-log consumers** (Kafka / Kinesis) require offset commit, consumer-group
  coordination, and per-partition state — a fundamentally different engine, not an extension
  of the pull-ack port.

Generalising the port to accommodate both models would produce a lowest-common-denominator
interface that leaks protocol details or weakens the Ack / Nack / Extend guarantees that
the engine relies on.

## Consequences

### Positive Consequences

- The pull-ack seam stays narrow and coherent; Ack / Nack / Extend semantics are
  unambiguous for all conforming adapters.
- Engine invariants (ordering, drain, circuit breaker) remain testable against a single
  in-memory fake.
- Teams that genuinely need Kafka or Kinesis are not misled into forcing those platforms
  through a mismatched adapter.

### Negative Consequences

- Push-type and streaming workloads require a separate subsystem (outside this scaffold)
  when they cannot be served by HTTP webhooks.
- The constraint must be communicated to integrators at setup time to prevent incorrect
  adapter attempts.

## Alternatives Considered

### A general multi-broker abstraction (incl. push / streaming)

Rejected: a lowest-common-denominator port would leak or weaken guarantees. Kafka-style
consumers (offset commit / consumer-group / partition) belong to a separate engine, not an
extension of this port.

### A separate streaming-log port alongside the pull-ack port

Not rejected in principle, but out of scope for this scaffold — no concrete demand, and
adding it speculatively violates [ADR-0001](0001-avoid-lock-in.md)'s preference for
concrete over hypothetical abstraction.

## Notes

- Related: [ADR-0039](0039-broker-agnostic-worker-scaffold.md) (pull-ack worker scaffold,
  the port this ADR qualifies).
- Source: `docs/decisions.md` (§ "Why a broker-agnostic worker scaffold", Decision bullet 3
  and Alternatives Considered).
- Source: `docs/design/worker.md` (§ 1 Role theory).
