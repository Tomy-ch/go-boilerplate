---
status: accepted
date: 2026-08-28
deciders: [maintainers]
tags: [outbox, async, reliability]
---

# ADR-0058: An outbox row dies on a permanent error; transient failures retry with per-message backoff

## Status

accepted

## Context

Some outbox rows will never succeed regardless of how many retries are attempted — a
permanently misconfigured endpoint, a payload the receiver rejects as invalid, a receiver that
has been decommissioned. Without a terminal state these rows stay `pending` forever, consuming
relay capacity on every poll and polluting lag metrics with noise that obscures genuine delivery
problems.

The first version of this decision made the terminal state a function of *count*: ten failed
attempts, then `dead`. That rule was the safety valve for a relay that had no per-message
backoff — a failed row was re-claimed on the very next poll, so *something* had to stop it — and
it never distinguished a receiver that rejects a message from a receiver that is briefly down.
[ADR-0111] records the consequence: a downstream outage of a few tens of seconds drives an entire
pending backlog to `dead`, and recovery is an operator replaying every row by hand.

The realtime delivery channel ([ADR-0072]) turns that consequence from a nuisance into a
correctness problem. Its ordering rule stops a stream at the first sequence that is not yet
appended; if a transient outage can dead-letter that head row, a substrate blip of a minute halts
every active stream until an operator intervenes. The channel also has no per-message permanent
failure at all — a payload is validated before its row is written, and the substrate either
accepts an append or is unreachable for everyone — so a count-based rule would only ever fire on
transient failures there.

## Decision

**A row becomes `dead` because its failure is permanent, not because it has failed often.**

- The publisher returns errors classified with the `apperror` sentinels ([ADR-0047],
  [ADR-0048]): `ErrPermanent` for a failure that a retry cannot change, `ErrRetryable` for one
  it might. The HTTP publisher derives the class from the outbound client's existing verdicts
  ([ADR-0024]): a non-retryable response (4xx other than 429) is permanent; 5xx, 429, and
  transport failures are retryable. The realtime publisher reports an unreachable substrate as
  retryable; its payload validation happens before emit, so it has no permanent class.
- **Permanent → `MarkDead` immediately.** `last_error` records the reason, `outbox.dead` is
  incremented, a warning is logged. Dead rows are terminal until an operator runs
  `outbox-relay replay [--message-id=<uuid>]`, exactly as before.
- **Retryable → the row stays `pending` and its `next_attempt_at` is advanced** by an
  exponential backoff with full jitter, capped at 60 seconds. The claim predicate adds
  `next_attempt_at <= now()`, so a backed-off row is simply not selected — it is never locked,
  which is why this coexists with `FOR UPDATE SKIP LOCKED` ([ADR-0056]) without the complication
  the first version of this decision feared. `attempts` still increments for diagnosis; it is no
  longer a criterion for anything.
- **An error that carries neither sentinel is treated as retryable.** This is the same default
  the worker engine applies to an unclassified error, and it errs toward not losing a message.
  What it gives up is the old count-based backstop against a permanent failure nobody
  classified. That is compensated by observation rather than by a counter: the age of the oldest
  `pending` row per delivery channel is a lag SLI, and a row that stays pending beyond a threshold
  is alerted on — surfaced for a human to classify, not dead-lettered by a number.

The three status values remain `pending`, `published`, and `dead`; there is no `failed` and no
`backing-off` status — backoff is a timestamp on a pending row.

Implementation lands with the outbox delivery-channel work of the parent issue (the same change
that introduces `next_attempt_at`); `docs/design/outbox.md` is updated in that change.

## Consequences

### Positive Consequences

- A downstream outage costs latency, not a dead backlog: rows wait, retry with widening
  intervals, and drain when the receiver returns. The relay is no longer the component that
  turns a transient failure into an incident.
- A permanent failure is dead on its first occurrence, with the reason recorded, instead of
  after ten identical attempts that add nothing.
- The realtime channel's head-of-line rule is safe to hold: a stream stalls only on a permanent
  failure, which that channel does not produce, or on an outage, which resolves itself.
- The classification the relay uses is the one the rest of the codebase already speaks; no new
  error taxonomy is introduced.

### Negative Consequences

- A permanent failure that nobody classified — a receiver returning `500` forever, a bug that
  surfaces as a transport error — retries indefinitely at the backoff cap. Nothing terminates it
  automatically; the lag SLI and the alert on it are the only guard, and they must be watched.
  On the realtime channel this is the only way a stream head can stall, because that channel has
  no permanent class at all: such a row never becomes `dead`, so `outbox-relay replay` does not
  apply to it, and recovery is fixing the misclassified cause and deploying — deliberately, since
  an operator command that forced the row to `dead` would be a way to skip a sequence, which
  [ADR-0072] forbids.
- The `next_attempt_at` column and the claim-predicate change are a schema migration and an
  index change, not a configuration flip.
- Backoff parameters (initial interval, cap, jitter) are fixed values in code. A deployment that
  wants different ones changes the code.

## Alternatives Considered

### Dead after a fixed attempt count (`MaxAttempts = 10`) — the previous decision

Replaced. It was the safety valve for a relay without per-message backoff, and it dead-letters
by frequency of failure rather than by kind: a receiver down for thirty seconds and a receiver
that will never accept the message look identical to it. Under the realtime channel's ordering
rule it would halt every active stream on a short outage.

### Unlimited retries with no classification and no backoff

Rejected, as before: undeliverable rows accumulate, consume poll capacity on every cycle, and
mask real lag. Classification is what makes "no attempt cap" safe — permanent failures leave the
loop on their first appearance.

### Delete on exhaustion

Rejected, as before: data is lost with no recovery path.

### Treat an unclassified error as permanent

Rejected. It dead-letters every failure the publisher did not foresee, most of which are
transient — the exact behaviour this decision removes. The retryable default plus an age alert
loses nothing and surfaces the rare permanent case for a human.

### Keep a very large attempt cap as a backstop

Rejected. A cap of a thousand is still a count masquerading as a classification, with no
principled value, and it re-creates the outage-to-dead path at a different scale.

## Notes

- Design reference: `docs/design/outbox.md` — its state diagram and glossary still describe the
  previous count-based rule and are rewritten together with the implementation (see Decision);
  until then this ADR is the only statement of the classification rule.
- This decision adopts item 3 of the hardening blueprint in [ADR-0111] (per-message backoff via
  `next_attempt_at`) on its own — not on operational evidence, but because the realtime channel
  cannot tolerate count-based dead-lettering. The other blueprint items stay deferred as that ADR
  states.
- Related: [ADR-0024] (the outbound client's retryable / non-retryable verdicts the HTTP
  publisher maps from), [ADR-0047] / [ADR-0048] (the sentinels), [ADR-0055] (retry by poll),
  [ADR-0056] (the claim predicate), [ADR-0059] (retention GC), [ADR-0072] (why a dead head row
  halts a stream).

[ADR-0024]: 0024-outbound-http-resilience.md
[ADR-0047]: 0047-apperror-protocol-agnostic-errors.md
[ADR-0048]: 0048-error-metadata-code-message-details.md
[ADR-0055]: 0055-at-least-once-outbox-poll.md
[ADR-0056]: 0056-skip-locked-outbox-relay.md
[ADR-0059]: 0059-outbox-retention-gc.md
[ADR-0072]: 0072-postgres-state-dynamodb-eventlog.md
[ADR-0111]: 0111-outbox-relay-hardening-delegated.md
