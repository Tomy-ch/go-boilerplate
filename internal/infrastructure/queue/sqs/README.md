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

## Dead-letter / redrive

The app-level dead-letter path is `worker.FailureHandler` (the `NewDeadLetter` handler here, which
`SendMessage`s to a DLQ). Alternatively, rely on the SQS **redrive policy**
(`maxReceiveCount` → DLQ) configured in **IaC**; in that mode do not wire `NewDeadLetter` and let
the app only monitor `ReceiveCount` (see worker invariant A7). Redrive policy is infrastructure
configuration, not application code.

## Config

`Config` here is adapter-specific (`QueueURL` / `MaxMessages` / `WaitTimeSeconds` /
`VisibilityTimeout`) and is intentionally separate from the engine-core `config.WorkerConfig`,
which holds no broker-specific vocabulary.
