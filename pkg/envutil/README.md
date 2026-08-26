# envutil

English | [日本語](README.ja.md)

Small utilities for working with environment variables.

## Usage

```go
restore, err := envutil.Override("SOME_KEY", "value")
if err != nil {
    return err
}
defer restore()
// ... read config while SOME_KEY is "value" ...
```

## Notes

- Useful for swapping a single env var only during config loading, avoiding lingering global state and keeping the operation idempotent.
- `pkg/` may not depend on `internal/` or other `pkg/` packages except `pkg/xerrors` (enforced by depguard); this package uses `os` and `pkg/xerrors`.
