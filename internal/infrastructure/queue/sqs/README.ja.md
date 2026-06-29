# infrastructure/queue/sqs

worker シーム（`internal/usecase/boundary/worker`）に対する AWS SQS の参照アダプタです。

## Role

`worker.Consumer` と `worker.FailureHandler` の各ポートを AWS SQS に対して実装します。
これは、シームが（in-memory の fake 以外の）2 つ目の実装でも成立することを示す**参照実装**であり、
この抽象が fake 都合の形になっていない（fake-shaped でない）ことを証明します。

## デフォルトでは配線されない（依存隔離, E3）

このパッケージは **`cmd/` のデフォルト配線から import されない**ため、`aws-sdk-go-v2` は
出荷バイナリ（`serve` / `worker`）に**リンクされません**。それでも CI ではビルド・テスト対象です
（`go build ./...`, `go test ./...`）。本番で利用するには、integrator が `NewConsumer` /
`NewDeadLetter` を `WorkerModule` に登録した `worker.Worker` に配線します。

隔離の検証: `./cmd/` からビルドしたバイナリに対する `go version -m <binary>` は
`github.com/aws/aws-sdk-go-v2` を列挙してはいけません。

## ポート対応

| seam | SQS |
| --- | --- |
| `Receive(ctx, max)` | `ReceiveMessage`（long-poll）。`ApproximateReceiveCount` → `ReceiveCount`、`MessageGroupId` → `PartitionKey`、`MessageAttributes`（`traceparent` を含む）→ `Attributes`、`ReceiptHandle` → 予約キー `_receipt_handle` |
| `Ack` | `DeleteMessage`（予約キーの receipt handle を使用） |
| `Nack` | `ChangeMessageVisibility(0)`（即時再配信、best-effort。遅延はポートの保証ではない） |
| `Extend` | `ChangeMessageVisibility(d)` |
| `FailureHandler.Fail` | `failure_reason="permanent"` 属性を付けて DLQ へ `SendMessage`。`cause` の詳細は意図的に**含めない**（PII / 内部詳細の漏洩ガード）。代わりに engine 側でログ出力する。 |
| `QueueStatsProvider.QueueStats` | source キュー（および `DLQURL` 設定時は DLQ）に対する `GetQueueAttributes`。`ApproximateNumberOfMessages` → `Visible`、`ApproximateNumberOfMessagesNotVisible` → `InFlight`、`ApproximateNumberOfMessagesDelayed` → `Delayed`。属性の欠落 / parse 不能は `0` 扱い。 |

## Queue depth / DLQ（任意 capability）

`NewQueueStatsProvider` は、**queue depth**（滞留量）を観測するための任意 capability
`worker.QueueStatsProvider` を実装します。これは engine の processed / failed / retry カウンタとは
別物です。`NewConsumer`（`worker.Consumer` interface を返したまま）とは**別個に** provide するため、
engine はこの broker 固有 API を知りません。

SQS の属性値は **approximate（近似値）** です。出力される `worker_queue_depth` gauge は厳密な件数では
なく滞留**傾向**として扱ってください。observability collector
（`internal/observability/metrics/queue`）がこの capability を scrape し、queue URL / ARN /
message id を metric label に入れません。

## Dead-letter / redrive

アプリレベルの dead-letter 経路は `worker.FailureHandler`（ここでの `NewDeadLetter` ハンドラ。DLQ へ
`SendMessage` する）です。あるいは、**IaC** で設定した SQS の **redrive policy**
（`maxReceiveCount` → DLQ）に委ねることもできます。そのモードでは `NewDeadLetter` を配線せず、
アプリは `ReceiveCount` の監視のみを行います（worker invariant A7 参照）。redrive policy は
アプリケーションコードではなくインフラ設定です。

## Config

ここでの `Config` はアダプタ固有（`QueueURL` / `DLQURL` / `MaxMessages` / `WaitTimeSeconds` /
`VisibilityTimeout`）であり、broker 固有の語彙を持たない engine-core の `config.WorkerConfig`
とは意図的に分離されています。`DLQURL` は `QueueStatsProvider` が DLQ の滞留量を読むためだけに使い、
空にすると DLQ depth の収集をスキップします（engine の dead-letter 経路はこの URL ではなく
`FailureHandler` / redrive です）。
