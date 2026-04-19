# ipextractor

English | [日本語](README.ja.md)

Configures client IP extraction strategy based on environment.

## Public API

|Function|Description|
|---|---|
|`New(e, appCfg, secCfg)`|Set IP extractor on Echo instance|
|`NewIPExtractor(appCfg, secCfg)`|Return `echo.IPExtractor` — X-Forwarded-For with CIDR trust in production, direct extraction in development|
