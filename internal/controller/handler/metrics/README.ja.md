# メトリクスハンドラー (`internal/controller/handler/metrics`)

[English](README.md) | 日本語

## 役割

`metrics` は Prometheus のスクレイプエンドポイント **`GET /metrics`** を公開します。

これは **OpenAPI に定義されない運用エンドポイント**であり、
[親ハンドラーガイド](../README.md)に記載された標準ハンドラーパターンに対する
意図的な例外です。**`gen/` パッケージ・`ServerInterface`・`server` struct・
`gen.NewStrictHandler`・Usecase 呼び出しをいずれも持たず**、Prometheus
クライアント自身の HTTP ハンドラーを Echo にマウントするだけです。

## 実装

```go
func BindHandler(e *echo.Echo, bav echomw.BasicAuthValidator) {
    e.GET("/metrics",
        echo.WrapHandler(promhttp.Handler()),
        echomw.BasicAuth(bav),
    )
}
```

- ルートは `gen.RegisterHandlers` を経由せず、
  `echo.WrapHandler(promhttp.Handler())` により `echo.HandlerFunc` として直接登録します。
- `BindHandler` はコントローラー DI モジュール
  ([`internal/di/module/controller.go`](../../../di/module/controller.go)) の
  `fx.Invoke(metrics.BindHandler)` で結線され、登録の仕組み自体は標準ハンドラーと同じです。

## 公開される内容

`promhttp.Handler()` は Prometheus の**デフォルトレジストリ**
(`prometheus.DefaultGatherer` / `prometheus.DefaultRegisterer`) を配信します。
このハンドラーパッケージ自身は何も登録せず、メトリクスは他のパッケージが
起動時にデフォルトレジストリへ登録した Collector から得られます。主なものは以下です。

- クライアントライブラリ組み込みの **Go ランタイム / プロセス Collector**
- **`app_build_info`** — ビルド・バージョン情報の info gauge。
  [`internal/observability/metrics/buildinfo`](../../../observability/metrics/buildinfo)
  が登録 (`buildinfo.Register` → `prometheus.DefaultRegisterer`)
- **RDB プール / クエリメトリクス**。
  [`internal/infrastructure/rdb/metrics`](../../../infrastructure/rdb/metrics)
  （その `NewRegisterer` は `prometheus.DefaultRegisterer` を返す）
- **HTTP RED メトリクス**。instrumentation ミドルウェアが DB モジュール提供の
  同一 `prometheus.Registerer` に登録
  （[instrumentation](../../../di/server/extension/instrumentation/README.ja.md) 参照）
- **ワーカーキューの統計 Collector**
  ([`internal/observability/metrics/queue`](../../../observability/metrics/queue),
  `RegisterStatsCollector`)

> OpenTelemetry のメトリクス (`OBS_METRICS_EXPORTER`) は **OTLP でプッシュ**され、
> ここでスクレイプされるものではありません。本エンドポイントは
> Prometheus ネイティブのデフォルトレジストリのみを配信します。

## アクセス制御

エンドポイントは Echo の Basic 認証ミドルウェア
(`echomw.BasicAuth(bav)`) で保護されます。`BasicAuthValidator` は DI により
`internal/controller/httpstack/basicauth` から供給されます
（`config.MetricsConfig` を根拠とする）。

## 標準ハンドラーパターンとの差分

| 標準 OpenAPI ハンドラー | 本エンドポイント |
| --- | --- |
| `gen.RegisterHandlers` によるルート登録 | `e.GET("/metrics", …)` の直接登録 |
| `server` struct + `gen.NewStrictHandler` | 素の `echo.WrapHandler` |
| リクエスト parse → Usecase 呼び出し → Presenter | Usecase・DTO 変換なし |
| `observability.LayerTracer` の span | span なし（トレース対象がない） |

この例外は非 OpenAPI の運用エンドポイントに限られます。機能 API は
[親ハンドラーガイド](../README.md)の標準パターンに従う必要があります。
