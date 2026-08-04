# infrastructure/queue/sqs

worker シーム（`internal/usecase/boundary/worker`）に対する AWS SQS の参照アダプタです。

## Role

`worker.Consumer` と `worker.FailureHandler` の各ポートを AWS SQS に対して実装します。
これは、シームが（in-memory の fake 以外の）2 つ目の実装でも成立することを示す**参照実装**であり、
この抽象が fake 都合の形になっていない（fake-shaped でない）ことを証明します。

## 配線と依存隔離（E3'）

このパッケージを配線すると `aws-sdk-go-v2/service/sqs` がバイナリにリンクされます。`serve` /
`worker` / `outbox-relay` は**単一**バイナリのサブコマンドであり、キューを消費する役割だけに
リンクを限定することはできません。そのため隔離は**サンプル削除後**の状態で定義します。すなわち
`make setup-remove-sample-api` の後、結合はサンプル追加前と同一でなければなりません。サンプル群
からの配線は、いずれも `sample-api` マーカーを伴います。
[ADR-0106](../../../../docs/adr/0106-broker-sdk-isolation-verified-after-sample-removal.ja.md) を参照。

本番で利用するには、integrator が `NewConsumer` / `NewDeadLetter` を `WorkerModule` に登録した
`worker.Worker` に配線します。

隔離の検証: サンプル削除後の `go list -deps ./cmd/` は
`github.com/aws/aws-sdk-go-v2/service/sqs` を列挙してはいけません。SDK コアと `service/s3` は
object storage adapter 経由で常にリンクされます。

<!-- sample-api:begin -->
## 送出側

`NewPublisher` は outbox の publish 境界を `SendMessage` で実装します。本文は outbox の payload を
そのまま載せ、受信側が本文を解釈せずに冪等キーを取り出せるよう、outbox の `message_id` は
`message_id` **メッセージ属性**として運びます（伝搬対象ヘッダの `traceparent` 等も同様）。
SQS 自身の `MessageId` は broker が採番し再 publish のたびに変わるため、冪等キーには使えません。

機微ヘッダ（`Authorization` / `Proxy-Authorization` / `Cookie` / `Set-Cookie`）は HTTP publisher と
同じくこの egress 境界で落とします。空値のヘッダは SQS が `InvalidParameterValue` で拒否するため
スキップします。

クライアントの生成は `NewClient` が担い、endpoint と資格情報の差し替えだけで ElasticMQ・LocalStack・
本番 SQS のいずれにも向けられます。いずれも削除可能なサンプル群からのみ配線されます。
<!-- sample-api:end -->

## ポート対応

| seam | SQS |
| --- | --- |
| `Receive(ctx, max)` | `ReceiveMessage`（long-poll）。`ApproximateReceiveCount` → `ReceiveCount`、`MessageGroupId` → `PartitionKey`、`MessageAttributes`（`traceparent` を含む）→ `Attributes`、`ReceiptHandle` → 予約キー `_receipt_handle` |
| `Ack` | `DeleteMessage`（予約キーの receipt handle を使用） |
| `Nack` | `ChangeMessageVisibility(0)`（即時再配信、遅延なし） |
| `NackWithBackoff(ctx, m, d)` | `ChangeMessageVisibility(d)`（最低 `d` だけ遅延させてから再配信。サブ秒の `d` は `visibilitySeconds` で切り上げ + 1 秒下限のため、正の `d` が即時再配信へ潰れない。`d<=0` は `Nack` と等価） |
| `Extend` | `ChangeMessageVisibility(d)`（同じ `visibilitySeconds` 丸め） |
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
