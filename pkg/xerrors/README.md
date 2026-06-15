# xerrors

English | [日本語](README.ja.md)

Wraps `github.com/cockroachdb/errors` to provide error operations with stack traces.

## Public API

|Function|Description|
|---|---|
|`New(msg)`|Create a new error with stack trace|
|`Wrap(err, msg)`|Wrap an existing error with message and stack trace|
|`Is(err, target)`|Check error identity (supports wrapped chains)|
|`As(err, target)`|Type-assert an error (supports wrapped chains)|
|`StackTrace(err)`|Get formatted stack trace string|
|`NewErrors()`|Return an `Errors` implementation (for DI / mocking)|

The package-level functions above are also exposed through the `Errors` interface
(`New` / `Wrap` / `Is` / `As` / `StackTrace`), which confines the dependency on
`cockroachdb/errors` to a single contract and makes it injectable / mockable.

## Wraps

`github.com/cockroachdb/errors`

## Notes

- All errors created via this package carry stack traces
- Use `Is` / `As` instead of direct `errors.Is` / `errors.As` for consistency
