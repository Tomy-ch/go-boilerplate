---
status: accepted
date: 2026-08-28
deciders: [maintainers]
tags: [architecture, persistence, async, realtime, reliability]
---

# ADR-0072: PostgreSQL holds current state; the DynamoDB EventLog is a bounded replay store, not an event-sourced log

## Status

accepted

## Context

A client that reconnects after a dropped SSE connection must receive every event it missed, in
order, and nothing twice. That requires a store that can answer "everything on this stream after
cursor N" cheaply, under a burst of reconnections, without touching the transactional database that
serves the ordinary REST traffic — a reconnect storm that lands on the PostgreSQL pool takes the
whole API down with it.

An event store next to the state store invites two familiar mistakes. One is to promote it to the
source of truth and rebuild state from it (event sourcing), which rewrites every existing aggregate
and doubles the number of places state lives. The other is to let it drift into an audit log, which
attaches retention, immutability, and legal requirements to what is only a delivery buffer.

Ordering has its own trap. A per-stream sequence assigned at write time is not enough: if the
publish of sequence 5 fails and is retried while sequence 6 is published and read by a client, the
client's cursor moves to 6 and sequence 5 is never delivered — the catch-up query only looks
*after* the cursor, and the mechanism deliberately tolerates no signal a client could use to notice
the hole.

## Decision

**PostgreSQL remains the sole source of truth for current domain state.** A DynamoDB table — the
**EventLog** — holds delivery events for a bounded period (7 days) purely so that a stream can be
replayed and resumed. Nothing is rebuilt from it. It is not an audit log.

- Partition key = `streamId`, sort key = `sequence`. Replay reads use `ConsistentRead=true`.
- History and any other read a feature offers are PostgreSQL projections, never EventLog scans.

### The ordering chain is one invariant

Correctness is not "sequences are assigned correctly". It is that the chain **feature commit order →
outbox → EventLog visibility → client cursor** is never broken. Three rules make it one invariant:

1. **Sequences have no gaps.** The feature allocates the stream-local sequence inside its own
   business transaction by updating the row that owns the stream (`UPDATE … RETURNING`) and holding
   that row lock until commit. Allocation order therefore equals commit order, and a rolled-back
   transaction rolls its increment back with it. Writes to one stream serialize on that row —
   the same single-stream ceiling the DynamoDB partition imposes on the read side.
2. **Client-visible sequences form a contiguous prefix.** The outbox relay claims a row only when
   no earlier sequence on the same ordering key is still unpublished — stream-local head-of-line
   blocking expressed in the claim predicate, alongside the existing `FOR UPDATE SKIP LOCKED`
   ([ADR-0056]). A row being published is still `pending`, so the predicate excludes its
   successors without any extra state.
3. **A terminal failure stops the stream; it never skips a sequence.** A stream whose head is dead
   is surfaced as a metric (`realtime_blocked_streams`) and resumes when the dead row is replayed.
   No failure marker, tombstone, or "consume the sequence and move on" exists.

The head-of-line rule is cheap to hold because the realtime channel has only three failure
classes: the substrate is unreachable (all events fail together), a conditional write collides
(an idempotent success), or a payload is invalid (rejected before the outbox row is written).
"One event permanently failing while the next succeeds" has no source, so the rule almost never
engages, and when it does the cause is systemic and the stall is the correct signal.

### What the invariant makes unnecessary

With no gaps and a contiguous prefix, the store needs no replay metadata — no per-stream
`latest` / `floor` / `version` item and no job to advance it:

- the latest sequence is the first item of a descending read;
- a cursor is expired when the item at `cursor + 1` is absent while a later one exists, or when it
  exists with an `occurredAt` older than the retention window — DynamoDB's asynchronous TTL
  deletion is never the authority for "expired";
- an unreadable EventLog is a retryable server error, never a guess about the cursor.

## Consequences

### Positive Consequences

- A reconnect storm is absorbed by a store built for key-range reads; the PostgreSQL pool that
  serves REST never sees it.
- Every existing aggregate keeps its shape. Adding realtime delivery to a feature is an adapter
  and an outbox row, not a rewrite.
- Resume is exact: `Last-Event-ID` or `after` names a point in a gap-free, contiguous sequence,
  and the client cannot be handed a later event while an earlier one is still in flight.
- The absence of replay metadata removes a table, a job, and a class of inconsistency between two
  records of the same fact.

### Negative Consequences

- Writes to one stream serialize on one PostgreSQL row. For the first consumer (one active
  conversation per user) this is invisible; a stream that is written to at high frequency will hit
  the row-lock ceiling before any other limit, and the mechanism offers no sharding for it.
- A dead head row halts its stream until an operator replays it. The halt is deliberate, and it
  depends on the metric being watched.
- Retention is a delivery property, not a business one: a client absent longer than 7 days must
  resynchronize from the feature's canonical read path (History), and the feature must offer one.
- Two stores hold copies of the same payload for up to 7 days; the EventLog therefore carries the
  same encryption-at-rest obligation as the primary database.

## Alternatives Considered

### Event sourcing — EventLog as the source of truth

Rejected. It makes two stores authoritative for the same fact, forces every aggregate to be
rebuilt from events, and turns a delivery concern into the persistence model of the whole
application.

### An event table in PostgreSQL

Rejected. It shares the pool with REST. The reconnect storm the mechanism must survive is the
exact load that would exhaust that pool.

### Contiguous watermark instead of head-of-line blocking

A per-stream "highest contiguous sequence appended" kept in the EventLog and used to clamp what
clients may see. Rejected: it lets later events be appended while earlier ones are stuck, but an
appended-yet-invisible event has no value, and the watermark is a second state that can disagree
with the outbox's own status column.

### Checking the predecessor at append time (publisher-side ordering)

Rejected. A successor that keeps failing because its predecessor is not there yet burns attempts
and, under a count-based dead rule, dead-letters itself; under any rule it repeats a DynamoDB read
per stream length for nothing. The claim predicate stops the successor from being claimed at all.

### A failure marker that consumes the sequence

Rejected. It solves a case that does not exist (a payload-specific permanent failure is rejected
before emit), and in the case that does exist (the substrate is down) the marker's own append
fails identically.

### Replay metadata with a floor advanced by a cleanup job

Rejected as redundant once sequences are gap-free: everything the metadata would record is
derivable from the log itself, and a job that scans every stream to advance a floor is a cost with
no information in it.

## Notes

- Design reference: `docs/design/realtime-delivery.md` §2 (the ordering chain as a state machine)
  and §5 (`stream`, `sequence`, `cursor`, `replay floor`).
- Related: [ADR-0071] (the mechanism), [ADR-0054] (events are emitted in the business
  transaction), [ADR-0056] (the claim predicate this decision extends), [ADR-0058] (what makes an
  outbox row dead — head-of-line blocking is why a dead head halts a stream), [ADR-0037]
  (UUIDv7 event identifiers).
- The sequence allocation and the claim predicate land in different phases of the parent issue
  (the feature adapter and the outbox routing respectively); each phase's tests pin its half of the
  chain.

[ADR-0037]: 0037-uuidv7-identifiers.md
[ADR-0054]: 0054-transactional-outbox.md
[ADR-0056]: 0056-skip-locked-outbox-relay.md
[ADR-0058]: 0058-outbox-dead-after-max-attempts.md
[ADR-0071]: 0071-realtime-delivery-driving-mechanism.md
