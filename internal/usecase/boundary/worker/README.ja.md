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

## なぜ抽象化するのか

- engine の receive → process → ack/nack ループを、実 broker なしでテスト可能にする
- engine が AWS SDK や `net/http` ではなく中立な seam に依存するようにする
- テストでモック差し替えにより決定論的な挙動を実現
- broker 固有値（receipt handle, lease）を `Message` 封筒の予約接頭辞付き属性に隔離し、エラー分類を `apperror` sentinel（`ErrRetryable` / `ErrPermanent` / `ErrFatal`）で表明する

## 任意 capability

一部の観測は broker 固有であり、すべての broker が同じ意味で表現できるわけではありません。
そのため、必須の `Consumer` interface には含めず、**任意 capability seam** として切り出します。

```go
// QueueStatsProvider は broker queue の滞留量（depth / DLQ count）を報告する。
// 任意: engine は依存しない。observability collector のみが利用する。
type QueueStatsProvider interface {
    QueueStats(ctx context.Context) (QueueStats, error)
}

type QueueStats struct {
    Source QueueDepth  // source キューの滞留量
    DLQ    *QueueDepth // DLQ が無い / 収集対象外なら nil（「0」と「対象外」を区別）
}

type QueueDepth struct {
    Visible  int64 // 受信可能なメッセージ数
    InFlight int64 // 受信済みで未確定（処理中）のメッセージ数
    Delayed  int64 // 配送遅延中のメッセージ数
}
```

depth を報告できる adapter（例: SQS）はこの capability を実装し、`Consumer` とは**別個に** provide
します（engine を broker 非依存に保つため）。collector は `internal/observability/metrics/queue` に
あり、`worker_queue_depth`（gauge）と `worker_queue_stats_collection_failures_total`（counter）を
公開します。

## 実装

- `internal/infrastructure/queue/sqs/` に `Consumer` と `FailureHandler` の具体実装（SQS adapter）、および `NewQueueStatsProvider` による任意 `QueueStatsProvider` が配置されています。
- `internal/controller/worker/` に `State` の具体実装と、これらの seam を駆動する engine が配置されています。`Worker` インスタンスは `internal/di/module/worker.go` で組み立てられます。
- `internal/observability/metrics/queue/` は `QueueStatsProvider` を scrape する Prometheus collector です。収集対象は `worker.queue_stats_targets` DI group 経由で配線します。
