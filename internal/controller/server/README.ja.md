# server

[English](README.md) | 日本語

`server` は、アプリケーションの **HTTP サーバー（Echo）を初期化し、DI ライフサイクルに統合する**ためのパッケージです。

extension で適用されたミドルウェア・サーバー設定を反映した完成された HTTP サーバーを起動します。

## 役割

- Echo インスタンスの生成
- DI ライフサイクル（fx / lifecycle.Registrar）へサーバー起動・停止処理を登録
- Echo コンテキストからのパラメータ抽出ユーティリティ

このパッケージでは **ミドルウェアを直接定義しません**。ミドルウェアの適用は `internal/controller/httpstack` と `internal/di/server/extension` が担います。

## 公開 API

### NewAppServer

Echo インスタンスを生成し、サーバータイムアウトを設定します。

```go
func NewAppServer(srvCfg *config.ServerConfig) *echo.Echo
```

設定される項目：

|設定|説明|
|---|---|
|`ReadHeaderTimeout`|ヘッダ読み取りタイムアウト|
|`ReadTimeout`|リクエスト読み取りタイムアウト|
|`WriteTimeout`|レスポンス書き込みタイムアウト|
|`IdleTimeout`|KeepAlive タイムアウト|

### Echo ユーティリティ

Echo コンテキストからリクエストパラメータを抽出するヘルパーです。主にロギングミドルウェアで使用されます。

|関数|説明|
|---|---|
|`ExtractPathParams`|Echo コンテキストからパスパラメータを `map[string]string` で抽出|
|`ExtractQueryParams`|Echo コンテキストからクエリパラメータを `map[string][]string` で抽出|
|`BuildHTTPRequestLogInput`|Echo コンテキストから `logging.HTTPRequestLogInput` を生成（エラーハンドラ / リカバリのログ経路で共用）|
|`MarkRecovered` / `IsRecovered`|パニックが上流で復旧済みかを記録 / 参照し、エラーハンドラの二重ログを抑止|

## 注意点

- `NewAppServer` で生成した Echo インスタンスには、後続の extension でミドルウェアが適用される
- ログには `logging.Logger` を使用し、zap の直接利用は禁止
- Graceful shutdown の Timeout は `ServerConfig` に従う
