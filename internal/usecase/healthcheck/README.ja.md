# healthcheck

サービスの健全性を報告するユースケースです。`Clock` 境界からアプリケーション
時刻を取得し、データベースを probe して、単一の health DTO を返します。

## ユースケース — `Usecase`

`New(dbSystemQuery, tracerFactory, clock, probes) Usecase`

- `CheckHealth(ctx) (*DTO, error)` は `clock.Now()` で現在時刻を取得し、続いて
  `DBSystemCqrs.CheckDBHealth(ctx)` を呼びます。DB エラー時は `nil` を返すため、
  DTO を参照してはいけません。成功時は各 `Probe` を実行し、`ApplicationTime`・
  `DBHealthCheck`・probe ごとの `DependencyStatus`・そして全 probe が通れば
  `Status = Ok`、1 つでも落ちれば `Degraded` を持つ `*DTO` を返します。

`DTO` は `Status`・`ApplicationTime`・`DBHealthCheck`（`query.DBHealth`）・
`Dependencies` を保持します。ステータス定数は `Ok`・`Degraded`・`Unhealthy` ですが、
`Unhealthy` は出力しません — 応答できないデータベースは返り値のエラーとして表面化し、
共有の `apperror` の写像が `503` に変えます。

## 縮退しうる依存 — `Probe`

`Probe` は `Name` と `Check(ctx) error` の組です。データベースはここに入りません。
このエンドポイントが存在する理由そのものであり、その失敗は degraded ではなくエラーです。
ここに入るのは、落ちていても通常の HTTP 応答を続けられる依存で、Realtime Delivery が
その最初の 1 つです。だから probe の失敗はエラーになりません — `503` を返すと、
まだ健全なトラフィックごと instance が load balancer から外れてしまいます。

このパッケージは、どの subsystem が存在するかを知りません。probe は value group
（`readiness.probes`）で届き、それを供給するのは各 subsystem を所有する DI module です。
依存の向きは subsystem → このパッケージの一方向で、逆流しません
（`internal/di/module/README.md`）。

## DB probe — `healthcheck/query`

`query` サブパッケージは薄い leaf 境界です。`DBSystemCqrs` は単一の
`CheckDBHealth(ctx) (DBHealth, error)` を持ち、`DBHealth` は `Ready`・
`RespondedAt`・`Latency` を報告します。具体的な実装は
`internal/infrastructure/rdb/system_cqrs/healthcheck/` にあり、軽量な
`SELECT 1` による liveness probe を実行します
（`database/dml/system_cqrs/health_check/`）。

## レイアウト

| 関心事 | パス |
| --- | --- |
| usecase | `internal/usecase/healthcheck/`（本パッケージ） |
| DB probe 境界 | `internal/usecase/healthcheck/query/`（`DBSystemCqrs`） |
| clock 境界 | `internal/usecase/boundary/clock/`（`Clock`） |
| infrastructure | `internal/infrastructure/rdb/system_cqrs/healthcheck/` |
| sqlc DML | `database/dml/system_cqrs/health_check/` |
