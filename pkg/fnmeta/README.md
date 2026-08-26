# fnmeta

English | [日本語](README.ja.md)

Extracts function and package names from runtime full function names.

Used to derive a short, human-readable identifier (e.g. for span or log naming) from a function's runtime full name.

## Wraps

Standard library `strings` package. Parses the full function-name strings produced by `runtime` (e.g. `runtime.FuncForPC(...).Name()`), but does not import `runtime` itself — the caller obtains those names.
