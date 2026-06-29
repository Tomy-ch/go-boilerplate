# recovery

[English](README.md) | 日本語

構造化ログ付きのパニックリカバリミドルウェアです。

スタックサイズ: 4KB（本番）、10KB（開発）。

## 役割

いずれかのハンドラで処理されないパニックが起きると、構造化された記録を残さないままリクエスト（あるいはプロセス）がクラッシュしてしまいます。リカバリをスタックの最外層に置くことで、あらゆるパニックを一箇所でログ付きの制御された `500` レスポンスに変換でき、個々のハンドラが防御的なリカバリを持つ必要がなくなり、障害が常に観測可能になります。

## errorhandler との連携

パニックをログ出力した後、ミドルウェアは `ctxhelper.SetRecoveredToEcho(c, true)` を呼びます。下流の `errorhandler` パッケージは `ctxhelper.GetRecoveredFromEcho(c)` でログ済みであることを検出し、ログの二重出力を抑止します（500 レスポンス自体は返します）。このフラグは Echo の内部ストアではなく request の `context.Context` 上に保持され、`scripts/genctxkey` が生成する typed sentinel です。
