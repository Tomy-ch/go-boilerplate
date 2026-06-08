# envutil

English | [日本語](README.ja.md)

Small utilities for working with environment variables.

## Public API

|Function|Description|
|---|---|
|`Override(key, value string) func()`|Temporarily set an env var and return a restore function. The restore reverts to the previous value if it existed, otherwise unsets the key.|

## Usage

```go
restore := envutil.Override("DB_NAME", "test")
defer restore()
// ... read config while DB_NAME is "test" ...
```

## Notes

- Useful for swapping a single env var (e.g. `DB_NAME`) only during config loading, avoiding lingering global state and keeping the operation idempotent.
- `pkg/` may not depend on `internal/` or other `pkg/` packages (enforced by depguard); this package only uses `os`.
