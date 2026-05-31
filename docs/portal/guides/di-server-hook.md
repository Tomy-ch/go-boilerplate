# server hook

English | [日本語](README.ja.md)

`internal/di/server/hook` is a package that **registers lifecycle hooks** tied to the application server.

## Hook List

|Function|Start|Stop|Description|
|---|---|---|---|
|`RegisterHTTPServerHooks`|Start Echo server|Graceful Shutdown|HTTP server lifecycle management|
|`RegisterDBCloseHooks`|—|Close DB connection|Safely close DB connection on shutdown|
|`RegisterRateLimitHooks`|Start cleanup|Stop cleanup|Periodically remove expired IP rate limiter entries|

## Flow

```mermaid
flowchart TB
    subgraph "Start Hooks"
        HTTP["Echo server start (goroutine)"]
        RL["Rate limit Cleanup Ticker start"]
    end

    subgraph "Stop Hooks"
        Shutdown["e.Shutdown()"]
        DBClose["db.Close()"]
        RLStop["Cleanup Ticker stop"]
    end

    HTTP --> Shutdown
    RL --> RLStop
    DBClose
```

## RegisterHTTPServerHooks

Registers HTTP server start/stop hooks with `lifecycle.Registrar`.

- **Start**: Executes `e.Start()` in a goroutine, logs port / allowed_origins / CIDR / mode
- **Stop**: Graceful Shutdown via `e.Shutdown(ctx)`
- Receives `extension.AppliedServerExtends` to ensure registration occurs after server extensions are applied

## RegisterDBCloseHooks

Registers a hook to close the database connection on shutdown.

- **Stop**: Calls `db.Close()` and logs any errors

## RegisterRateLimitHooks

Manages periodic cleanup of the IP rate limiter.

- If `ipCfg.Enabled()` is `false`, nothing is registered
- **Start**: Starts a cleanup loop in a goroutine using `time.NewTicker(ipCfg.CleanupInterval())`
- **Stop**: Closes `stopCh` to safely terminate the loop

## DI Registration Example

```go
fx.Invoke(
    hook.RegisterHTTPServerHooks,
    hook.RegisterDBCloseHooks,
    hook.RegisterRateLimitHooks,
)
```

## Notes

- `RegisterHTTPServerHooks` depends on the `AppliedServerExtends` token, so it executes after extension application
- The HTTP server starts in a goroutine; startup failures are logged but the Start hook itself does not return an error
- Rate limit cleanup interval is controlled by `config.IPRateLimitConfig`
