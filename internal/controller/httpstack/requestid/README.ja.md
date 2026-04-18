# requestid

[English](README.md) | 日本語

リクエストごとに一意な X-Request-ID ヘッダを生成します。

## 公開 API

|関数|説明|
|---|---|
|`Middleware()`|X-Request-ID を生成する Echo ミドルウェアを返す|
|`GetRequestIDFromResponse(c)`|レスポンスヘッダから Request ID を取得|
