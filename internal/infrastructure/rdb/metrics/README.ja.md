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

## 注意点

- `DatabaseDriver.Stats()` から `pgxpool.Stat` を取得してメトリクスに変換
- 重複登録時は `prometheus.AlreadyRegisteredError` を無視して安全にスキップ
