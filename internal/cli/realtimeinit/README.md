# realtimeinit

Core of the `realtime-init` command: creates the three Realtime Delivery tables — EventLog,
StreamTicket, InstanceLease — in the DynamoDB-compatible store that `REALTIME_*` /
`ENDPOINT_REALTIME` point at. It is an idempotent one-shot: `dynamodbclient.EnsureTable` creates a
table only when it is absent, waits for `ACTIVE`, and enables TTL only when it is not already on, so
running it again converges on the same state. The application never creates tables at start
([`docs/design/realtime-delivery.md`](../../../docs/design/realtime-delivery.md) §3.1).

|Function|Role|
|---|---|
|`Specs(cfg)`|The three table definitions, named through `RealtimeConfig` (`realtime_<kind>_<suffix>`), taken from each adapter package's `TableSpec`|
|`Run(ctx, cfg, ensure, logger)`|Ensures the tables in order and stops at the first failure with the table name attached — the rest is left for the next run, which is safe because every step is idempotent|

`Ensurer` is the seam: `cmd/realtime_init.go` passes `dynamodbclient.EnsureTable` bound to a real
client, the tests pass a recording function.

## Test strategy

The decision logic — ordering, stop-on-first-failure, the names handed to the seam — is unit-tested
with a fake `Ensurer` and never opens a connection. Whether `EnsureTable` really converges against
a DynamoDB is the contract of `internal/infrastructure/dynamodbclient`, tested there against
DynamoDB Local.
