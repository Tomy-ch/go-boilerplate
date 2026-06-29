# server hook

English | [日本語](README.ja.md)

`internal/di/server/hook` is a package that **registers lifecycle hooks** tied to the application server.

## Hook List

|Function|Start|Stop|Description|
|---|---|---|---|
|`RegisterHTTPServerHooks`|Start Echo server|Graceful Shutdown|HTTP server lifecycle management|
|`RegisterDBCloseHooks`|—|Close DB connection|Safely close DB connection on shutdown|
|`RegisterObservabilityShutdownHooks`|—|Shut down TracerProvider / MeterProvider|Flush and release OpenTelemetry providers on shutdown|

## Flow

```mermaid
flowchart TB
    subgraph "Start Hooks"
        HTTP["Echo server start (goroutine)"]
    end

    subgraph "Stop Hooks"
        Shutdown["e.Shutdown()"]
        DBClose["db.Close()"]
        O11yShutdown["tp.Shutdown() / mp.Shutdown()"]
    end

    HTTP --> Shutdown
    DBClose
    O11yShutdown
```

## RegisterHTTPServerHooks

Registers HTTP server start/stop hooks with `lifecycle.Registrar`.

- **Start**: Executes `e.Start()` in a goroutine, logs port / allowed_origins / CIDR / mode
- **Stop**: Graceful Shutdown via `e.Shutdown(ctx)`
- Receives `extension.AppliedServerExtends` to ensure registration occurs after server extensions are applied

## RegisterDBCloseHooks

Registers a hook to close the database connection on shutdown.

- **Stop**: Calls `db.Close()` and logs any errors

## RegisterObservabilityShutdownHooks

Registers shutdown hooks for the OpenTelemetry `TracerProvider` / `MeterProvider`.

- **Stop**: Calls `observability.ProviderShutdowner.Shutdown()`, which flushes buffered spans / metrics and releases the `TracerProvider` / `MeterProvider`
- Construction (`observability.NewTracerProvider` / `NewMeterProvider`) is lifecycle-agnostic; this hook owns the shutdown registration, keeping the `observability` package free of any `di/lifecycle` dependency
- Receives `observability.ProviderShutdowner` — an otel-agnostic handle that bundles both providers' `Shutdown` — so that otel SDK types do not leak into the DI layer

## DI Registration Example

```go
fx.Invoke(
    hook.RegisterHTTPServerHooks,
    hook.RegisterDBCloseHooks,
    hook.RegisterObservabilityShutdownHooks,
)
```

## Notes

- `RegisterHTTPServerHooks` depends on the `AppliedServerExtends` token, so it executes after extension application
- The HTTP server starts in a goroutine; startup failures are logged but the Start hook itself does not return an error
