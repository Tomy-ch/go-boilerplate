# healthcheck

[English](README.md) | 日本語

サービスの健全性を報告するユースケースです。`Clock` 境界からアプリケーション
時刻を取得し、データベースを probe して、単一の health DTO を返します。

## ユースケース — `Usecase`

`New(dbSystemQuery, tracerFactory, clock) Usecase`

- `CheckHealth(ctx) (*DTO, error)` は `clock.Now()` で現在時刻を取得し、続いて
  `DBSystemQuery.CheckDBHealth(ctx)` を呼びます。成功時は `Status = Ok`・
  `ApplicationTime`・`DBHealthCheck` を持つ `*DTO` を返します。DB エラー時は
  `nil` を返すため、DTO を参照してはいけません。

`DTO` は `Status`・`ApplicationTime`・`DBHealthCheck`（`query.DBHealth`）を
保持します。ステータス定数は `Ok`・`Degraded`・`Unhealthy` ですが、現在の実装は
成功時に `Ok` のみを出力します（DB エラーは `Degraded` / `Unhealthy` の DTO では
なく、返り値のエラーとして表面化します）。

## DB probe — `healthcheck/query`

`query` サブパッケージは薄い leaf 境界です。`DBSystemQuery` は単一の
`CheckDBHealth(ctx) (DBHealth, error)` を持ち、`DBHealth` は `Ready`・
`RespondedAt`・`Latency` を報告します。具体的な実装は
`internal/infrastructure/rdb/system_cqrs/healthcheck/` にあり、軽量な
`SELECT 1` による liveness probe を実行します
（`database/dml/system_cqrs/health_check/`）。

## レイアウト

| 関心事 | パス |
| --- | --- |
| usecase | `internal/usecase/healthcheck/`（本パッケージ） |
| DB probe 境界 | `internal/usecase/healthcheck/query/`（`DBSystemQuery`） |
| clock 境界 | `internal/usecase/boundary/clock/`（`Clock`） |
| infrastructure | `internal/infrastructure/rdb/system_cqrs/healthcheck/` |
| sqlc DML | `database/dml/system_cqrs/health_check/` |
