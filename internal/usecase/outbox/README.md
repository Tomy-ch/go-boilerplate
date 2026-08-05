# outbox

English | [日本語](README.ja.md)

Usecases for the transactional outbox: **emit** (record an outbox entry in the
same transaction as the domain change), **relay** (claim pending entries and
publish them), **GC** (prune old published entries), and **replay** (return dead
entries to pending). All persistence goes through the `Store` boundary
(`internal/usecase/boundary/outbox`); the concrete RDB implementation lives in
`internal/infrastructure/rdb/system_cqrs/outbox/`.

## Why an outbox?

Publishing an event to an external broker is not part of the database
transaction. If the domain change commits but the publish fails (or vice versa),
the two diverge — a **lost event** or a **phantom event**. The outbox closes
that gap: the event is recorded inside the *same* transaction as the domain
change, and a separate relay process publishes it afterwards
(at-least-once). Consumers must therefore be idempotent — the `MessageID` is the
stable dedup key propagated into the `Idempotency-Key`.

## Entry lifecycle

```mermaid
stateDiagram-v2
    [*] --> pending: Emit (in business tx)
    pending --> published: relay ClaimPending (FOR UPDATE SKIP LOCKED), publish ok
    pending --> pending: publish fail (attempts++), retried next poll
    pending --> dead: publish fail, attempts ≥ maxAttempts
    published --> [*]: SweepPublished (GC), entry deleted
    dead --> pending: ReplayDead
```

`published` entries are pruned by **GC** (`SweepPublished`); `dead` entries are
recovered by **replay** (`ReplayDead`). These are distinct paths — GC never
touches `dead`, and replay never touches `published`.

## Usecases

### emit — `EmitUsecase`

`NewEmit(store, tracerFactory) EmitUsecase`

- `Emit(ctx, EmitInput) (uuid.UUID, error)` records exactly one outbox entry and
  returns the allocated `message_id`. It must be called **inside the same
  `tx.Manager.Do` as the domain change** so a rolled-back business transaction
  also rolls back the outbox entry (no lost / phantom event).
- The current trace context is captured into the entry's headers as `traceparent`
  (via `observability.InjectTraceContextToCarrier`), so the eventual
  relay → consumer stays on the same trace.
- `EmitInput` fields: `AggregateType`, `AggregateID` (observation only),
  `EventType` (type + version), `Payload` (caller-marshaled event body JSON),
  and `Headers` (propagated to the external endpoint). Do **not** put sensitive
  headers (`Authorization` / `Cookie`) in `Headers` — they are sent verbatim
  to the external endpoint.
- **Where to build `Payload` — keep it out of the usecase body.** The caller owns
  the marshaling, but the versioned event contract (its struct, JSON field names,
  and `EventType` constant) must live in a **dedicated event unit** (its own package
  / function) that the usecase merely calls. Inlining the wire representation into
  the usecase method re-introduces a fat usecase and couples the orchestration to a
  serialization format; isolating it keeps the usecase a thin orchestrator and gives
  the event contract a single, testable home.

### relay — `RelayUsecase`

`NewRelay(txm, store, publisher, metrics, clock, logger, tracerFactory) RelayUsecase`

- `RelayBatch(ctx, batchSize) (RelayResult, error)` claims up to `batchSize`
  pending entries and publishes them, all in **one transaction** so multiple relay
  instances never double-publish the same entry. `batchSize <= 0` falls back to
  `DefaultBatchSize` (100). `RelayResult` reports `Claimed` and `Published`.
  - A **publish failure does not roll back the transaction**: the entry is marked
    failed (`attempts++`) and left for the next poll to retry; once `attempts`
    reaches `DefaultMaxAttempts` (10) the entry is marked `dead`, `Metrics.IncDead`
    is counted, and a warning is logged.
  - Only a **DB access failure** (claim / mark) returns an error that rolls the
    transaction back.
- `RecordLag(ctx) error` records the age of the oldest pending entry as the outbox
  lag SLI via `Metrics.SetLagSeconds`; with no pending entries it records `0`.
- `Metrics` is the outbox-specific o11y sink: `SetLagSeconds(ctx, seconds)` and
  `IncDead(ctx)`.

### GC — `GCUsecase`

`NewGC(store, clock) GCUsecase`

- `SweepPublished(ctx, batchSize) (int64, error)` deletes `published` entries older
  than `DefaultRetention` (7 days) in batches of `batchSize` and returns the
  total deleted. `batchSize <= 0` falls back to `DefaultGCBatchSize` (10,000).
  It loops until a short batch signals no more entries remain.

### replay — `ReplayUsecase`

`NewReplay(store, tracerFactory) ReplayUsecase`

- `ReplayDead(ctx, messageID *uuid.UUID) (int64, error)` moves `dead` entries back
  to `pending` and returns the count restored. `messageID == nil` replays **all**
  dead entries; a non-nil value targets that single `message_id`.

## The consuming end

This package is only the **producing** end. What happens to a message after `relay` publishes it is
the worker subsystem's concern, and the two ends are wired by the integrator — an outbox with nothing
consuming from it is a valid configuration, not an incomplete one.

The two ends meet at the transport, not in code: `relay` hands `publisher.Message` to an adapter,
which puts the payload in the message body and the event type and `message_id` in named metadata, and
a `worker.Handler` reads them back from `worker.Message`. Neither end imports the other.

<!-- sample-api:begin -->
The sample wires both ends so the path can actually be run:

| Stage | Where |
| --- | --- |
| emit `user.withdrawn.v1` in the withdrawal transaction | `internal/usecase/user` |
| relay → publish | `outbox-relay` + `internal/infrastructure/queue/sqs` (`OUTBOX_PUBLISHER=sqs`) |
| consume → archive the withdrawal | [`internal/controller/worker/withdrawalarchive`](../../controller/worker/withdrawalarchive/README.md) |

<!-- sample-api:end -->

## Layout

| Concern | Path |
| --- | --- |
| boundary (`Store`) | `internal/usecase/boundary/outbox/` |
| usecase | `internal/usecase/outbox/` (this package) |
| infrastructure | `internal/infrastructure/rdb/system_cqrs/outbox/` |
| sqlc DML | `database/dml/system_cqrs/outbox/` |
