# testspan

English | [日本語](README.ja.md)

Injects test trace spans into Echo request context.

## Public API

|Function|Description|
|---|---|
|`StartTestSpanForEcho(t, c)`|Embed test span into echo.Context, return cleanup function|
