# worker

[English](README.md) | 日本語

pull-ack クラスのキューを consume する worker の seam（port）と、broker 非依存のメッセージ封筒を定義します。
engine（controller 層）と broker adapter（infrastructure 層）の双方がこの境界に依存します。

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

## なぜ抽象化するのか

- engine の receive → process → ack/nack ループを、実 broker なしでテスト可能にする
- engine が AWS SDK や `net/http` ではなく中立な seam に依存するようにする
- テストでモック差し替えにより決定論的な挙動を実現
- broker 固有値（receipt handle, lease）を `Message` 封筒の予約接頭辞付き属性に隔離し、エラー分類を `apperror` sentinel（`ErrRetryable` / `ErrPermanent` / `ErrFatal`）で表明する

## 実装

- `internal/infrastructure/queue/sqs/` に `Consumer` と `FailureHandler` の具体実装（SQS adapter）が配置されています。
- `internal/controller/worker/` に `State` の具体実装と、これらの seam を駆動する engine が配置されています。`Worker` インスタンスは `internal/di/module/worker.go` で組み立てられます。
