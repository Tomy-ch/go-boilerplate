# security

English | [日本語](README.ja.md)

This directory provides fx module groups for incorporating **HTTP layer security middleware** (CORS, security headers, Cookie settings) into Echo via DI.

These are applied with unified Priority to the controller layer's middleware pipeline.

## Role

This package isolates the **security** concern of the composable server pipeline — security headers, CORS, and Cookie attributes — into a single DI unit. Grouping these cross-cutting protections behind dependency-injection registration lets them be applied with a fixed, conflict-free priority and reviewed as one surface, while ensuring the security knowledge stays in the HTTP layer and never leaks into the domain or usecase layers.

## Modules

|Module|Type|Description|
|---|---|---|
|`Module()`|Use|Security headers (HSTS, X-Frame-Options, Content-Type-Options, Referrer-Policy)|
|`CORSModule()`|Use|CORS configuration (AllowOrigins / AllowMethods / AllowHeaders)|
|`CookieModule()`|Use|Cookie security attributes (Secure / HttpOnly / SameSite)|

## Notes

- Middleware is applied in **Priority order** — managed with fixed values to avoid conflicts with other middleware
- Cookie `Secure` attribute only works over HTTPS (behavior differs in local HTTP environments)
- When updating `SecurityConfig`, review middleware settings for consistency
- Security logic is HTTP layer only — **must not leak into domain or usecase**
