# infrastructure/queue/sqs

worker シーム（`internal/usecase/boundary/worker`）に対する AWS SQS の参照アダプタです。

## Role

`worker.Consumer` と `worker.FailureHandler` の各ポートを AWS SQS に対して実装します。
これは、シームが（in-memory の fake 以外の）2 つ目の実装でも成立することを示す**参照実装**であり、
この抽象が fake 都合の形になっていない（fake-shaped でない）ことを証明します。

## 配線と依存隔離（E3）

このパッケージを配線すると `aws-sdk-go-v2/service/sqs` がバイナリにリンクされます。`serve` /
`worker` / `outbox-relay` は**単一**バイナリのサブコマンドであり、キューを消費する役割だけに
リンクを限定することはできません。そのため隔離は**結合**で定義します。すなわち SQS を名指すのは
このパッケージと、それを選ぶ配線だけです。サンプル群からの配線は、いずれも `sample-api` マーカーを
伴います。
[ADR-0048](../../../../docs/ja/adr/0048-broker-sdk-isolation-measured-as-coupling.ja.md) を参照。

本番で利用するには、integrator が `NewConsumer` / `NewDeadLetter` を `WorkerModule` に登録した
`worker.Worker` に配線し、outbox の publish 先として `NewPublisher` を選びます。

隔離はリンクグラフに現れます。`github.com/aws/aws-sdk-go-v2/service/sqs` が
`go list -deps ./cmd/` に現れるのは、このパッケージを選ぶ配線があるときだけです。SDK コアと
`service/s3` は object storage adapter 経由で常にリンクされます。
本リポジトリがボイラープレートとして頒布されている間、この条件はサンプル削除を実行して結果を突き合わせる形で検査されます。詳細は [`docs/get-started/boilerplate-only-conventions.md`](../../../../docs/ja/get-started/boilerplate-only-conventions.ja.md) に記録しています。 <!-- boilerplate-only:line -->

## 送出側

`NewPublisher` は outbox の publish 境界を `SendMessage` で実装します。本文は outbox の payload を
そのまま載せ、受信側が本文を解釈せずに冪等キーを取り出せる — かつそのメッセージが自分宛かを
判定できる — よう、outbox の `message_id` とイベント種別を `message_id` / `event_type` の
**メッセージ属性**として運びます（伝搬対象ヘッダの `traceparent` 等も同様）。
SQS 自身の `MessageId` は broker が採番し再 publish のたびに変わるため、冪等キーには使えません。
`event_type` を載せるのは、1 つのキューに outbox が publish する全種別が流れるためです。受信側が
本文を解釈する前に自分の種別を選別できないと、他の種別をすべて payload 不正として扱い DLQ を
埋めてしまいます。

機微ヘッダ（`Authorization` / `Proxy-Authorization` / `Cookie` / `Set-Cookie`）は HTTP publisher と
同じくこの egress 境界で落とします。空値のヘッダは SQS が `InvalidParameterValue` で拒否するため
スキップします。予約属性と同名のヘッダは上書きさせずに落とすため、受信側が選別に使う値と、
本文を生んだ outbox 行が食い違うことはありません。

SQS のメッセージ属性は最大 10 件で、うち 2 件は予約属性が占めます。超過するメッセージは
切り詰めずに、送信前に `ErrTooManyAttributes` で弾きます。切り詰めるとどのヘッダが残るかが Go の
map の反復順に従うため、`traceparent` を失っても再現せず気付けないためです。relay がエラーを
outbox 行へ記録し、試行回数を使い切った時点で dead になります。どのキューも受け取らない
ペイロードの終わり方として、これが正しい。この上限は SQS 固有なのでここに留め、
`publisher.Message` は属性数を持ちません。

クライアントの生成は `NewClient` が担い、endpoint と資格情報の差し替えだけで ElasticMQ・LocalStack・
本番 SQS のいずれにも向けられます。資格情報は
[`infrastructure/awsclient`](../../awsclient/README.ja.md) を通すため、`AccessKeyID` /
`SecretAccessKey` を空にしておけば SDK 既定の chain（IAM ロール等）へ解決を委ねられ、解決できない
設定のデプロイは初回送信時ではなく起動時に落ちます。`HTTPClient` にはアプリの他の外部通信と同じ
SSRF ガード付き transport を渡すため、link-local（クラウドメタデータ）へ向けた endpoint は取得される
前に dial で拒否されます。nil のままにすると SDK 自身の transport に落ち、このガードを失います。

本パッケージから実行中のバイナリへ至る経路には、いずれも `sample-api` マーカーが付いています。
送出側は outbox publisher の `sqs` 分岐です。受信側は、controller 層が本パッケージを import できない
ため、worker の adapter が常に DI で組み立てられます。

<!-- sample-api:begin -->
同梱サンプルにとってのその組み立て箇所が `internal/di/module/withdrawalarchive.go` で、
`CONSUMER_QUEUE_*` から `NewConsumer` / `NewDeadLetter` / `NewQueueStatsProvider` を作り、
`WorkerModule` へ登録された `worker.Worker` へ渡します。
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
とは意図的に分離されています。`DLQURL` は `NewDeadLetter` の送出先であり、`QueueStatsProvider` が
滞留量を読む対象でもあります。空にするとその両方を持たない構成になり、`FailureHandler` を配線せず
poison message は broker の redrive policy に委ねます。避けるべき組み合わせは 1 つだけで、
URL が空のまま `NewDeadLetter` を配線することです。送出が必ず失敗し、engine は退避に失敗した
メッセージを Ack しないため、再配送のたびに戻ってきます。
