# queue

[English](README.md) | 日本語

`internal/observability/metrics/queue` は、**worker の broker キュー滞留量（queue depth / DLQ count）
を Prometheus メトリクスとして公開する**パッケージです。

## 役割

worker 自体は正常（処理中・低エラー率）でも、キューが詰まることがあります（producer が consumer を
上回る、DLQ にメッセージが溜まる、など）。engine の processed / failed / retry カウンタはスループットを
表すもので滞留量は表現できません。本パッケージはその差を埋めます。broker adapter（SQS など）が実装する
任意 capability `worker.QueueStatsProvider` を scrape し、現在の滞留量を gauge として公開することで、
スループットと並べて滞留を観測できるようにします。

engine は `QueueStatsProvider` に依存しません。依存するのはこの collector だけです。depth は scrape 時に
capability 経由で pull され、broker 固有 API を engine から隔離します。

## メトリクス一覧

Namespace: `worker`, Subsystem: `queue`

### Gauge（現在値）

|メトリクス名|ラベル|説明|
|---|---|---|
|`worker_queue_depth`|`worker`, `adapter`, `queue`, `state`|状態別の近似滞留量。`queue` は `source` / `dlq`、`state` は `visible` / `not_visible` / `delayed`。|

### Counter（累積値）

|メトリクス名|ラベル|説明|
|---|---|---|
|`worker_queue_stats_collection_failures_total`|`worker`, `adapter`, `queue`|scrape 時の収集失敗回数。`QueueStats` 1 回の呼び出しでは source / DLQ を区別しないため `queue` は `unknown`。|

## ラベル方針

許可: `worker` / `adapter` / `queue` / `state`。queue URL / ARN、message id、receipt handle など
高カーディナリティ・秘匿情報を含みうる値は**ラベルに入れません**。集計・アラートには worker 名と
adapter 種別で十分です。

## 補足

- 収集対象は `worker.queue_stats_targets` DI group 経由で集約します。対象が無ければ何も出力しません。
- SQS の属性値は **approximate（近似値）** です。`worker_queue_depth` は厳密な件数ではなく滞留傾向として
  扱ってください。
- scrape のたびに broker API を直接呼びます（cache なし）。将来 API rate / cost を抑えるための TTL cache を
  追加しうる設計です。
- `prometheus.AlreadyRegisteredError` を無視して二重登録を安全にスキップします。
