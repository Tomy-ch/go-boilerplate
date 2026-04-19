# logging

English | [日本語](README.ja.md)

HTTP request/response structured logging middleware with trace context.

## Public API

|Function / Constant|Description|
|---|---|
|`Middleware(logger, lf)`|Return Echo middleware that logs requests and responses with trace IDs, latency, and params|
|`MinStatusError`|Minimum HTTP status code treated as error (500)|

Skips ops endpoints (`/health`, `/metrics`, etc.).
