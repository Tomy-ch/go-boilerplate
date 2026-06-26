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

## Why Abstract?

- Ensure testability of the engine's receive → process → ack/nack loop without a real broker
- Keep the engine depending on neutral seams, not on the AWS SDK or `net/http`
- Allow mock substitution in tests for deterministic behavior
- Isolate broker-specific values (receipt handle, lease) inside the `Message` envelope's reserved-prefix attributes, and express error classification via `apperror` sentinels (`ErrRetryable` / `ErrPermanent` / `ErrFatal`)

## Implementation

- `internal/infrastructure/queue/sqs/` provides the concrete `Consumer` and `FailureHandler` (SQS adapter).
- `internal/controller/worker/` provides the concrete `State` and hosts the engine that drives these seams; `Worker` instances are assembled in `internal/di/module/worker.go`.
