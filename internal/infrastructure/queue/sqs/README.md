# infrastructure/queue/sqs

AWS SQS reference adapter for the worker seam (`internal/usecase/boundary/worker`).

## Role

Implements the `worker.Consumer` and `worker.FailureHandler` ports against AWS SQS.
This is a **reference implementation** that demonstrates the seam works with a second
implementation (besides the in-memory fake) — proving the abstraction is not fake-shaped.

## Not wired by default (dependency isolation, E3)

This package is **NOT imported by `cmd/`'s default wiring**, so `aws-sdk-go-v2` is **not
linked into the shipped binary** (`serve` / `worker`). It is still built and tested by CI
(`go build ./...`, `go test ./...`). To use it in production, an integrator wires
`NewConsumer` / `NewDeadLetter` into a `worker.Worker` registered in `WorkerModule`.

Verify isolation: `go version -m <binary>` for a binary built from `./cmd/` must not list
`github.com/aws/aws-sdk-go-v2`.

## Port mapping

| seam | SQS |
| --- | --- |
| `Receive(ctx, max)` | `ReceiveMessage` (long-poll). `ApproximateReceiveCount` → `ReceiveCount`, `MessageGroupId` → `PartitionKey`, `MessageAttributes` (incl. `traceparent`) → `Attributes`, `ReceiptHandle` → reserved key `_receipt_handle` |
| `Ack` | `DeleteMessage` (reserved-key receipt handle) |
| `Nack` | `ChangeMessageVisibility(0)` (immediate redelivery, best-effort; delay is not a port guarantee) |
| `Extend` | `ChangeMessageVisibility(d)` |
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
