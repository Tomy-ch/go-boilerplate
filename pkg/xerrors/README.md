# xerrors

English | [日本語](README.ja.md)

Wraps `github.com/cockroachdb/errors` to provide error operations with stack traces.

## Wraps

`github.com/cockroachdb/errors`

## Notes

- All errors created via this package carry stack traces
- `StackTrace(err)` returns the `%+v` representation, including the attached stack when present (`""` for a `nil` error)
- Use `Is` / `As` instead of direct `errors.Is` / `errors.As` for consistency
- Use `New` / `Wrap` / `Join` instead of `fmt.Errorf` for error creation and wrapping
  (`Join` combines multiple errors). When a value must be embedded in the message, compose it with
  `fmt.Sprintf` and pass it to `Wrap`. `fmt.Errorf` is forbidden by `forbidigo`.
- In production code, `New` belongs in a **package-level `var` declaration** — never in a function
  body. Declare the sentinel once (`var errXxx = New("...")`) and attach the dynamic context at the
  raising site with `Wrap(errXxx, ctx)`, so callers and tests identify the error with `Is` instead of
  matching its message. A static check in this repository enforces this convention; `_test.go`
  is out of scope, where an ad-hoc error to inject is a legitimate use.
- When attaching an apperror sentinel to an underlying error, prefer `Join(sentinel, err)`
  (or `Join(sentinel, Wrap(err, "context"))` when context is needed) so the original error stays
  in the chain for `Is` / `As`. Do not flatten it with `Wrap(sentinel, err.Error())`, which drops
  the original error's type and stack.
  - Exception: if the underlying error may carry sensitive data (e.g. a URL with query / userinfo),
    do not `Join` it — redact the message first and `Wrap(sentinel, <redacted string>)` so the raw
    error never propagates.
  - Caveat (load-bearing flatten): `Wrap(sentinel, err.Error())` deliberately drops the underlying
    type from the chain, so a downstream `Is` / `As` can no longer reach it. Before converting an
    existing normalizer from `Wrap` to `Join`, check every predicate that inspects the result — if one
    relies on NOT matching the underlying type, `Join` re-exposes it and silently changes behavior
    (e.g. a tx retry predicate matching `*pgconn.PgError` SQLSTATE). Keep `Wrap` when the flattening is
    intentional.
