# infrastructure/queue/sqs

AWS SQS reference adapter for the worker seam (`internal/usecase/boundary/worker`).

## Role

Implements the `worker.Consumer` and `worker.FailureHandler` ports (consuming side) and the
`publisher.Publisher` port (publishing side) against AWS SQS.
This is a **reference implementation** that demonstrates the seam works with a second
implementation (besides the in-memory fake) — proving the abstraction is not fake-shaped.

## Wiring and dependency isolation (E3)

Wiring this package links `aws-sdk-go-v2/service/sqs` into the binary. Because `serve` /
`worker` / `outbox-relay` are subcommands of a **single** binary, linkage cannot be scoped to
the role that consumes a queue — so isolation is defined over **coupling** instead: SQS is named
only here and in the wiring that selects this package. Any wiring from the sample set therefore
carries a `sample-api` marker.
See [ADR-0048](../../../../docs/adr/0048-broker-sdk-isolation-measured-as-coupling.md).

To use it in production, an integrator wires `NewConsumer` / `NewDeadLetter` into a
`worker.Worker` registered in `WorkerModule`, and selects `NewPublisher` as the outbox
publish target.

Verify isolation: after a sample removal, `go list -deps ./cmd/` must not list
`github.com/aws/aws-sdk-go-v2/service/sqs`. The SDK core and `service/s3` are linked
regardless, via the object-storage adapter.

## Publishing side

`NewPublisher` implements the outbox publish boundary with `SendMessage`. The message body is the
outbox payload verbatim, so a consumer can read the dedup key — and decide whether the message is
its own at all — without parsing the body: the outbox `message_id` and the event type travel as the
`message_id` and `event_type` **message attributes**, alongside the propagated headers
(`traceparent` and friends). SQS's own `MessageId` is broker-assigned and changes on every
re-publish, which is why it cannot serve as the idempotency key. `event_type` is carried because one
queue receives every kind of event the outbox emits; a consumer that could not select its own kind
before decoding would classify every other kind as a malformed payload and fill the DLQ with them.

Sensitive headers (`Authorization` / `Proxy-Authorization` / `Cookie` / `Set-Cookie`) are dropped at
this egress boundary, mirroring the HTTP publisher, and empty-valued headers are skipped because SQS
rejects them with `InvalidParameterValue`. A header named after either reserved attribute is dropped
rather than allowed to overwrite it, so what a consumer selects on cannot disagree with the outbox
row that produced the body.

SQS accepts at most ten message attributes, and the two reserved ones occupy two of them. A message that
would exceed the limit is rejected before the send with `ErrTooManyAttributes` rather than trimmed:
which headers survived a trim would follow Go's map iteration order, so a lost `traceparent` would
be neither reproducible nor visible. The relay records the error on the outbox row and the message
goes dead once it runs out of attempts, which is the correct end for a payload no queue will take.
The limit is SQS's own, so it stays here — `publisher.Message` carries no attribute count.

`NewClient` builds the client; swapping endpoint and credentials is enough to target ElasticMQ,
LocalStack, or real SQS. Credentials go through
[`infrastructure/awsclient`](../../awsclient/README.md), so leaving `AccessKeyID` /
`SecretAccessKey` empty hands resolution to the SDK's default chain (IAM role and friends) and a
deployment whose credentials do not resolve fails at startup rather than on the first send. Its
`HTTPClient` is the SSRF-guarded transport the rest of the application uses, so an endpoint pointed
at link-local — cloud metadata — is refused at dial time rather than fetched; leaving it nil falls
back to the SDK's own transport and loses that guard.

Every route from this package into a running binary carries a `sample-api` marker. On the publishing
side that is the outbox publisher's `sqs` branch. On the consuming side, a worker's adapters are
always assembled in DI, because the controller layer cannot import this package.

<!-- sample-api:begin -->
`internal/di/module/withdrawalarchive.go` is that assembly point for the bundled sample worker: it
builds `NewConsumer`, `NewDeadLetter`, and `NewQueueStatsProvider` from `CONSUMER_QUEUE_*` and hands
them to a `worker.Worker` registered in `WorkerModule`.
<!-- sample-api:end -->

## Port mapping

| seam | SQS |
| --- | --- |
| `Receive(ctx, max)` | `ReceiveMessage` (long-poll). `ApproximateReceiveCount` → `ReceiveCount`, `MessageGroupId` → `PartitionKey`, `MessageAttributes` (incl. `traceparent`) → `Attributes`, `ReceiptHandle` → reserved key `_receipt_handle` |
| `Ack` | `DeleteMessage` (reserved-key receipt handle) |
| `Nack` | `ChangeMessageVisibility(0)` (immediate redelivery, no delay) |
| `NackWithBackoff(ctx, m, d)` | `ChangeMessageVisibility(d)` (redelivery after at least `d`; sub-second `d` is rounded up with a 1s floor via `visibilitySeconds`, so a positive `d` never collapses to immediate. `d<=0` is equivalent to `Nack`) |
| `Extend` | `ChangeMessageVisibility(d)` (same `visibilitySeconds` rounding) |
| `FailureHandler.Fail` | `SendMessage` to the DLQ with a `failure_reason="permanent"` attribute. The `cause` detail is intentionally **not** included (PII/internal-detail leak guard); it is logged engine-side instead. |
| `QueueStatsProvider.QueueStats` | `GetQueueAttributes` for the source queue (and the DLQ when `DLQURL` is set). `ApproximateNumberOfMessages` → `Visible`, `ApproximateNumberOfMessagesNotVisible` → `InFlight`, `ApproximateNumberOfMessagesDelayed` → `Delayed`. Missing / unparseable attributes are treated as `0`. |

## Queue depth / DLQ (optional capability)

`NewQueueStatsProvider` implements the optional `worker.QueueStatsProvider` capability for
observing **queue depth** (backlog) — distinct from the engine's processed/failed/retry counters.
It is provided **separately** from `NewConsumer` (which keeps returning the `worker.Consumer`
interface), so the engine never learns about this broker-specific API.

SQS attribute values are **approximate**: treat the resulting `worker_queue_depth` gauge as a
backlog **trend**, not an exact count. The observability collector
(`internal/observability/metrics/queue`) scrapes this capability and never puts queue URL / ARN /
message id into metric labels.

## Dead-letter / redrive

The app-level dead-letter path is `worker.FailureHandler` (the `NewDeadLetter` handler here, which
`SendMessage`s to a DLQ). Alternatively, rely on the SQS **redrive policy**
(`maxReceiveCount` → DLQ) configured in **IaC**; in that mode do not wire `NewDeadLetter` and let
the app only monitor `ReceiveCount` (see worker invariant A7). Redrive policy is infrastructure
configuration, not application code.

## Config

`Config` here is adapter-specific (`QueueURL` / `DLQURL` / `MaxMessages` / `WaitTimeSeconds` /
`VisibilityTimeout`) and is intentionally separate from the engine-core `config.WorkerConfig`,
which holds no broker-specific vocabulary. `DLQURL` names the queue `NewDeadLetter` sends to and the
queue `QueueStatsProvider` reads the backlog from. Leaving it empty means both are out: wire no
`FailureHandler` at all and let the broker's redrive policy handle poison messages. Wiring
`NewDeadLetter` with an empty URL is the one combination to avoid — every send fails, and the engine
does not ack a message whose dead-lettering failed, so it comes back on every redelivery.
