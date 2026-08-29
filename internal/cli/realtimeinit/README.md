# realtimeinit

Core of the `realtime-init` command: creates the three Realtime Delivery tables — EventLog,
StreamTicket, InstanceLease — in the DynamoDB-compatible store that `REALTIME_*` /
`ENDPOINT_REALTIME` point at, then the fan-out topic that `REALTIME_TOPIC_ARN` names on the SNS-compatible
endpoint `ENDPOINT_REALTIME_PUBSUB`. It is an idempotent one-shot: `dynamodbclient.EnsureTable` creates a
table only when it is absent, waits for `ACTIVE`, and enables TTL only when it is not already on, and
`CreateTopic` returns the existing topic unchanged, so running it again converges on the same state. The
application never creates tables or the topic at start
([`docs/design/realtime-delivery.md`](../../../docs/design/realtime-delivery.md) §3.1); the per-instance
queues are the only resources it creates itself.

|Function|Role|
|---|---|
|`TableNames(cfg)`|The three table names in creation order, taken from `RealtimeConfig` (`realtime_<kind>_<suffix>`)|
|`Run(ctx, cfg, ensure, logger)`|Ensures the tables in order and stops at the first failure with the table name attached — the rest is left for the next run, which is safe because every step is idempotent|
|`TopicName(arn)`|The topic name, i.e. the last `:`-separated element of `REALTIME_TOPIC_ARN`; `ErrTopicARNInvalid` when there is none|
|`RunTopic(ctx, topicARN, ensure, logger)`|Ensures the topic and requires the ARN the substrate returns to equal the configured one (`ErrTopicARNMismatch` otherwise) — an emulator whose account or region differs from the configured ARN would otherwise leave the publisher pointing at a topic nobody subscribes to|

`Ensurer` and `TopicEnsurer` are the seams and take a *name*: `cmd/realtime_init.go` (the composition root,
where infrastructure may be imported) maps each table name to its adapter package's `TableSpec` and binds
`dynamodbclient.EnsureTable` to a real client, and binds the topic name to `CreateTopic` on the SNS client
from `infrastructure/realtime`; the tests pass recording functions. The core itself imports no
infrastructure package, as `internal/cli/README.md` requires.

## Test strategy

The decision logic — ordering, stop-on-first-failure, the names handed to the seams, the ARN
comparison — is unit-tested with fake `Ensurer` / `TopicEnsurer` functions and never opens a connection. Whether `EnsureTable` really converges against
a DynamoDB is the contract of `internal/infrastructure/dynamodbclient`, tested there against
DynamoDB Local.
