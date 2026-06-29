# server

English | [日本語](README.ja.md)

`server` is the package that **initializes the HTTP server (Echo) and integrates it with the DI lifecycle**.

It starts a fully configured HTTP server with middleware and server settings applied by extensions.

## Role

- Echo instance creation (`NewAppServer`)
- Register server start/shutdown with the DI lifecycle (fx / lifecycle.Registrar)
- Provide Echo context parameter extraction utilities

This package **does not define middleware directly**. Middleware application is handled by `internal/controller/httpstack` and `internal/di/server/extension`.

## Notes

- `ServeHTTP` **registers Start / Shutdown with fx / lifecycle.Registrar only** -- do not call it directly (doing so breaks DI lifecycle management)
- The Echo instance created by `NewAppServer` receives middleware from subsequent extensions -- this package **does not define middleware directly**
- Use `logging.Logger` for logging; direct use of zap is prohibited (sealed layer)
- Security information (allowed_origins / CIDR) can be verified via startup logs
- Graceful shutdown timeout follows `ServerConfig` -- ensure the configuration is correct
- The recovered-panic flag previously exposed here (`MarkRecovered` / `IsRecovered`) has moved to `internal/controller/ctxhelper` as the typed helpers `SetRecoveredToEcho` / `GetRecoveredFromEcho`; depend on `ctxhelper` directly rather than going through this package
