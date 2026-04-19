# core module

English | [日本語](README.ja.md)

`internal/di/module/core` provides **DI module groups for core components** commonly used in the HTTP stack.

Each module returns an `fx.Option` that registers the corresponding component in the DI container.

## Module List

|Function|File|Provided Component|
|---|---|---|
|`AuthnModule()`|`auth.go`|Authentication (Authenticator + Auth controller)|
|`BasicAuthModule()`|`basicauth.go`|Basic auth validator for metrics endpoint|
|`IPRateLimiterModule()`|`ip_rate_limiter.go`|IP-based rate limiter|
|`SecurityCookieModule()`|`security_cookie.go`|Cookie security attribute configuration|
|`SkipperModule()`|`skipper.go`|Skip OpenAPI validation for ops endpoints|
|`ValidatorModule()`|`validator.go`|OpenAPI schema validator|

## Design Policy

- Each module is isolated as one file = one module
- Internal implementation simply wraps constructors from `internal/controller/httpstack` etc. with `fx.Provide`
- Does not contain business logic

## Notes

- Adding or removing modules requires updating references from the parent module in `internal/di/module`
- Tests verify that each module correctly provides its component
