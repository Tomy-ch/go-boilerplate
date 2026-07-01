# metrics

[English](README.md) | 日本語

`internal/infrastructure/rdb/metrics` は、pgxpool（PostgreSQL コネクションプール）の **統計情報を Prometheus メトリクスとして公開する**パッケージです。

## 役割

コネクションプールは Infrastructure 層で最も希少な共有資源であり、枯渇すると個々のクエリが失敗する前に接続取得待ちが積み上がってレイテンシが悪化します。本パッケージはそのプール可観測性の関心ごとを閉じ込めるために存在し、コネクションプールのランタイム統計スナップショットを標準的なメトリクスへ変換することで、飽和のシグナル（取得待ち・接続枯渇・接続の作り直し）を運用者やアラートから見えるようにします。この変換を切り出すことで、プールライブラリ固有の統計 API を他層から隠蔽し、プールの健全性を汎用的なメトリクス基盤のシグナルとして公開し、容量やタイムアウトの問題を障害化する前に検知できるようにします。

## メトリクス一覧

namespace: `pgxpool`

### Gauge（現在値）

|メトリクス名|説明|
|---|---|
|`pgxpool_acquired_conns`|現在取得中の接続数|
|`pgxpool_idle_conns`|現在アイドル中の接続数|
|`pgxpool_total_conns`|プール内の総接続数|
|`pgxpool_constructing_conns`|構築中の接続数|
|`pgxpool_max_conns`|最大許可接続数|

### Counter（累積値）

|メトリクス名|説明|
|---|---|
|`pgxpool_acquire_count_total`|接続取得成功の累計|
|`pgxpool_acquire_duration_seconds_total`|接続取得にかかった累計時間|
|`pgxpool_canceled_acquire_count_total`|キャンセルされた接続取得の累計|
|`pgxpool_empty_acquire_count_total`|プール空時に新規接続を作成した累計|
|`pgxpool_new_conns_count_total`|新規作成された接続の累計|
|`pgxpool_max_lifetime_destroy_count_total`|最大寿命により破棄された接続の累計|
|`pgxpool_max_idle_destroy_count_total`|最大アイドル時間により破棄された接続の累計|
|`pgxpool_empty_acquire_wait_time_seconds_total`|プール空時の待機累計時間|

## クエリメトリクス

プール統計が「プールが飽和していないか」を見るのに対し、クエリメトリクスは「どの DB 操作が遅い／失敗しているか」を見るためのものです。`NewQueryRecorder` は `driver.QueryRecorder` を返し、pgx クエリトレーサー（`driver.NewQueryTracer`）が `TraceQueryEnd` ごとにこれを呼び出すため、Repository / QueryService の SQL 実行経路を変更せず透過的に計装できます。

namespace: `rdb` / subsystem: `query`

|メトリクス名|型|ラベル|説明|
|---|---|---|---|
|`rdb_query_duration_seconds`|Histogram|`query_name`, `operation`, `status`|DB クエリの実行時間（秒）|
|`rdb_query_errors_total`|Counter|`query_name`, `operation`, `error_class`|失敗した DB クエリの累計|

ラベルの意味（低カーディナリティを保ち、秘匿情報は含めません）:

- `query_name`: アプリ側で管理する安定名。`driver.WithQueryName(ctx, "user.find_by_id")` で設定します。未設定は `unknown`。
- `operation`: SQL 先頭トークンのみから正規化（`select` / `insert` / `update` / `delete` / `begin` / `commit` / `rollback` / `copy` / `other`）。
- `status`: `success` / `error`。
- `error_class`: `constraint` / `timeout` / `retryable` / `connection` / `unknown`。`pgerror` の正規化を用いて分類します。`retryable` は `serialization_failure` (40001) / `deadlock_detected` (40P01)、すなわちリトライ可能なトランザクション競合を表します。

`pgx.ErrNoRows` は `status=success` として扱い、`rdb_query_errors_total` には数えません（上位層が判断する通常系の「not found」のため）。

SQL 本文・bind 値・テーブル名 / カラム名 / 制約名・PII は意図的にラベルへ含めません。その粒度の詳細はクエリログ / OTel trace を利用してください。

## 注意点

- `DatabaseDriver.Stats()` から `pgxpool.Stat` を取得してメトリクスに変換
- 重複登録時は `prometheus.AlreadyRegisteredError` を無視して安全にスキップ（クエリメトリクスは重複初期化時に登録済みコレクタを再利用）
