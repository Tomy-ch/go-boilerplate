# infrastructure/streamticket

Adapters that implement the stream-ticket seam (`internal/usecase/boundary/realtime.StreamTicketStore`)
— the hashed, short-lived credential a client presents to open a stream
([ADR-0074](../../../docs/adr/0074-query-ticket-stream-authentication.md)).

|Path|Role|
|---|---|
|`streamticket.go`|The single place that chooses an implementation; the DI module passes the shared `*dynamodb.Client`|
|`dynamodb/table.go`|`TableSpec` — partition key `ticket_hash`, TTL on `expires_at`, GSI `by_subject_destination` (`KEYS_ONLY`)|
|`dynamodb/stream_ticket.go`|The adapter|

## Port mapping

| seam | DynamoDB |
| --- | --- |
| item | `ticket_hash` (S, partition key), `subject`, `destination`, `scope`, `initial_cursor` (N), `issued_at` (RFC 3339 nano), `expires_at` (N, epoch **seconds** — it is also the TTL attribute, so expiry is second-precise), `subject_destination` (S, `subject` + `\x1f` + `destination`, the GSI key) |
| `Save` | `PutItem` — a second save of the same hash overwrites; an empty hash is `ErrInvalidArgument` |
| `Find(hash, asOf)` | `GetItem` with `ConsistentRead` on the partition key, so the connect path never reads through the eventually consistent index. Absent, or `asOf >= expires_at`, is `ok=false` — the caller's clock decides expiry, the TTL sweep merely tidies up |
| `Invalidate(subject, destination)` | `Query` the GSI for the composite key, `DeleteItem` per hash, paginated. The GSI is eventually consistent, so a ticket saved an instant earlier may survive one call; revocation's primary mechanism is the fan-out closing the connection (`STOP`), and this is the backstop for a client that ignores it |

Why the key is the hash rather than `subject` + `destination`: the hot path is the connect-time
lookup, and putting it on the partition key keeps it strongly consistent; only invalidation — the
rare, best-effort direction — goes through the index.

## Error normalization

`dynamodbclient.Normalize` — `ErrUnavailable`, `ErrCanceled` on cancellation; an item of the wrong
shape is `ErrInternal`.

## Test strategy

Declared here because the substrate is DynamoDB, not the database (see
[`internal/infrastructure/README.md`](../README.md)):

- Every method's `TestXxx` is a contract test against DynamoDB Local (shared `dynamodb_local` locally,
  the `go-test` service container in CI), each on its own uniquely named table dropped on cleanup;
  `REALTIME_TEST_*` points the same tests at AWS DynamoDB.
- Expiry is asserted on both sides of the boundary (`asOf` one second before and exactly at
  `expires_at`), reuse by repeated `Find`, and invalidation by subject × destination leaving other
  subjects' and other destinations' tickets untouched. The invalidation test waits for the index to
  catch up before asserting, which is the eventual consistency the adapter documents.
- The item mapping (`toItem` / `fromItem` / the composite key) is unit-tested without a connection.
