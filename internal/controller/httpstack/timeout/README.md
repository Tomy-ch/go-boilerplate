# timeout

English | [日本語](README.ja.md)

Sets a per-request deadline budget on the request context (`SERVER_REQUEST_TIMEOUT`).

## Role

This is the entry point of the **deadline budget** (M1): a single per-request deadline is set once here and propagated through `ctx` to every downstream layer — the remaining `Use` middleware, OpenAPI validation, the handler, DB queries (pgx cancels on `ctx` deadline), and outbound HTTP (`httpclient` already honours the remaining budget via `ctx.Deadline()`). Instead of placing independent timeout knobs at each boundary, every layer derives its deadline from this one budget; `statement_timeout` / `lock_timeout` (future) are coarse backstops for queries that ignore `ctx`, not the primary mechanism.

It wraps Echo's standard `middleware.ContextTimeout` (race-free; the deprecated `middleware.Timeout` has a response-writer data race). On deadline exceedance the middleware wraps the error into `apperror.ErrUnavailable` and returns it, so Echo's central unified `HTTPErrorHandler` produces the same error body shape as every other error (HTTP 503).

## Notes

- Registered as a **Pre** middleware (priority 2, just outside `uri`=1) so the deadline covers all `Use` middleware, validation, handler, and DB — see `internal/di/server/extension/inbound`.
- The timeout duration is injected from `ServerConfig.RequestTimeout()`; the package itself takes the duration as a parameter and is framework-config-free.
- `ContextTimeout` provides the deadline `ctx`; it does not forcibly interrupt a handler that ignores `ctx`. Handlers / drivers respect `ctx` (pgx, `httpclient`) so the budget is enforced cooperatively.
