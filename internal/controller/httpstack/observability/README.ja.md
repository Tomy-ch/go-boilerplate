# observability

[English](README.md) | 日本語

Echo 用の OpenTelemetry トレーシングミドルウェアです。

## 役割

分散トレーシングは有用であるためにすべてのリクエストを一律にラップする必要があり、スパン生成を各ハンドラに通すのは反復的で誤りやすくなります。トレースの起点をミドルウェアとして分離することで、リクエストごとのサーバスパンを一箇所で開始でき、下流の層は伝播されたトレースコンテキストを自動的に引き継ぎ、ハンドラは計装のボイラープレートから解放されます。

## 補足

- `Middleware(serviceName)` は OTel（`otelecho`）トレーシングミドルウェアを返します。`PassthroughMiddleware()` は次のハンドラへそのまま渡す no-op ミドルウェアを返します。DI 層はトレースが無効（`ObservabilityConfig.TracesEnabled()` が false）のとき passthrough を選択するため、登録側で条件分岐を持たずに常にミドルウェアスロットが埋まります。
