# infrastructure/eventlog

Adapters that implement the EventLog seam (`internal/usecase/boundary/realtime.EventLogStore`) —
the bounded replay store of Realtime Delivery ([ADR-0072](../../../docs/adr/0072-postgres-state-dynamodb-eventlog.md)).

## Role

`New` in this package selects the implementation; `dynamodb/` holds the DynamoDB one. Vendor
vocabulary (table / partition key / `ConsistentRead` / `LastEvaluatedKey`) stops here — the seam
above speaks of streams, sequences and envelopes.

|Path|Role|
|---|---|
|`eventlog.go`|The single place that chooses an implementation. The DI module passes the shared `*dynamodb.Client` built by [`dynamodbclient`](../dynamodbclient/README.md)|
|`dynamodb/table.go`|`TableSpec` — the table definition `realtime-init` and the contract tests create from|
|`dynamodb/event_log.go`|The adapter|

## Port mapping

| seam | DynamoDB |
| --- | --- |
| item | partition key `stream_id` (S), sort key `sequence` (N, decimal); `event_id`, `event_type`, `occurred_at` (RFC 3339 nano, UTC), `schema_version` (N), `payload` (B, absent when empty), `expires_at` (N, epoch seconds = `occurred_at` + `realtime.EventLogRetention`), `origin` (M, the originating command's trace carrier, absent when empty). One further item per stream carries the append watermark at sort key `0` — `appended_through` (N) and **no `expires_at`**, so it outlives the events it describes; sequences start at 1, so it never collides with one |
| `Append` | `PutItem` with `attribute_not_exists(stream_id)`. On `ConditionalCheckFailedException` the existing item is read back with `ConsistentRead` and its `event_id` compared: equal ⇒ success (the outbox relay's retry is idempotent without a special case), different ⇒ `ErrSequenceConflict`. `event.Validate()` runs first, so nothing invalid is stored |
| `ReadAfter` | `Query` `stream_id = :s AND sequence > :after`, `ConsistentRead`, ascending, `Limit` (default 100, capped at 1000). `HasMore` is `LastEvaluatedKey != nil` — not `len == Limit`, because DynamoDB also stops at 1 MiB. The caller continues from the last event's sequence; no opaque cursor crosses the seam, since sequences are gap-free |
| `Latest` | `Query` descending with `Limit: 1`, `ConsistentRead`, `sequence > 0` so the watermark item is not read back as an event |
| `Find` | `GetItem` with `ConsistentRead` |
| `AppendedThrough` | `GetItem` on the watermark item with `ConsistentRead`; absent ⇒ `0`. `Append` advances it with `UpdateItem` guarded by `attribute_not_exists(appended_through) OR appended_through < :seq`, after the event is written — so a failure there leaves the watermark behind the log, never ahead, and the relay's retry closes it |

`origin` carries the trace context of the command that produced the event, so a delivery happening
minutes later on another instance can still link back to it. It is stored rather than propagated because
the reader is a different process at a different time, and it never reaches the client: the envelope's
serialized form does not include it, so the 64 KiB cap is unaffected. The store writes whatever the
carrier holds without reading it — which key means what belongs to `internal/observability`.

`expires_at` only feeds the TTL sweep. Whether a cursor is still replayable is decided in
`internal/usecase/realtime` from `OccurredAt` and `realtime.EventLogRetention`; the store does not
filter by age, so the two never disagree about the number.

## Error normalization

SDK failures become `apperror.ErrUnavailable` (context cancellation `ErrCanceled`) through
`dynamodbclient.Normalize`. An item whose shape does not match the mapping above is `ErrInternal` —
the store holds something this adapter never wrote.

## Test strategy

The adapter's substrate is DynamoDB, not the database, so its strategy is declared here rather than
inherited from [`internal/infrastructure/README.md`](../README.md):

- **Every method's `TestXxx` is a contract test against DynamoDB Local** — the shared
  `dynamodb_local` service locally, the `go-test` service container in CI — so the 1:1 mapping and
  "the same test passes against Local and production DynamoDB" are the same tests. Pointing them at
  AWS is a matter of `REALTIME_TEST_*` (see [`dynamodbclient/testkit`](../dynamodbclient/testkit/README.md)).
- Each test creates its own uniquely named table from `TableSpec` and drops it on cleanup, so runs
  from several checkouts share one DynamoDB Local without touching each other's data.
- The item mapping (`toItem` / `fromItem` / attribute readers) is unit-tested without a connection,
  including the malformed-item paths.
- DynamoDB Local does not delete expired items, so the retention test asserts that an old event is
  returned with its `OccurredAt` intact — the expiry decision is the usecase's, and that is what makes
  the test meaningful on both substrates.
