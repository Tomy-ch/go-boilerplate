# inbound

English | [日本語](README.ja.md)

`inbound` is a layer that provides **middleware and server configuration for HTTP request (input) preprocessing** via DI.

It centrally manages Binder / Validator / URI normalization / IP extraction executed at request reception, ensuring quality, safety, and consistency at the API entry point.

## Modules

|Module|Type|Description|
|---|---|---|
|`URIModule()`|Pre (priority 1)|Remove trailing slashes from URIs|
|`BodyLimitModule()`|Pre (priority 2)|Request body size limit (`SERVER_BODY_LIMIT_MB`); applied before OpenAPI validation reads the body|
|`IPExtractorModule()`|Configurator|Client IP extraction (X-Forwarded-For / direct)|
|`OpenAPIModule()`|Use|OpenAPI-based automatic request validation|

## Notes

- **Validator (OpenAPI) is UseMiddleware, URI is PreMiddleware** — designed to execute in correct order via Priority
- IP Extractor depends on SecurityConfig / ApplicationConfig — **behavior may differ between production and local**
- Binder / Validator affect handlers — **must not leak logic into controller/domain layers**
- When adding new inbound features, classify them as either **ServeCfg (Echo config) or Pre/UseMiddleware**
