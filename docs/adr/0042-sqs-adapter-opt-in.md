---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [worker, async, dependencies]
---

# ADR-0042: SQS adapter is opt-in and not linked into the default binary

## Status

accepted

## Context

The worker scaffold ([ADR-0040](0040-broker-agnostic-worker-scaffold.md)) ships a reference
broker adapter for AWS SQS. If that adapter were wired into the default build, every binary —
including `serve`, which never consumes a queue — would link `aws-sdk-go-v2`, enlarging the
dependency surface against the lock-in-avoidance principle
([ADR-0001](0001-avoid-lock-in.md)).

## Decision

Keep the SQS adapter **opt-in**. The reference adapter lives in
`internal/infrastructure/queue/sqs` and is **not wired into the default `cmd` build**, so
`aws-sdk-go-v2` is not linked into the shipped binary. A deployment that wants SQS wires the
adapter explicitly.

## Consequences

### Positive Consequences

- Default binaries stay free of the AWS SDK — smaller dependency surface and build.
- The reference adapter still exists as a worked example for any pull-ack broker.
- Dependency isolation is achieved by *not importing* from `cmd`, without build tags.

### Negative Consequences

- Enabling SQS requires an explicit wiring step; it is not on by default.

## Alternatives Considered

### Wire SQS by default

Rejected: it would force `aws-sdk-go-v2` into every binary (including `serve`), against the
replaceable-infrastructure goal.

### Build tags for dependency isolation

Rejected: no precedent in this repo, and a single binary makes module separation
insufficient. Not importing the adapter from `cmd` isolates the dependency without tags.

## Notes

- Parent decision: [ADR-0040](0040-broker-agnostic-worker-scaffold.md). Principle: [ADR-0001](0001-avoid-lock-in.md).
- Reference: `internal/infrastructure/queue/sqs/README.md`.
- Migrated from `docs/decisions.md` (§ "Why a broker-agnostic worker scaffold" — dependency isolation).
