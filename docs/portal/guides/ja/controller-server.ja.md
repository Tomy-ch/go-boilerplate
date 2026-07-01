# server

[English](README.md) | 日本語

`server` は、アプリケーションの **HTTP サーバー（Echo）インスタンスを生成・設定する**とともに、Echo コンテキスト向けの HTTP リクエストログ／パラメータ抽出ユーティリティを提供するパッケージです。

`ServerConfig` のタイムアウトを反映した Echo インスタンスを構築します。ミドルウェアの適用と DI ライフサイクル（起動・停止）への登録は別パッケージが担います（役割を参照）。

## 役割

- Echo インスタンスの生成（`NewAppServer`）—— `ServerConfig` のタイムアウトを適用
- HTTP リクエストのログ入力の組み立て（`BuildHTTPRequestLogInput`）—— エラー／リカバリのログ経路の共通生成点
- Echo コンテキストからのパラメータ抽出ユーティリティ（`ExtractPathParams` / `ExtractQueryParams`）

このパッケージでは **ミドルウェアを直接定義しません**。ミドルウェアの適用は `internal/controller/httpstack` と `internal/di/server/extension` が担います。サーバーの起動・停止処理の DI ライフサイクル（`lifecycle.Registrar`）への登録は、本パッケージではなく `internal/di/server/hook` が担います。

## 注意点

- `NewAppServer` で生成した Echo インスタンスには、後続の extension でミドルウェアが適用される —— 本パッケージは **ミドルウェアを直接定義しない**
- ログには `logging.Logger` を使用し、zap の直接利用は禁止（sealed layer）
- Graceful shutdown の Timeout は `ServerConfig` に従う —— 設定が正しいことを確認すること
- かつて本パッケージで公開していた「パニックが上流で復旧済みか」のフラグ（`MarkRecovered` / `IsRecovered`）は `internal/controller/ctxhelper` に移し、typed helper の `SetRecoveredToEcho` / `GetRecoveredFromEcho` として提供しています。利用側は本パッケージ経由ではなく `ctxhelper` を直接参照してください
