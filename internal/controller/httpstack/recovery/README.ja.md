# recovery

[English](README.md) | 日本語

構造化ログ付きのパニックリカバリミドルウェアです。

## 公開 API

|関数|説明|
|---|---|
|`Middleware(z, lf, appCfg)`|パニックをキャッチし、リクエストコンテキストとスタックトレース付きでログ出力する Echo ミドルウェアを返す|

スタックサイズ: 4KB（本番）、10KB（開発）。

## errorhandler との連携

パニックをログ出力した後、ミドルウェアは `ctxhelper.SetRecoveredToEcho(c, true)` を呼びます。下流の `errorhandler` パッケージは `ctxhelper.GetRecoveredFromEcho(c)` でログ済みであることを検出し、ログの二重出力を抑止します（500 レスポンス自体は返します）。このフラグは Echo の内部ストアではなく request の `context.Context` 上に保持され、`scripts/genctxkey` が生成する typed sentinel です。
