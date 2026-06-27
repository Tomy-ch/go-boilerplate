# bodylimit

English | [日本語](README.ja.md)

Caps the request body size (`SERVER_BODY_LIMIT_MB`, in MB).

## Role

Bounds memory pressure and DoS surface from oversized request bodies. `Middleware(limitMB int)` thinly wraps Echo's `middleware.BodyLimit`, converting the MB value to Echo's byte string (`"%dM"`; gommon/Echo treats `1M` as 1,000,000 bytes, decimal). On exceedance Echo returns **413 Request Entity Too Large**.

## Notes

- Registered as a **Pre** middleware (priority 2) — see `internal/di/server/extension/inbound`. Echo's `BodyLimit` only wraps the body reader and is routing-independent, so placing it in `Pre` guarantees the limit applies **before** the OpenAPI validator (a `Use` middleware that decodes/buffers `requestBody`) reads the body. Registering it later would let the validator read an unbounded body on validated `POST`/`PUT` routes, silently defeating the limit.
- The limit is configured as an **integer in MB** (`ServerConfig.BodyLimitMB()`) rather than a free-form string, so the unit is unambiguous and an invalid value is rejected at config load.
- The outbound counterpart (response body) is already bounded by the HTTP client's `MaxResponseBytes`; this covers the inbound gap.
