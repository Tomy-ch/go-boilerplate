# Ready ハンドラ (`internal/controller/handler/ready`)

## 役割

`ready` は、readiness プローブエンドポイント **`GET /ready`** を公開します。

readiness プローブは「このインスタンスは今トラフィックを処理できるか？」に答えます。
依存に触れず静的な `{ "status": "ok" }` を返す liveness プローブ
（[`health`](../health) / [`healthz`](../healthz)）とは異なり、`ready` は ready を
報告する前に、重要な下流依存であるデータベースを実際に検証します。

## 確認する内容

`GetReady` は `healthcheck` usecase の `CheckHealth` に委譲します。何を確認するかは
[`internal/usecase/healthcheck/README.ja.md`](../../../usecase/healthcheck/README.ja.md)
を参照してください。

DB クエリが失敗した場合、`CheckHealth` はエラーを返し、ハンドラはそれを伝搬します
（レスポンスボディなし）。エラーは共有の [apperror](../../../apperror/README.md)
マッピングにより HTTP ステータスへ変換されます（例: DB 障害は
`apperror.ErrUnavailable` 経由で `503` として表出）。成功時は usecase が
`status: ok` を報告します。

## 標準ハンドラパターン

これは[親ハンドラガイド](../README.md)に記載された標準パターンに従う恒久ハンドラ
です。`server` 構造体を `BindHandler` が `gen.NewStrictHandler` /
`gen.RegisterHandlers` を通じて組み立て、ハンドラ本体を tracer span で包み、単一の
usecase 呼び出しを行います。

```go
func BindHandler(
    e *echo.Echo,
    tf observability.TracerFactory,
    healthUsecase healthcheckuc.Usecase,
)
```

- `healthUsecase healthcheckuc.Usecase` が実際の readiness チェックを実行します。
- `tf observability.TracerFactory` は、controller 層の `LayerTracer` を生成します。

ハンドラはリクエストコンテキストを usecase へ伝搬するため、
`ctx, endSpan := s.tracer.Start(ctx)` で `ctx` を再束縛します。

`BindHandler` は controller DI モジュール
（[`internal/di/module/controller.go`](../../../di/module/controller.go)）で
`fx.Invoke(ready.BindHandler)` として結線されています。

## レスポンス

`GetReady` は `gen.ReadyResponse`（`GetReady200JSONResponse`）を返します。

| フィールド | 由来 |
| --- | --- |
| `Status` | usecase のステータス（`ok` / `degraded` / `unhealthy`） |
| `ApplicationTime` | チェック時点の `clock.Now()` |
| `DbLatencyMs` | DB ヘルスチェックの往復レイテンシ（ミリ秒） |
| `DbRespondedAt` | DB が最後に正常応答した時刻 |
| `Dependencies` | usecase が probe した縮退しうる依存の一覧。1 つも結線されていなければ項目ごと現れない |

probe に失敗した依存があると `Status` は `degraded` になりますが、応答は `200` のままです。
instance は通常の HTTP をまだ返せるので、`503` を返すと健全なトラフィックごと
load balancer から外れてしまいます。エラーになるのはデータベースが応答しないときだけで、
共有の `apperror` の写像がそれを `503` に変えます。

`Status` は `ReadyResponseStatus` enum で、そのメンバー（`ok`・`degraded`・
`unhealthy`）は usecase のステータス定数に対応します。
