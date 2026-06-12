# envutil

English | [日本語](README.ja.md)

Small utilities for working with environment variables.

## Public API

|Function|Description|
|---|---|
|`Override(key, value string) (func(), error)`|Temporarily set an env var and return a restore function. Returns an error (leaving no side effect) if the set fails, e.g. an invalid key. The restore reverts to the previous value if it existed, otherwise unsets the key (best-effort, intended for `defer`).|

## Usage

```go
restore, err := envutil.Override("DB_NAME", "test")
if err != nil {
    return err
}
defer restore()
// ... read config while DB_NAME is "test" ...
```

## Notes

- Useful for swapping a single env var (e.g. `DB_NAME`) only during config loading, avoiding lingering global state and keeping the operation idempotent.
- `pkg/` may not depend on `internal/` or other `pkg/` packages (enforced by depguard); this package only uses `os`.
