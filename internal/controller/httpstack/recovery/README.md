# recovery

English | [日本語](README.ja.md)

Panic recovery middleware with structured logging.

## Public API

|Function|Description|
|---|---|
|`Middleware(z, lf, appCfg)`|Return Echo middleware that catches panics and logs with request context and stack trace|

Stack size: 4KB (production), 10KB (development).
