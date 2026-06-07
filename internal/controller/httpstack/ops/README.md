# ops

English | [日本語](README.ja.md)

Identifies operational/infrastructure endpoints.

## Public API

|Function|Description|
|---|---|
|`IsOpsPath(path)`|Return true if path is `/metrics`, `/health`, `/healthz`, `/ready`, or `/version`|

Used by logging middleware to skip ops endpoints.
