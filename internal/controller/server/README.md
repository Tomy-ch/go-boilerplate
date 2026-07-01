# server

English | [日本語](README.ja.md)

`server` is the package that **creates and configures the HTTP server's Echo instance** and provides HTTP request logging / parameter-extraction utilities for the Echo context.

It builds the Echo instance with server timeouts from `ServerConfig`; middleware application and DI-lifecycle (start / shutdown) registration are handled by other packages (see Role).

## Role

- Echo instance creation (`NewAppServer`) -- applies the timeouts from `ServerConfig`
- Build HTTP request log input (`BuildHTTPRequestLogInput`) -- the shared construction point for error / recovery log paths
- Provide Echo context parameter extraction utilities (`ExtractPathParams` / `ExtractQueryParams`)

This package **does not define middleware directly**. Middleware application is handled by `internal/controller/httpstack` and `internal/di/server/extension`. Registering the server's start / shutdown with the DI lifecycle (`lifecycle.Registrar`) is handled by `internal/di/server/hook`, not this package.

## Notes

- The Echo instance created by `NewAppServer` receives middleware from subsequent extensions -- this package **does not define middleware directly**
- Use `logging.Logger` for logging; direct use of zap is prohibited (sealed layer)
- Graceful shutdown timeout follows `ServerConfig` -- ensure the configuration is correct
- The recovered-panic flag previously exposed here (`MarkRecovered` / `IsRecovered`) has moved to `internal/controller/ctxhelper` as the typed helpers `SetRecoveredToEcho` / `GetRecoveredFromEcho`; depend on `ctxhelper` directly rather than going through this package
