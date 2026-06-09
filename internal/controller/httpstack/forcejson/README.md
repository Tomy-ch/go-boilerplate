# forcejson

English | [日本語](README.ja.md)

Forces response Content-Type to `application/json`.

## Public API

|Function|Description|
|---|---|
|`Middleware()`|Return Echo middleware that overrides Content-Type when response is HTML or has no Content-Type|
