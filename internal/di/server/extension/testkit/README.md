# testkit

English | [日本語](README.ja.md)

`testkit` provides **shared test helpers for the di server extension module tests** (the sibling `inbound` / `outbound` / `security` / `instrumentation` packages).

It centralizes the boilerplate of building a throwaway `fx` app and asserting what a module provides into a fx group, so each extension's `module_graph_test.go` stays small and consistent.

## Helpers

|Helper|Description|
|---|---|
|`RequireProvidesOne[T any](t, group, opts...)`|Asserts that the given modules provide exactly one element of type `T` to the specified fx `group`|

## Usage

```go
testkit.RequireProvidesOne[extension.UseMiddleware](t, "middlewares.use", security.Module())
```

## Notes

- Generic over `T` (the provided element type); `group` is the fx group tag (e.g. `middlewares.use`, `server.configurators`)
- Builds an `fx.App` from `opts`, runs `Start` / `Stop`, and asserts the populated slice has exactly one element
- `fx.NopLogger` only suppresses construction-time logs — it does not affect the assertion result
- Test-only helper — **must not be used in production wiring**
