# server

English | [日本語](README.ja.md)

`server` is the package that **initializes the HTTP server (Echo) and integrates it with the DI lifecycle**.

It starts a fully configured HTTP server with middleware and server settings applied by extensions.

## Role

- Echo instance creation (`NewAppServer`)
- Register server start/shutdown with the DI lifecycle (fx / lifecycle.Registrar)
- Provide Echo context parameter extraction utilities

This package **does not define middleware directly**. Middleware application is handled by `internal/controller/httpstack` and `internal/di/server/extension`.

## Public API

### NewAppServer

Creates an Echo instance and configures server timeouts.

```go
func NewAppServer(srvCfg *config.ServerConfig) *echo.Echo
```

Configured settings:

|Setting|Description|
|---|---|
|`ReadHeaderTimeout`|Header read timeout|
|`ReadTimeout`|Request read timeout|
|`WriteTimeout`|Response write timeout|
|`IdleTimeout`|KeepAlive timeout|

### Echo Utilities

Helpers to extract request parameters from Echo context. Primarily used by the logging middleware.

|Function|Description|
|---|---|
|`ExtractPathParams`|Extract path parameters as `map[string]string` from Echo context|
|`ExtractQueryParams`|Extract query parameters as `map[string][]string` from Echo context|
|`BuildHTTPRequestLogInput`|Build a `logging.HTTPRequestLogInput` from Echo context (shared by the error-handler / recovery log paths)|

The recovered-panic flag previously exposed here (`MarkRecovered` / `IsRecovered`) has been moved to `internal/controller/ctxhelper` as the typed helpers `SetRecoveredToEcho` / `GetRecoveredFromEcho`; consumers should depend on `ctxhelper` directly rather than going through this package.

## Notes

- `ServeHTTP` **registers Start / Shutdown with fx / lifecycle.Registrar only** -- do not call it directly (doing so breaks DI lifecycle management)
- The Echo instance created by `NewAppServer` receives middleware from subsequent extensions -- this package **does not define middleware directly**
- Use `logging.Logger` for logging; direct use of zap is prohibited (sealed layer)
- Security information (allowed_origins / CIDR) can be verified via startup logs
- Graceful shutdown timeout follows `ServerConfig` -- ensure the configuration is correct
