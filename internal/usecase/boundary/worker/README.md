# worker

English | [日本語](README.ja.md)

Defines the seams (ports) for workers that consume a pull-ack class queue, plus
a broker-agnostic message envelope. Both the engine (controller layer) and the
broker adapter (infrastructure layer) depend on this boundary.

```go
// Worker is the selectable unit (1 worker type = 1 Worker).
type Worker interface {
    Name() string
    Consumer() Consumer
    Handler() Handler
    FailureHandler() FailureHandler
}

// Consumer is the minimal seam over a pull-ack queue.
type Consumer interface {
    Receive(ctx context.Context, limit int) ([]Message, error)
    Ack(ctx context.Context, m Message) error
    Nack(ctx context.Context, m Message) error
    NackWithBackoff(ctx context.Context, m Message, d time.Duration) error
    Extend(ctx context.Context, m Message, d time.Duration) error
}

// Handler runs the business processing for one message (must be idempotent).
type Handler interface {
    Handle(ctx context.Context, m Message) error
}

// FailureHandler is the dead-letter seam for Permanent messages.
type FailureHandler interface {
    Fail(ctx context.Context, m Message, cause error) error
}

// State holds the selected worker name/args and the engine's done channel.
type State interface {
    Set(name string, args []string, done chan error)
    Snapshot() (name string, args []string, done chan error)
}
```

> **`State.args` is a reserved seam — not consumed by the worker engine today.** The CLI passes
> `args` through `Set`, but the hook discards it on `Snapshot` and `engine.Run` takes only the worker
> name. It is kept deliberately to mirror the sibling `job` boundary and as a forward seam for future
> per-invocation parameters (e.g. shard / replay-offset / mode selection). Workers are currently
> configured via env, not `args`; before relying on `args`, wire an actual consumer through to
> `engine.Run` (and cover it with a test) rather than reading it off `Snapshot` alone.

## Why Abstract?

- Ensure testability of the engine's receive → process → ack/nack loop without a real broker
- Keep the engine depending on neutral seams, not on the AWS SDK or `net/http`
- Allow mock substitution in tests for deterministic behavior
- Isolate broker-specific values (receipt handle, lease) inside the `Message` envelope's reserved-prefix attributes, and express error classification via `apperror` sentinels (`ErrRetryable` / `ErrPermanent` / `ErrFatal`)

## Optional capabilities

Some observations are broker-specific and not every broker can express them the same way, so
they are **optional capability seams** kept out of the required `Consumer` interface:

```go
// QueueStatsProvider reports broker queue backlog (depth / DLQ count).
// Optional: the engine never depends on it. Only an observability collector consumes it.
type QueueStatsProvider interface {
    QueueStats(ctx context.Context) (QueueStats, error)
}

type QueueStats struct {
    Source QueueDepth  // source queue backlog
    DLQ    *QueueDepth // nil when there is no DLQ / it is not collected (distinguishes "0" from "absent")
}

type QueueDepth struct {
    Visible  int64 // receivable messages
    InFlight int64 // received-but-unacked (in-flight) messages
    Delayed  int64 // delayed messages
}
```

An adapter that can report depth (e.g. SQS) implements this capability and is provided
**separately** from its `Consumer` so the engine stays broker-agnostic. The collector lives in
`internal/observability/metrics/queue` and exposes `worker_queue_depth` (gauge) and
`worker_queue_stats_collection_failures_total` (counter).

## Implementation

- `internal/infrastructure/queue/sqs/` provides the concrete `Consumer` and `FailureHandler` (SQS adapter), plus the optional `QueueStatsProvider` via `NewQueueStatsProvider`.
- `internal/controller/worker/` provides the concrete `State` and hosts the engine that drives these seams; `Worker` instances are assembled in `internal/di/module/worker.go`.
- `internal/observability/metrics/queue/` is the Prometheus collector that scrapes `QueueStatsProvider`; targets are wired via the `worker.queue_stats_targets` DI group.
