# httpstack

[English](README.md) | 日本語

Echo サーバ起動時に登録する **HTTP 周りの共通ミドルウェア群**をまとめたディレクトリです。

各サブパッケージは小さな役割に分割されており、アプリケーションの起動処理で組み合わせて利用します。

## 設計方針

- 各機能は小さい単位で実装し、必要なものを選んで組み合わせる
- ミドルウェアは Echo の `e.Use(...)` で登録可能な形にラップ
- このディレクトリはミドルウェアの **実装のみ** を提供し、登録は `internal/di/server/extension` で行う

## サブパッケージ一覧

### サーバ設定

|パッケージ|関数|説明|
|---|---|---|
|`banner`|`New`|本番環境で Echo バナーを非表示|
|`debugmode`|`New`|開発環境でのみデバッグモードを有効化|
|`defaultport`|`New`|本番環境でポート番号表示を非表示|
|`binder`|`New`|Echo のリクエストボディバインダーを初期化|

### ミドルウェア

|パッケージ|関数|説明|
|---|---|---|
|`requestid`|`Middleware`|X-Request-ID ヘッダの自動生成|
|`logging`|`Middleware`|HTTP リクエスト / レスポンスの構造化ログ|
|`recovery`|`Middleware`|パニックのキャッチとログ出力|
|`cors`|`Middleware`|CORS 設定|
|`security`|`Middleware`|セキュリティヘッダ（HSTS, X-Frame-Options 等）|
|`cookie`|`Middleware`|Set-Cookie ヘッダのセキュリティ属性強制|
|`forcejson`|`Middleware`|レスポンスの Content-Type を JSON に強制|
|`uri`|`Middleware`|末尾スラッシュの除去|
|`observability`|`Middleware`|OpenTelemetry トレーシング統合|
|`ratelimit`|`Middleware`|IP ベースのレートリミット（トークンバケット）|

### エラーハンドリング

|パッケージ|関数|説明|
|---|---|---|
|`errorhandler`|`New`|Echo / OpenAPI / apperror の統一エラーハンドラ|

### OpenAPI 統合

|パッケージ|関数|説明|
|---|---|---|
|`oapi`|`Middleware`|OpenAPI リクエストバリデーション|
|`oapi/auth`|`NewAuthenticator`|Cookie / Header からのトークン認証|
|`oapi/skipper`|`New`|ops エンドポイントのバリデーションスキップ|
|`oapi/validator`|`Middleware`, `GetValidator`|OpenAPI スキーマの読み込みとバリデーション|

### インフラ / ユーティリティ

|パッケージ|関数|説明|
|---|---|---|
|`basicauth`|`NewBasicAuthValidator`|メトリクスエンドポイント用 Basic 認証|
|`ipextractor`|`New`|環境に応じたクライアント IP 抽出|
|`ops`|`IsOpsPath`|運用系パス（/health, /metrics 等）の判定|

## ミドルウェア登録

ミドルウェアの登録は `internal/di/server/extension` で行います。

```go
// internal/di/server/extension での呼び出し例（概念）
func ConfigureHTTP(e *echo.Echo, cfg *config.ApplicationConfig, logger logging.Logger, lf logging.LogFieldBuilder) {
    banner.New(e, cfg)
    defaultport.New(e, cfg)
    e.Use(requestid.Middleware())
    e.Use(logging.Middleware(logger, lf))
    e.Use(recovery.Middleware(logger, lf, cfg))
    e.Use(cors.Middleware(cfg.SecurityConfig))
    e.Use(observability.Middleware(cfg))
}
```

`httpstack` 内で直接 Echo インスタンスへの登録を行わないでください。依存関係や初期化順序の問題が発生します。

## 環境による振る舞いの違い

|機能|Development|Production|
|---|---|---|
|バナー表示|表示|非表示|
|デバッグモード|有効|無効|
|ポート表示|表示|非表示|
|IP 抽出|直接抽出|X-Forwarded-For + CIDR|
|リカバリスタック|10KB（フル）|4KB（制限）|
