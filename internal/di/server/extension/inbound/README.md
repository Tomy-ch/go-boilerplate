# inbound

English | [日本語](README.ja.md)

`inbound` is a layer that provides **middleware and server configuration for HTTP request (input) preprocessing** via DI.

It centrally manages Binder / Validator / URI normalization / IP extraction executed at request reception, ensuring quality, safety, and consistency at the API entry point.

## Modules

|Module|Type|Description|
|---|---|---|
|`URIModule()`|Pre (priority 1)|Remove trailing slashes from URIs|
|`TimeoutModule()`|Pre (priority 2)|Per-request deadline budget (`SERVER_REQUEST_TIMEOUT`)|
|`IPExtractorModule()`|Configurator|Client IP extraction (X-Forwarded-For / direct)|
|`OpenAPIModule()`|Use|OpenAPI-based automatic request validation|

## Notes

- **Validator (OpenAPI) is UseMiddleware, URI is PreMiddleware** — designed to execute in correct order via Priority
- **Timeout is a Pre middleware (priority 2)** — placed just outside `uri` so the per-request deadline budget covers all `Use` middleware, OpenAPI validation, the handler, and DB queries. The deadline propagates via `ctx`; downstream layers (pgx, `httpclient`) honour it. This is the entry point of the deadline budget (M1); independent timeouts at each boundary are avoided in favour of this single budget
- IP Extractor depends on SecurityConfig / ApplicationConfig — **behavior may differ between production and local**
- Binder / Validator affect handlers — **must not leak logic into controller/domain layers**
- When adding new inbound features, classify them as either **ServeCfg (Echo config) or Pre/UseMiddleware**
