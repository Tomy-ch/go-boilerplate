# 直接依存インベントリ

English: [dependencies.md](../../reference/dependencies.md)

本プロジェクトの直接的なサードパーティ依存を、それぞれが担う**単一の責務**でグループ化した
**生きた目録**です。ADR と異なり、この一覧は `go.mod` に追従して*変化することを前提*とした
参照であり、不変の記録ではありません。

- 依存採用の**ポリシー**（一責務 = 一関心）は決定であり ADR です:
  [`ADR-0068`](../adr/0068-library-selection-policy.ja.md)。
- 2 つの上流にまたがる **bridge / instrumentation** ライブラリは、そのポリシーの
  境界のある例外として受容します: [`ADR-0069`](../adr/0069-bridge-instrumentation-exceptions.ja.md)。

> この表は `go.mod`（`require` ブロックの非 indirect エントリ）と同期を保つこと。以下の
> バージョンはスナップショットであり正ではありません（正は `go.mod`）。

## 責務別の直接依存

| 領域 | ライブラリ | 責務 |
| --- | --- | --- |
| Web / API | `labstack/echo/v4` | HTTP web フレームワーク（[ADR-0017](../adr/0017-echo-http-framework.ja.md) 参照） |
| Web / API | `oapi-codegen/echo-middleware` | Echo 向け OpenAPI リクエスト検証ミドルウェア |
| Web / API | `oapi-codegen/runtime` | oapi-codegen 生成コードのランタイムサポート |
| Web / API | `getkin/kin-openapi` | OpenAPI 3 ドキュメントモデル / ローダ |
| Config | `caarlos0/env/v11` | 環境変数 → 構造体デコード |
| Config | `joho/godotenv` | `.env` ファイルの読み込み |
| Database | `jackc/pgx/v5` | PostgreSQL ドライバ |
| Database | `golang-migrate/migrate/v4` | スキーママイグレーション実行 |
| Errors / utils | `cockroachdb/errors` | スタックトレース付きエラーラップ |
| Errors / utils | `google/uuid` | UUID 生成（UUIDv7、[ADR-0031](../adr/0031-uuidv7-identifiers.ja.md) 参照） |
| Errors / utils | `golang.org/x/crypto` | 暗号プリミティブ |
| Errors / utils | `golang.org/x/sync` | 並行プリミティブ（errgroup など） |
| DI / logging / CLI | `go.uber.org/fx` | DI コンテナ（[ADR-0032](../adr/0032-uber-fx-di.ja.md) 参照） |
| DI / logging / CLI | `go.uber.org/zap` | 構造化ロギング |
| DI / logging / CLI | `spf13/cobra` | CLI コマンドフレームワーク |
| Testing | `go.uber.org/mock` | モック生成ランタイム |
| Testing | `stretchr/testify` | アサーション |
| Messaging / worker | `aws/aws-sdk-go-v2` | AWS API クライアントコア（worker アダプタ、opt-in — [ADR-0044](../adr/0044-sqs-adapter-opt-in.ja.md) 参照） |
| Messaging / worker | `aws/aws-sdk-go-v2/service/sqs` | SQS クライアント（pull-ack worker） |
| Metrics exposition | `prometheus/client_golang` | Prometheus 形式メトリクスエンドポイント + カスタムコレクタ |
| Metrics exposition | `prometheus/client_model` | Prometheus メトリクスデータモデル（共有型） |
| Observability (otel core) | `go.opentelemetry.io/otel`（+ `trace` / `metric`） | OpenTelemetry API |
| Observability (otel core) | `go.opentelemetry.io/otel/sdk` | OpenTelemetry トレース SDK |
| Observability (otel core) | `go.opentelemetry.io/otel/sdk/metric` | OpenTelemetry メトリクス SDK |
| Observability (otel core) | `go.opentelemetry.io/otel/sdk/log` | OpenTelemetry ログ SDK |
| Observability (otel core) | `exporters/otlp/otlptrace/{otlptracehttp,otlptracegrpc}` | OTLP トレースエクスポータ（`OBS_*` config から構築） |
| Observability (otel core) | `exporters/otlp/otlpmetric/{otlpmetrichttp,otlpmetricgrpc}` | OTLP メトリクスエクスポータ（`OBS_*` config から構築） |
| Observability (otel core) | `exporters/otlp/otlplog/{otlploghttp,otlploggrpc}` | OTLP ログエクスポータ（`OBS_*` config から構築） |
| Observability (otel core) | `contrib/instrumentation/runtime` | Go ランタイムメトリクス |

otel core グループには pre-v1.0（`v0.x`）モジュール（OTLP ログエクスポータと `sdk/log`）が
含まれますが、いずれも**単一**の上流（OpenTelemetry 自体）に結合しており 2 つではないため、
ポリシー内で例外扱いしません。OTLP エクスポータは `contrib/exporters/autoexport` ではなく
typed な `OBS_*` config から明示的に構築されます（[ADR-0062](../adr/0062-config-driven-observability-gating.ja.md) 参照）。

## bridge / instrumentation 例外

以下は**独立にバージョニングされる 2 つの上流**（フレームワーク/ライブラリ × OpenTelemetry）に
またがるため「一関心・一上流」から外れ、[ADR-0069](../adr/0069-bridge-instrumentation-exceptions.ja.md)
に基づき境界のある例外として受容します。

| ライブラリ | 結合 | 役割 |
| --- | --- | --- |
| `contrib/instrumentation/github.com/labstack/echo/otelecho` | Echo `MiddlewareFunc` × otel trace | リクエスト毎のルートサーバスパン（status / パス正規化 / W3C 伝播） |
| `contrib/instrumentation/net/http/otelhttp` | `net/http` `RoundTripper`/`Handler` × otel trace | `net/http` の計装（クライアントトランスポート + ハンドラスパン） |
| `exaring/otelpgx` | pgx `QueryTracer` × otel trace | pgx トレーサフック経由の SQL クエリスパン |
| `contrib/bridges/otelzap` | zap `zapcore.Core` × otel/log | zap レコードを OTel ログレコードへブリッジし OTLP 送出 |

> fork コスト境界の根拠となるライブラリ毎のバージョン・prod LOC 調査は、必要なら ADR の履歴に
> あります。これらは時点情報であり、ここでは意図的に維持しません。

## 補足

- 以前この依存表は `docs/decisions.md` にインラインでしたが、ドリフトしていました
  （`net/http/otelhttp` 計装と `otel/sdk/log` SDK が欠落）。**目録**（本ファイル）を
  **ポリシー**（[ADR-0068](../adr/0068-library-selection-policy.ja.md)）から分離した理由がこれで、
  不変の決定が `go.mod` を追う一覧を抱えなくて済むようになりました。
