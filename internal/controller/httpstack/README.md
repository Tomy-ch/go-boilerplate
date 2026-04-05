# `httpstack` パッケージ

概要: Echo サーバ起動時に登録する HTTP 周りの共通ミドルウェア群（バナー/ポート非表示、CORS、セキュリティヘッダ、リクエストログ、エラーハンドラ、OpenAPI バリデーション など）をまとめたディレクトリです。各サブパッケージは小さな役割に分割されており、アプリケーションの起動処理で組み合わせて利用します。

用途と設計方針:

- 各機能は小さい単位で実装され、必要なものを選んで組み合わせる想定です。
- ミドルウェアは基本的に Echo の `e.Use(...)` で登録可能な形にラップされていますが、アプリケーション側の起動処理（DI）でまとめて設定するのが推奨です。

DI（依存注入）についての注意:

- この `httpstack` 配下のパッケージ自体はミドルウェアの実装を提供するのみで、実際の登録は `internal/di/server/extension` で行う設計になっています。直接このディレクトリ内で Echo インスタンスに対する登録処理を行おうとすると、依存関係や初期化順序の問題でコンパイルエラーになる可能性があります。
- したがって、ミドルウェアの登録は `internal/di/server/extension` 側から行ってください。具体的には `internal/di/server/extension` の初期化関数から個別のサブパッケージの `Middleware` や `New` を呼び出し、`e.Use(...)` や `e.HTTPErrorHandler = ...` 等を設定します。

例（概念的）:

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
    // ...その他のミドルウェアやハンドラ登録
}
```

補足:

- 個別の README は各サブパッケージ内にあり、実装の詳細と利用法が記載されています。まずはそちらを参照の上、`internal/di/server/extension` からまとめて登録してください。
