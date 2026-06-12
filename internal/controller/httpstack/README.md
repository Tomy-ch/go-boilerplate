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

### Server Configuration

|Package|Function|Description|
|---|---|---|
|`debugmode`|`New`|Enable debug mode in development only|

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
|`observability`|`Middleware`|OpenTelemetry tracing integration|

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
|`oapi/validator`|`Middleware`, `GetValidator`|Load OpenAPI schema and validate|

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
|Debug mode|Enabled|Disabled|
|IP extraction|Direct|X-Forwarded-For + CIDR|
|Recovery stack|10KB (full)|4KB (limited)|

## Notes

- Add new middleware as its own sub-package; do not stuff multiple concerns into one package.
- Each middleware should be safe to combine in any order, but registration order in `internal/di/server/extension` does matter for `recovery` (should wrap the rest) and `requestid` (should run first so all downstream logs carry the ID).
- Do NOT call `e.Use(...)` from inside this directory — keeping registration out of `httpstack` is what allows the same middleware to be reused in tests with `testkit/testecho`.
