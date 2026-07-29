# httpstack

English | [日本語](README.ja.md)

A directory of **common HTTP middleware** registered when starting the Echo server.

Each sub-package is split into small responsibilities and combined during application startup.

## Role

`httpstack` is the catalog of Echo middleware and server-configuration helpers used across the application. Each sub-package owns one concern (request ID, logging, recovery, CORS, security headers, etc.) and exposes a thin `Middleware(...)` or `New(...)` constructor. Middleware is intentionally registered elsewhere (`internal/di/server/extension`) so that this directory stays free of fx and Echo-instance dependencies, keeping each unit independently testable and reusable.

## Design Policy

- Each feature is implemented in small units, selected and combined as needed
- Middleware is wrapped in a form that can be registered with `e.Use(...)`
- This directory only provides middleware **implementations**; registration is done in `internal/di/server/extension`

## Sub-package List

### Middleware

|Package|Function|Description|
|---|---|---|
|`requestid`|`Middleware`|Auto-generate X-Request-ID header|
|`logging`|`Middleware`|Structured logging for HTTP request / response|
|`recovery`|`Middleware`|Catch panics and log them|
|`cors`|`Middleware`|CORS configuration|
|`security`|`Middleware`|Security headers (HSTS, X-Frame-Options, etc.)|
|`cookie`|`Middleware`|Enforce security attributes on Set-Cookie headers|
|`forcejson`|`Middleware`|Force response Content-Type to JSON|
|`uri`|`Middleware`|Remove trailing slashes|
|`bodylimit`|`Middleware`|Per-request body size limit (MB), 413 on exceed|
|`timeout`|`Middleware`|Per-request deadline budget (entry point of deadline propagation)|
|`observability`|`Middleware`|OpenTelemetry tracing integration|
|`redmetrics`|`Middleware`|HTTP RED metrics (request count / duration / status); labels limited to method / route / status_code / status_class|
|`idempotency`|`Middleware` / `StrictMiddleware`|`Idempotency-Key`-based request dedup entry point (oapi-codegen StrictMiddleware slot, not `e.Use`)|

### Error Handling

|Package|Function|Description|
|---|---|---|
|`errorhandler`|`New`|Unified error handler for Echo / OpenAPI / apperror|

### OpenAPI Integration

|Package|Function|Description|
|---|---|---|
|`oapi`|`Middleware`|OpenAPI request validation|
|`oapi/auth`|`NewAuthenticator`|Token authentication from Cookie / Header|
|`oapi/skipper`|`New`|Skip validation for ops endpoints|
|`oapi/validator`|`GetValidator`|Load and provide the OpenAPI schema (spec); validation itself is done by `oapi`|

### Infrastructure / Utilities

|Package|Function|Description|
|---|---|---|
|`basicauth`|`NewBasicAuthValidator`|Basic auth for metrics endpoint|
|`ipextractor`|`New`|Environment-aware client IP extraction|
|`ops`|`IsOpsPath`|Identify ops paths (/health, /metrics, etc.)|

## Middleware Registration

Middleware registration is done in `internal/di/server/extension`.

```go
// Conceptual example in internal/di/server/extension
func ConfigureHTTP(e *echo.Echo, cfg *config.ApplicationConfig, logger logging.Logger, lf logging.LogFieldBuilder) {
    e.Use(requestid.Middleware())
    e.Use(logging.Middleware(logger, lf))
    e.Use(recovery.Middleware(logger, lf, cfg))
    e.Use(cors.Middleware(cfg.SecurityConfig))
    e.Use(observability.Middleware(cfg))
}
```

Do not register middleware directly within `httpstack`. This can cause dependency and initialization order issues.

## Environment-dependent Behavior

|Feature|Development|Production|
|---|---|---|
|IP extraction|Direct|X-Forwarded-For + CIDR|
|Recovery stack|10KB (full)|4KB (limited)|

## Test Strategy

Each sub-package is tested **in isolation as a single middleware**. Registration order and the assembled chain belong to `internal/di/server/extension`, and the composed stack is verified by the `internal/integration` HTTP-boundary tests — do not re-test either here.

### Real vs mocked

|Dependency|Method|
|---|---|
|`*echo.Echo` / router / `*echo.Context`|real (`echo.New()` + `httptest`)|
|Downstream handler (`next`)|a test closure that records what it received / returns a fixed error|
|`logging.Logger`|`logging.NewTestLogger` / `NewObservedTestLogger` — assert on observed entries (message, field), never on a formatted log string|
|`config.*Config`|`config.MockConfigForTest` + the `Set*(t, …)` setters|
|A collaborator interface the package declares (e.g. `redmetrics.Recorder`)|generated mock under `*/mock/`|

Drive the middleware directly (`Middleware(...)(next)(c)`) when the assertion is about the returned error, and through `e.ServeHTTP` when it is about the response actually written (status / header / body) — the response is only committed on the real Echo path. A middleware that occupies the oapi-codegen `StrictMiddleware` slot (`idempotency`) is driven through that signature instead of `e.Use`.

### Viewpoints every middleware covers

- **Pass-through** — a request the middleware does not act on reaches `next` unchanged, and `next`'s return value is propagated verbatim.
- **Ops-path exclusion** — for middleware that consults `ops.IsOpsPath` (`logging` / `redmetrics`, and the `oapi/skipper` skipper), assert both sides: an ops path (`/health`, `/metrics`, …) produces no log / no metric, an application path does.
- **`server.ResponseOf` degradation** — middleware that unwraps the Echo response degrades to a plain pass-through when the writer cannot be unwrapped. Reproduce it with `c.SetResponse(httptest.NewRecorder())` and assert the middleware neither fails nor records anything. This branch is unreachable through the production stack, so the package-level test is the only thing holding it.
- **Environment-dependent branches** — when config selects a variant (`recovery` stack size, `ipextractor` extraction mode), exercise each mode through the config setters, including the unknown-mode fallback.

### Viewpoints for `Before` / `After` hooks

`server.ResponseOf(c).Before(...)` / `.After(...)` register deferred work, so the assertion belongs after the response is written — not after the middleware call returns.

- **Firing condition** — `Before` runs immediately before `WriteHeader`, which is the last point a header can still be corrected (`forcejson`); `After` runs on `Write`, once the status is final, so it observes the status the error handler / recovery ultimately produced (`logging` / `redmetrics`).
- **Repeat firing** — `After` fires per `Write`, so a chunked / streaming response invokes it more than once. A once-per-request effect must be asserted to stay once-per-request (`redmetrics` guards this with `sync.Once`).
- **Non-firing** — a body-less response (204 / 304) never calls `Write`, so `After` does not run. Where that costs an observation, the gap is a documented limitation of the hook and is pinned by a test rather than left implicit.

## Notes

- Add new middleware as its own sub-package; do not stuff multiple concerns into one package.
- Each middleware should be safe to combine in any order, but registration order in `internal/di/server/extension` does matter for `recovery` (should wrap the rest) and `requestid` (should run first so all downstream logs carry the ID).
- Do NOT call `e.Use(...)` from inside this directory — keeping registration out of `httpstack` is what allows the same middleware to be reused in tests with `testkit/testecho`.
