# redmetrics

[English](README.md) | 日本語

HTTP の RED（Rate / Errors / Duration）メトリクスの recorder と、その Echo ミドルウェアです。

## 役割

HTTP サービスの運用可観測性は RED シグナル（リクエストレート・エラーレート・レイテンシ）を土台としており、これらが信頼できるものであるためには、すべてのリクエストに対して一様に計測される必要があります。計測をミドルウェアとして分離することで、各ハンドラが独自に計測値を出さずとも 1 リクエストにつき一貫したメトリクス集合を記録できます。同時に、label 集合を意図的に低カーディナリティに抑えることで、生成される時系列が発散せず秘匿情報も含まないようにしています。

## 補足

- `Middleware(rec Recorder)` は Echo ミドルウェアを返します。error handler / recovery によって確定した status を正しく観測するため、status は `c.Response().After` フック内で読み取ります。また `After` フックはストリーミング / チャンク応答で複数回発火しうるため、`Observe` を `sync.Once` でガードし、1 リクエストにつき厳密に 1 回だけ呼び出します。
- 既知の限界: ボディ無し応答（例: `204 No Content` / `304 Not Modified`）は `Write` が呼ばれないため `After` フックが発火せず、計測されません。これは status を確定後に安全に観測するための採用に伴うトレードオフとして許容しています。
- 運用系エンドポイント（`/metrics` など）は `ops.IsOpsPath` により計測対象から除外します。
- label は `method` / `route` / `status_code` / `status_class` のみで、高カーディナリティ・秘匿情報になりうる値（raw path・query・user id 等）は含めません。`route` は Echo の route pattern（例: `/users/:id`）で、取得できない場合は `unknown` に丸めます。分類できない status も同様に `unknown` に丸めます。
- `Recorder` は 1 リクエスト分の記録インターフェースです。`PrometheusRecorder` はこれを実装し、`prometheus.Collector` も満たします。`NewPrometheusRecorder()` は副作用を持たず、`RegisterRecorder(reg, r)` がレジストリへ登録し `AlreadyRegisteredError` は無視します。
- 公開メトリクス: `http_server_requests_total`（counter）と `http_server_request_duration_seconds`（histogram, デフォルトバケット）。
- `mock/` には mockgen 生成の `Recorder` モックが置かれます。手で編集しないでください。
