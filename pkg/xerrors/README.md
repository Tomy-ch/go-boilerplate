# xerrors

English | [日本語](README.ja.md)

Wraps `github.com/cockroachdb/errors` to provide error operations with stack traces.

## Wraps

`github.com/cockroachdb/errors`

## Notes

- All errors created via this package carry stack traces
- Use `Is` / `As` instead of direct `errors.Is` / `errors.As` for consistency
