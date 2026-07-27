# server

English | [日本語](README.ja.md)

`server` is the package that **creates and configures the HTTP server** and provides HTTP request logging / parameter-extraction utilities for the Echo context.

It builds the Echo instance and the `http.Server` that serves it with the timeouts from `ServerConfig`; middleware application and DI-lifecycle (start / shutdown) registration are handled by other packages (see Role).

## Role

- Echo instance creation (`NewAppServer`)
- HTTP server creation (`NewHTTPServer`) -- serves the Echo instance and applies the timeouts from `ServerConfig`
- Build HTTP request log input (`BuildHTTPRequestLogInput`) -- the shared construction point for error / recovery log paths
- Provide Echo context parameter extraction utilities (`ExtractPathParams` / `ExtractQueryParams`) and Echo response access (`ResponseOf`)

This package **does not define middleware directly**. Middleware application is handled by `internal/controller/httpstack` and `internal/di/server/extension`. Registering the server's start / shutdown with the DI lifecycle (`lifecycle.Registrar`) is handled by `internal/di/server/hook`, not this package.

## Notes

- The Echo instance created by `NewAppServer` receives middleware from subsequent extensions -- this package **does not define middleware directly**
- Echo v5 concentrates server start / stop in `echo.StartConfig`, whose blocking model does not fit the DI container's separate start / stop hooks; the template therefore owns an `http.Server` (`NewHTTPServer`) and that is also where the request timeouts live (see [ADR-0017](../../../docs/adr/0017-echo-http-framework.md))
- `Context.Response()` returns an `http.ResponseWriter` in Echo v5; use `ResponseOf` when the Echo-specific status or `Before` / `After` hooks are needed
- Use `logging.Logger` for logging; direct use of zap is prohibited (sealed layer)
- Graceful shutdown timeout follows `ServerConfig` -- ensure the configuration is correct
- The recovered-panic flag previously exposed here (`MarkRecovered` / `IsRecovered`) has moved to `internal/controller/ctxhelper` as the typed helpers `SetRecoveredToEcho` / `GetRecoveredFromEcho`; depend on `ctxhelper` directly rather than going through this package
