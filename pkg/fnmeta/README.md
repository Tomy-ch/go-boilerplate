# fnmeta

English | [日本語](README.ja.md)

Extracts function and package names from runtime full function names.

Primarily used by `internal/observability` for span name generation.

## Public API

|Function|Description|
|---|---|
|`ExtractFunctionName(full)`|Extract method name from full function name|
|`ExtractPackageName(full)`|Extract package name from full function name|

## Wraps

Standard library `runtime` package.
