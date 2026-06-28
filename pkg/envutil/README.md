# envutil

English | [日本語](README.ja.md)

Small utilities for working with environment variables.

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
- `pkg/` may not depend on `internal/` or other `pkg/` packages except `pkg/xerrors` (enforced by depguard); this package uses `os` and `pkg/xerrors`.
