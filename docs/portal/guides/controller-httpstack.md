# httpstack

English | [日本語](README.ja.md)

A directory of **common HTTP middleware** registered when starting the Echo server.

Each sub-package is split into small responsibilities and combined during application startup.

## Design Policy

- Each feature is implemented in small units, selected and combined as needed
- Middleware is wrapped in a form that can be registered with `e.Use(...)`
- This directory only provides middleware **implementations**; registration is done in `internal/di/server/extension`

## Sub-package List

### Server Configuration

|Package|Function|Description|
|---|---|---|
|`banner`|`New`|Hide Echo banner in production|
|`debugmode`|`New`|Enable debug mode in development only|
|`defaultport`|`New`|Hide port number display in production|
|`binder`|`New`|Initialize Echo request body binder|

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
|`ratelimit`|`Middleware`|IP-based rate limiting (token bucket)|

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
    banner.New(e, cfg)
    defaultport.New(e, cfg)
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
|Banner display|Shown|Hidden|
|Debug mode|Enabled|Disabled|
|Port display|Shown|Hidden|
|IP extraction|Direct|X-Forwarded-For + CIDR|
|Recovery stack|10KB (full)|4KB (limited)|
