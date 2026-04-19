# logging

[English](README.md) | 日本語

トレースコンテキスト付きの HTTP リクエスト / レスポンス構造化ログミドルウェアです。

## 公開 API

|関数 / 定数|説明|
|---|---|
|`Middleware(logger, lf)`|トレース ID、レイテンシ、パラメータ付きでリクエスト / レスポンスをログ出力する Echo ミドルウェアを返す|
|`MinStatusError`|エラーとして扱う最小 HTTP ステータスコード（500）|

ops エンドポイント（`/health`, `/metrics` 等）はスキップします。
