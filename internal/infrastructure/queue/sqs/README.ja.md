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

## Dead-letter / redrive

アプリレベルの dead-letter 経路は `worker.FailureHandler`（ここでの `NewDeadLetter` ハンドラ。DLQ へ
`SendMessage` する）です。あるいは、**IaC** で設定した SQS の **redrive policy**
（`maxReceiveCount` → DLQ）に委ねることもできます。そのモードでは `NewDeadLetter` を配線せず、
アプリは `ReceiveCount` の監視のみを行います（worker invariant A7 参照）。redrive policy は
アプリケーションコードではなくインフラ設定です。

## Config

ここでの `Config` はアダプタ固有（`QueueURL` / `MaxMessages` / `WaitTimeSeconds` /
`VisibilityTimeout`）であり、broker 固有の語彙を持たない engine-core の `config.WorkerConfig`
とは意図的に分離されています。
