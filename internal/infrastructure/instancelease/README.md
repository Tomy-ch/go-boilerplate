# infrastructure/instancelease

Adapters that implement the instance-lease seam (`internal/usecase/boundary/realtime.InstanceLeaseStore`)
— the liveness record a serve instance keeps so that its fan-out resources can be reclaimed after a
crash ([ADR-0073](../../../docs/adr/0073-sns-sqs-instance-fanout.md)). Not a lock, not a leader election.

|Path|Role|
|---|---|
|`instancelease.go`|The single place that chooses an implementation; the DI module passes the shared `*dynamodb.Client`|
|`dynamodb/table.go`|`TableSpec` — partition key `instance_id`, **no TTL**: a lease that DynamoDB deletes on its own takes the evidence of the orphan with it|
|`dynamodb/instance_lease.go`|The adapter|

## Port mapping

Timestamps are stored as epoch nanoseconds (`N`) so the store can compare them in expressions.

| seam | DynamoDB |
| --- | --- |
| `Heartbeat` | `UpdateItem` `SET heartbeat_at, expires_at` — creates the item when absent and never touches the cleanup fields, so a heartbeat racing a cleanup claim cannot erase the claim |
| `Delete` | `DeleteItem`; an absent lease is success |
| `ListExpired(asOf)` | `Scan` with `expires_at < :asOf`, `ConsistentRead`, paginated. The population is the number of serve instances, so a scan is the right shape |
| `ReleaseCleanup(release)` | `DeleteItem` under `attribute_exists(instance_id) AND cleanup_owner = :o AND expires_at < :before`. Both terms are load-bearing: the first keeps a lease closable only by whoever claimed it, and the second refuses to close one whose instance has come back — `Heartbeat` rewrites `expires_at` without touching `cleanup_owner`, so ownership alone cannot tell the two apart. A refused condition is `false, nil` |
| `AcquireCleanup(claim)` | `UpdateItem` `SET cleanup_owner, cleanup_owner_until` under `attribute_exists(instance_id) AND expires_at < :before AND (attribute_not_exists(cleanup_owner) OR cleanup_owner_until < :now)`. A refused condition is `false, nil` — someone else owns the cleanup, or there is nothing to clean — never an error. The margin between expiry and `:before` is the caller's (`internal/usecase/realtime`) |

## Error normalization

`dynamodbclient.Normalize` — `ErrUnavailable`, `ErrCanceled` on cancellation; an item of the wrong
shape is `ErrInternal`.

## Test strategy

Declared here because the substrate is DynamoDB, not the database (see
[`internal/infrastructure/README.md`](../README.md)):

- Every method's `TestXxx` is a contract test against DynamoDB Local (shared `dynamodb_local` locally,
  the `go-test` service container in CI), each on its own uniquely named table dropped on cleanup;
  `REALTIME_TEST_*` points the same tests at AWS DynamoDB.
- The ownership race is asserted directly: two claims on one expired lease, only the first is `true`;
  a claim inside the safety margin is `false`; a claim on an absent lease creates nothing.
- The item mapping (`nano` / `fromNano` / `fromItem`) is unit-tested without a connection.
