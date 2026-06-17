# server hook

[English](README.md) | 日本語

`internal/di/server/hook` は、アプリケーションサーバーのライフサイクルに結び付く **各種フックを登録する**パッケージです。

## フック一覧

|関数|Start|Stop|説明|
|---|---|---|---|
|`RegisterHTTPServerHooks`|Echo サーバー起動|Graceful Shutdown|HTTP サーバーのライフサイクル管理|
|`RegisterDBCloseHooks`|—|DB 接続クローズ|シャットダウン時に DB コネクションを安全に閉じる|
|`RegisterObservabilityShutdownHooks`|—|TracerProvider / MeterProvider のシャットダウン|シャットダウン時に OpenTelemetry プロバイダを flush して解放する|

## フロー

```mermaid
flowchart TB
    subgraph "Start フック"
        HTTP["Echo サーバー起動（goroutine）"]
    end

    subgraph "Stop フック"
        Shutdown["e.Shutdown()"]
        DBClose["db.Close()"]
        O11yShutdown["tp.Shutdown() / mp.Shutdown()"]
    end

    HTTP --> Shutdown
    DBClose
    O11yShutdown
```

## RegisterHTTPServerHooks

HTTP サーバーの起動・停止を `lifecycle.Registrar` に登録します。

- **Start**: goroutine で `e.Start()` を実行し、起動ログにポート / allowed_origins / CIDR / モードを出力
- **Stop**: `e.Shutdown(ctx)` で Graceful Shutdown
- `extension.AppliedServerExtends` を受け取ることで、サーバー拡張が適用された後に登録されることを保証

## RegisterDBCloseHooks

シャットダウン時にデータベース接続を閉じるフックを登録します。

- **Stop**: `db.Close()` を呼び出し、エラーがあればログに出力

## RegisterObservabilityShutdownHooks

OpenTelemetry の `TracerProvider` / `MeterProvider` のシャットダウンフックを登録します。

- **Stop**: `tp.Shutdown()` / `mp.Shutdown()` を呼び出し、バッファされた span / metric を flush してリソースを解放
- 構築（`observability.NewTracerProvider` / `NewMeterProvider`）はライフサイクル非依存で行われ、シャットダウン登録はこの hook が担う。これにより `observability` パッケージは `di/lifecycle` への依存を持たない
- otel のインターフェース（`trace.TracerProvider` / `metric.MeterProvider`）は `Shutdown` を公開しないため、具象の `*sdktrace.TracerProvider` / `*sdkmetric.MeterProvider` を受け取る

## DI 登録例

```go
fx.Invoke(
    hook.RegisterHTTPServerHooks,
    hook.RegisterDBCloseHooks,
    hook.RegisterObservabilityShutdownHooks,
)
```

## 注意点

- `RegisterHTTPServerHooks` は `AppliedServerExtends` トークンに依存するため、extension 適用後に実行される
- HTTP サーバーは goroutine で起動するため、起動失敗はログに出力されるが Start フック自体はエラーを返さない
