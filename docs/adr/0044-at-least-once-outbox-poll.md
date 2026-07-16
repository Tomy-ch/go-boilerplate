---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [outbox, async]
---

# ADR-0044: At-least-once delivery via polling (transport-level retry disabled)

## Status

accepted

## Context

Outbox rows must reach the external endpoint despite transient network failures or receiver
downtime. Two retry mechanisms are available independently:

1. **Transport-level retry** — retry the HTTP call within the same poll cycle (e.g. three
   attempts before giving up).
2. **Poll-loop retry** — leave the row `pending` and let the next poll cycle pick it up.

Enabling both simultaneously causes double-retry amplification: a row that exhausts its
transport retries still stays `pending` and receives the full retry budget again on every
subsequent poll. This can inflate the `attempts` counter far faster than intended and
obscure whether a problem is transient or permanent. The design doc calls this constraint D10.

## Decision

Use **poll-loop retry** as the sole at-least-once mechanism. When `Publish` fails, the
relay does **not** roll back the transaction; the row stays `pending`, `attempts` is
incremented, and `last_error` is updated. The next poll cycle reclaims and retries it.
Transport-level retry is disabled: `MaxAttempts = 1` in `NewDownstreamProfile` (D10).

## Consequences

### Positive Consequences

- Retry behaviour is deterministic: exactly one transport attempt per poll cycle.
- A single delivery failure does not stall the relay loop or block other rows.
- `attempts` and `last_error` provide direct, unambiguous observability of what failed
  and how many times.
- Eventual dead-lettering (see [ADR-0047](0047-outbox-dead-after-max-attempts.md)) is
  predictable because `attempts` advances at most once per poll.

### Negative Consequences

- Delivery latency after a failure is bounded by `PollInterval` (default 1 s), not by an
  immediate transport retry.
- At-least-once semantics mean the receiver must be idempotent on the `Idempotency-Key`
  header (see [ADR-0046](0046-message-id-idempotency-propagation.md)).

## Alternatives Considered

### Transport-level retry enabled

Simpler per-attempt logic, but causes double-retry amplification: every poll cycle
multiplies the retry budget by the transport attempt count. Also obscures the distinction
between transient and permanent failures.

### Exponential backoff per row

More sophisticated and avoids hot-polling a failing row. Requires per-row timer state and
complicates the `FOR UPDATE SKIP LOCKED` claim model (a back-off row must be skipped, not
claimed). Deferred in favour of the simpler poll-loop approach.

## Notes

- At-least-once and D10 are described in `docs/design/outbox.md` (§ "Design invariants",
  Glossary entry "retry-by-poll").
- Transport retry is disabled in `internal/infrastructure/publisher/http_publisher.go`
  (`NewDownstreamProfile`, `MaxAttempts = 1`).
- Related ADRs: [ADR-0043](0043-transactional-outbox.md), [ADR-0045](0045-skip-locked-outbox-relay.md),
  [ADR-0047](0047-outbox-dead-after-max-attempts.md).
