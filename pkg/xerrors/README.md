# xerrors

English | [日本語](README.ja.md)

Wraps `github.com/cockroachdb/errors` to provide error operations with stack traces.

## Wraps

`github.com/cockroachdb/errors`

## Notes

- All errors created via this package carry stack traces
- Use `Is` / `As` instead of direct `errors.Is` / `errors.As` for consistency
- Use `New` / `Wrap` / `Join` instead of `fmt.Errorf` for error creation and wrapping
  (`Join` combines multiple errors). When a value must be embedded in the message, compose it with
  `fmt.Sprintf` and pass it to `New` / `Wrap`. `fmt.Errorf` is forbidden by `forbidigo`.
