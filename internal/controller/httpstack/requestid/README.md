# requestid

English | [日本語](README.ja.md)

Generates unique X-Request-ID header for each request.

## Public API

|Function|Description|
|---|---|
|`Middleware()`|Return Echo middleware that generates X-Request-ID|
|`GetRequestIDFromResponse(c)`|Extract Request ID from response headers|
