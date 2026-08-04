# infrastructure/queue/sqs

AWS SQS reference adapter for the worker seam (`internal/usecase/boundary/worker`).

## Role

Implements the `worker.Consumer` and `worker.FailureHandler` ports (consuming side) and the
`publisher.Publisher` port (publishing side) against AWS SQS.
This is a **reference implementation** that demonstrates the seam works with a second
implementation (besides the in-memory fake) — proving the abstraction is not fake-shaped.

## Wiring and dependency isolation (E3')

Wiring this package links `aws-sdk-go-v2/service/sqs` into the binary. Because `serve` /
`worker` / `outbox-relay` are subcommands of a **single** binary, linkage cannot be scoped to
the role that consumes a queue — so isolation is defined over the **post-sample-removal** state
instead: after `make setup-remove-sample-api`, the coupling must equal what it was before the
sample was added. Any wiring from the sample set therefore carries a `sample-api` marker.
See [ADR-0106](../../../../docs/adr/0106-broker-sdk-isolation-verified-after-sample-removal.md).

To use it in production, an integrator wires `NewConsumer` / `NewDeadLetter` into a
`worker.Worker` registered in `WorkerModule`, and selects `NewPublisher` as the outbox
publish target.

Verify isolation: after a sample removal, `go list -deps ./cmd/` must not list
`github.com/aws/aws-sdk-go-v2/service/sqs`. The SDK core and `service/s3` are linked
regardless, via the object-storage adapter.

## Publishing side

`NewPublisher` implements the outbox publish boundary with `SendMessage`. The message body is the
outbox payload verbatim, so a consumer can read the dedup key without parsing the body: the outbox
`message_id` travels as the `message_id` **message attribute**, alongside the propagated headers
(`traceparent` and friends). SQS's own `MessageId` is broker-assigned and changes on every
re-publish, which is why it cannot serve as the idempotency key.

Sensitive headers (`Authorization` / `Proxy-Authorization` / `Cookie` / `Set-Cookie`) are dropped at
this egress boundary, mirroring the HTTP publisher, and empty-valued headers are skipped because SQS
rejects them with `InvalidParameterValue`.

`NewClient` builds the client; swapping endpoint and credentials is enough to target ElasticMQ,
LocalStack, or real SQS. Like the consuming side, both are built and unit-tested here but reach a
running binary only through wiring, which carries a `sample-api` marker.

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
which holds no broker-specific vocabulary. `DLQURL` is only used by `QueueStatsProvider` to read
the DLQ backlog; leave it empty to skip DLQ depth collection (the engine's dead-letter path is
`FailureHandler` / redrive, not this URL).
