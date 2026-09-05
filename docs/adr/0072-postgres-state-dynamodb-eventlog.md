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

Visibility has a second trap, symmetric to the first. The cursor a feature hands out — a History
`streamCursor` — is the stream's committed position in PostgreSQL, while the store that validates it
is filled by the relay asynchronously. Committed is not the same as appended. A validation that
reads "not here" as "gone" refuses a cursor the relay has simply not reached, and sends the client to
a recovery path that hands back the same cursor: a loop with no exit. The same loop opens from the
other side once retention has removed the last event of an idle stream — the client that saw that
event has lost nothing, yet the item at its cursor is absent.

## Decision

**PostgreSQL remains the sole source of truth for current domain state.** A DynamoDB table — the
**EventLog** — holds delivery events for a bounded period (7 days) purely so that a stream can be
replayed and resumed. Nothing is rebuilt from it. It is not an audit log.

- Partition key = `streamId`, sort key = `sequence`. Replay reads use `ConsistentRead=true`.
- History and any other read a feature offers are PostgreSQL projections, never EventLog scans.

### The ordering chain is one invariant

Correctness is not "sequences are assigned correctly". It is that the chain **feature commit order →
outbox → EventLog visibility → client cursor** is never broken. Three rules make it one invariant:

1. **Sequences have no gaps.** The feature's adapter allocates the stream-local sequence inside
   the feature's own business transaction through the mechanism's **sequence allocator**: one row
   per stream in a `system_cqrs` table owned by Realtime Delivery ([ADR-0033]), updated with
   `UPDATE … RETURNING` and held locked until commit. The row is mechanism state, like an outbox
   row — no aggregate carries a sequence field, and no Repository allocates one. Allocation order
   therefore equals commit order, and a rolled-back transaction rolls its increment back with it.
   Writes to one stream serialize on that row — the same single-stream ceiling the DynamoDB
   partition imposes on the read side.
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

### What the invariant makes unnecessary, and the one thing it does not

With no gaps and a contiguous prefix, the store needs no replay *floor* — no per-stream `floor` /
`version` item and no job to advance one. Everything about what is still replayable is derivable
from the log itself.

What is **not** derivable from the log is where the log *ends*. Retention deletes items without
trace, so an absent item at position `n` reads the same whether `n` was appended and has since
expired or the relay has not written `n` yet — and a cursor may legitimately name the latter. The
EventLog therefore keeps one piece of metadata per stream: the **append watermark**, the highest
sequence ever appended, advanced by the append itself, never rolled back, and exempt from the TTL.
It is a fact about the log, recorded by the log — not a copy of the outbox's status — and it is the
shape every retained log exposes (a partition's end offset, a stream's last generated id) so that
"before the beginning" and "past the end" stay distinguishable after trimming. It clamps nothing and
orders nothing; the claim predicate still does both.

Cursor validity is then one strongly consistent read after the cursor and, when that read is empty,
one read of the watermark:

- the first item after `cursor` is `cursor + 1` and within retention ⇒ replayable;
- it is later than `cursor + 1`, or older than retention ⇒ expired;
- nothing after `cursor` and `cursor >= watermark` ⇒ replayable. Nothing the client has not seen was
  ever appended: either it is caught up, or the relay has not reached `cursor` yet and the
  connection waits for it;
- nothing after `cursor` and `cursor < watermark` ⇒ expired. Events after the cursor were appended
  and have since aged out — the case a stream that went idle and expired entirely would otherwise
  hide as "caught up";
- DynamoDB's asynchronous TTL deletion is never the authority for "expired", and an unreadable
  EventLog is a retryable server error, never a guess about the cursor.

Whether the item *at* the cursor still exists plays no part, and the initial position is not a
special case: it is the cursor `0` against a watermark that is `0` until the first append.

## Consequences

### Positive Consequences

- A reconnect storm is absorbed by a store built for key-range reads; the PostgreSQL pool that
  serves REST never sees it.
- Every existing aggregate keeps its shape. Adding realtime delivery to a feature is an adapter
  and an outbox row, not a rewrite.
- Resume is exact: `Last-Event-ID` or `after` names a point in a gap-free, contiguous sequence,
  and the client cannot be handed a later event while an earlier one is still in flight.
- The only replay metadata is one item per stream in the same table, advanced in-line by the append
  that makes it true — no separate table, no job, and no second record of anything the outbox
  already records.

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

### A contiguous watermark instead of head-of-line blocking

A per-stream "highest contiguous sequence appended" kept in the EventLog and used to **clamp what
clients may see**, so that later events could be appended while an earlier one is stuck. Rejected as
an ordering mechanism: an appended-yet-invisible event has no value, and a visibility clamp is a
second state that can disagree with the outbox's own status column. The append watermark this
decision keeps is not that — it clamps nothing, the claim predicate still holds the order, and it
only records where the log ends so that an absent item can be classified after retention has removed
it.

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
no information in it. The end of the log is the one value the log cannot derive once retention has
run, and it is advanced by the append itself, not by a job.

### Deriving the History cursor from the EventLog

Have History report the relay's position instead of the committed one, so that a cursor is
replayable by construction. Rejected. It puts a feature's canonical read path on the delivery
buffer's availability, contradicting "History is a PostgreSQL projection"; and the rows and the
cursor then disagree — either History withholds committed rows, so the caller's own write is missing
from its next read, or the stream re-delivers what History already returned.

### Accepting any cursor past the end

Admit a cursor with nothing after it and let the continuity check at delivery time raise `RESYNC`.
Rejected. That check fires only when the next event arrives; on a stream that went idle and expired
entirely, a client holding an older cursor is told nothing and believes it is caught up. The
retention-window resynchronization obligation would exist only on paper.

### Reading the relay's position from the outbox

Derive "relayed through" from the outbox — the earliest unpublished sequence on the key, else the
allocator's current position — at connect time. Rejected. It puts PostgreSQL on the reconnect path
in exactly the branch a mass reconnect to idle streams takes, and it couples cursor validation to
the claim predicate's head-of-line semantics: the two things the rest of this decision keeps apart.

## Notes

- Design reference: `docs/design/realtime-delivery.md` §2 (the ordering chain as a state machine)
  and §5 (`stream`, `sequence`, `cursor`, `replay floor`).
- Related: [ADR-0071] (the mechanism), [ADR-0054] (events are emitted in the business
  transaction), [ADR-0056] (the claim predicate this decision extends), [ADR-0058] (what makes an
  outbox row dead — head-of-line blocking is why a dead head halts a stream), [ADR-0037]
  (UUIDv7 event identifiers), [ADR-0033] (the `system_cqrs` category the sequence table belongs
  to — mechanism state beside the outbox, not a feature's aggregate).
- The sequence allocation and the claim predicate land in different phases of the parent issue
  (the feature adapter and the outbox routing respectively); each phase's tests pin its half of the
  chain.

[ADR-0033]: 0033-system-cqrs-dml-category.md
[ADR-0037]: 0037-uuidv7-identifiers.md
[ADR-0054]: 0054-transactional-outbox.md
[ADR-0056]: 0056-skip-locked-outbox-relay.md
[ADR-0058]: 0058-outbox-dead-on-permanent-error.md
[ADR-0071]: 0071-realtime-delivery-driving-mechanism.md
