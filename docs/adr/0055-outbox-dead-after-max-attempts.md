---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [outbox, async]
---

# ADR-0055: MaxAttempts = 10, then the message is dead (terminal until manual replay)

## Status

accepted

## Context

Some outbox rows will never succeed regardless of how many retries are attempted — for
example, a permanently misconfigured endpoint, a payload the receiver rejects as invalid,
or a receiver that has been decommissioned. Without a retry cap, these rows stay `pending`
forever, consuming relay capacity on every poll and polluting lag metrics with noise that
obscures genuine delivery problems.

## Decision

`DefaultMaxAttempts = 10`. Each `MarkFailed` call increments `attempts` and sets
`last_error`. When `attempts >= MaxAttempts`, the relay calls `MarkDead`, transitioning
the row to the `dead` status. Dead rows are **terminal**: the relay makes no further
delivery attempt, the `outbox.dead` metric counter is incremented, and a warning is
logged. Recovery is manual: an operator invokes
`outbox-relay replay [--message-id=<uuid>]`, which resets `attempts = 0` and
`last_error = NULL`, returning the row to `pending`.

The three permitted status values are `pending`, `published`, and `dead` (CHECK-constrained
in the schema). There is no `failed` status; a failed publish stays `pending` until it
either succeeds or exhausts `MaxAttempts`.

## Consequences

### Positive Consequences

- The relay loop is protected from spending capacity on permanently undeliverable rows.
- Dead rows are surfaced by the `outbox.dead` metric, making them observable and alertable.
- Manual replay allows selective recovery (single message or full batch) without
  reprocessing healthy rows.
- The `last_error` field on each dead row records the final failure reason for diagnosis.

### Negative Consequences

- A message that fails transiently more than 10 consecutive times is quarantined until an
  operator manually replays it; no automatic recovery occurs.
- `MaxAttempts = 10` is a fixed default; there is no per-event-type override.

## Alternatives Considered

### Unlimited retries

Protects against data loss but lets undeliverable rows accumulate indefinitely, consuming
poll capacity and masking real lag.

### Delete on exhaustion

Simplest bounded behaviour, but data is permanently lost; no recovery path exists.

### Automatic exponential backoff with dead-letter only after a timeout

More sophisticated retry envelope, but requires per-row timer state and complicates the
`FOR UPDATE SKIP LOCKED` claim model — a back-off row must be skipped without consuming
its lock. Deferred in favour of the simpler attempt-count approach.

## Notes

- `DefaultMaxAttempts`, the `dead` state, and manual replay are described in
  `docs/design/outbox.md` (§ "State transitions", Glossary entries "MaxAttempts" and "dead").
- Related ADRs: [ADR-0052](0052-at-least-once-outbox-poll.md),
  [ADR-0056](0056-outbox-retention-gc.md).
