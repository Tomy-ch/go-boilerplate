# infrastructure/realtime

Adapters for the fan-out substrate of Realtime Delivery ([ADR-0073](../../../docs/adr/0073-sns-sqs-instance-fanout.md)):
the publish side that appends an event to the EventLog and then wakes every serve instance, and the
receive side that owns one instance's queue and subscription for the lifetime of that instance.

## Role

`realtime.go` is the single place that chooses an implementation; `aws/` holds the SNS / SQS one.
SDK vocabulary (topic and queue ARNs, queue URLs, receipt handles, message attributes) stops here — the
port above speaks of an instance subscription, notifications carrying an opaque receipt, wakeups and revocations.

|Path|Role|
|---|---|
|`realtime.go`|Chooses the implementation and re-exports what the DI module, the CLI and the tests need (`NewClients`, `NewPublisher`, `NewRevocationNotifier`, `NewInstanceSubscription`, `EnsureTopic`, the two `QueueAttributes` builders, the keyed inputs `QueueAttributesInput` / `SubscriptionTarget`)|
|`aws/client.go`|`NewClients` — one credential resolution (`awsclient.Resolve`), two service clients on one endpoint (SNS and SQS share it; GoAWS and AWS differ only in endpoint and credentials)|
|`aws/publisher.go`|`boundary/publisher.Publisher` for the `realtime` outbox channel: payload → `DeliveryEvent` → `EventLogStore.Append` → SNS `Publish` of the wakeup|
|`aws/revocation.go`|`realtime.RevocationNotifier`: the revocation on the same topic, told apart by the message attribute|
|`aws/subscription.go`|`realtime.InstanceSubscription`: create queue → resolve ARN → set attributes → subscribe → `RawMessageDelivery=true`; long-poll receive; delete; unsubscribe → delete queue|
|`aws/attributes.go`|`QueueAttributes` — the attribute set an instance queue is created with; the production builder (`NewQueueAttributes(QueueAttributesInput{TopicARN, DLQARN})`) returns all of them|
|`aws/topic.go`|`EnsureTopic` — idempotent `CreateTopic` returning the ARN, for `realtime-init` and the contract tests (never at application start)|
|`aws/policy.go`|The queue access policy: `sqs:SendMessage` from `sns.amazonaws.com` only when `aws:SourceArn` is the wakeup topic|
|`aws/message.go`|The wire form: the wakeup body `{eventId, streamId, sequence}` fixed by ADR-0073, the revocation body `{subject, destination}`, and the `type` message attribute that separates them|
|`local/attributes.go`|The emulator's attribute set — only what GoAWS accepts (see below)|
|`testkit/`|Clients and a per-run topic for the contract tests; ARNs are always taken from API responses, never assembled|

## Port mapping

| seam | SNS / SQS |
| --- | --- |
| wakeup | `Publish` to the topic with body `{"eventId","streamId","sequence"}` (sequence as a decimal string) and message attribute `type=wakeup`. The body carries no payload: a client is only ever served from the EventLog, so a duplicate wakeup is the same catch-up read |
| revocation | `Publish` to the same topic with body `{"subject","destination"}` and `type=revocation` |
| `Provision(id)` | `CreateQueue(<prefix>-<id>)` → `GetQueueAttributes(QueueArn)` → `SetQueueAttributes` (the `QueueAttributes` set) → `Subscribe(protocol=sqs, endpoint=queue ARN)` → `SetSubscriptionAttributes(RawMessageDelivery=true)`. Idempotent for the same `id`; a second `id` is `ErrConflict`. A failure part-way tears down what was created so a failed start leaves nothing behind. The queue name is deterministic so the orphan-cleanup job can reach the queue from the lease alone; the subscription is found through `ListSubscriptionsByTopic` by queue ARN |
| `Receive(limit)` | `ReceiveMessage` with `WaitTimeSeconds=20`, `MaxNumberOfMessages=min(limit,10)`, `MessageAttributeNames=[All]`. A message without a readable `type` comes back with an empty `Kind` and its receipt, so the consumer can delete it instead of letting it redeliver forever |
| `Delete(n)` | `DeleteMessage` by receipt handle |
| `Teardown()` | `Unsubscribe` → `DeleteQueue`; each step is attempted even when the previous one failed, and failures are joined — whatever survives is the orphan-cleanup job's to reclaim |

Fixed values live in `aws/attributes.go`: visibility timeout 30 s, long polling 20 s, `maxReceiveCount` 5.
The topic, the queue prefix and the (optional) DLQ are deployment-dependent and come from
`RealtimeConfig` (`REALTIME_TOPIC` / `REALTIME_QUEUE_PREFIX` / `REALTIME_DLQ`; the first and last are ARNs on AWS).

## Error classification

The publisher is an outbox publisher, so its errors are what the relay's dead-letter rule ([ADR-0058](../../../docs/adr/0058-outbox-dead-on-permanent-error.md)) reads:

| failure | class | effect |
| --- | --- | --- |
| payload not a `DeliveryEvent`, invalid envelope, `eventId` ≠ outbox `message_id` (`ErrEventIDMismatch`) | `apperror.ErrPermanent` | the row is marked dead and its stream halts at the head (ordering chain, [ADR-0072](../../../docs/adr/0072-postgres-state-dynamodb-eventlog.md)) |
| `realtime.ErrSequenceConflict` (a different event already at that position) | `ErrPermanent` | same — retrying cannot make the position free |
| EventLog unreachable / SNS `Publish` failed | `ErrRetryable` (+ `ErrUnavailable`) | `next_attempt_at` advances; the retry re-appends idempotently (same `eventId` ⇒ success) and publishes again — a second wakeup, which is harmless |

The subscription and the notifier normalize SDK failures to `apperror.ErrUnavailable` (context
cancellation `ErrCanceled`).

## Emulator compatibility (`local/`)

`make realtime-smoke` against GoAWS v0.5.4 (`scripts/realtime-smoke`) showed that fan-out, the
lifecycle API, the `type` attribute and `ListSubscriptionsByTopic` are wire-compatible, and that four
queue attributes are not: `Policy` is refused (`InvalidParameterValue`), `RedrivePolicy` is accepted
but afterwards a received message can no longer be deleted, and `SqsManagedSseEnabled` /
`KmsMasterKeyId` are accepted but not stored. `local.NewQueueAttributes` therefore sets only the
timing attributes. The production builder never drops anything — a missing policy in production is a
misconfiguration to fail on, not to absorb — and the DI module picks the builder by environment.

## Test strategy

The adapter's substrate is SNS / SQS, not the database, so its strategy is declared here rather than
inherited from [`internal/infrastructure/README.md`](../README.md):

- Every method is unit-tested against the generated `SNSAPI` / `SQSAPI` mocks — the call order of
  `Provision` and `Teardown`, the partial-failure rollback, the receive parameters, and every branch of
  the error classification. These need no emulator.
- `TestInstanceSubscriptionContract` runs the provision → publish → receive → delete → teardown round
  trip against GoAWS (the shared `goaws` service locally, the `go-test` service container in CI) with a
  per-run topic and queue prefix from `testkit`. Pointing it at AWS is a matter of `REALTIME_TEST_*`
  (`REALTIME_TEST_PUBSUB_ENDPOINT` for this substrate). There is no skip: if the emulator is absent the
  test fails, the same rule the DynamoDB stores follow.
- The N-subscriber fan-out and the "mark failure after publish does not deliver twice" scenarios are
  contract tests as well (`aws/fanout_contract_test.go`): they need the publisher and the EventLog store
  together, so they run against DynamoDB Local and GoAWS side by side, with the same no-skip rule.
