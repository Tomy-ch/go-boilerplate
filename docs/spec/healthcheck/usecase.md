# Health Check — Usecase Spec

> 既存実装（`internal/usecase/healthcheck`）を spec 化したもの。手書き実装から逆生成した現状仕様。
> domain 集約を持たないシステム系 usecase。永続化集約ではなく system query（DB 死活）を参照するため domain.md は持たない。

## Overview

ヘルスチェックユースケースは、アプリケーションとデータベースの健全性を確認する。アプリケーション時刻（`clock`）と DB 死活情報（system query）を取得し、健全性ステータス（`ok` / `unhealthy`、将来的に `degraded`）を含む DTO を返す。DB チェックに失敗した場合は `unhealthy` とエラーを返す。

ドメイン集約を介さず system query を直接参照する点で通常の集約系 usecase と異なる（read-only・トランザクション不要）。

## Interface

```yaml
package: internal/usecase/healthcheck
name: Usecase
methods:
  - name: CheckHealth
    signature: CheckHealth(ctx context.Context) (DTO, error)
```

## DTOs

```yaml
- name: DTO
  description: システムの健全性を表す。Status は定数 Ok="ok" / Unhealthy="unhealthy" / Degraded="degraded" のいずれか。
  fields:
    - name: Status
      type: string
    - name: ApplicationTime
      type: time.Time
    - name: DBHealthCheck
      type: query.DBHealth
- name: query.DBHealth
  description: DB 死活情報（query サブパッケージで定義、system query が返す）。
  fields:
    - name: Ready
      type: bool
    - name: RespondedAt
      type: time.Time
    - name: Latency
      type: time.Duration
```

## Dependencies

```yaml
- tracer            # observability.TracerFactory -> LayerTracer
- clock             # boundary/clock.Clock
- db_system_query   # healthcheck/query.DBSystemQuery（DB 死活を確認する system query）
```

## Workflow

### CheckHealth

```yaml
tx_required: false
steps:
  - clock.Now でアプリケーション時刻を取得
  - db_system_query.CheckDBHealth で DB 死活情報を取得
  - エラー時は Status=Unhealthy + ApplicationTime のみ設定した DTO とエラーを返す
  - 正常時は Status=Ok + ApplicationTime + DBHealthCheck を設定した DTO を返す
calls:
  - clock.Now
  - db_system_query.CheckDBHealth
errors:
  - db_system_query.CheckDBHealth のエラーをそのまま伝播（DTO は Unhealthy）
```
