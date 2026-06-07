# shutdowner DI Wrapper

English | [日本語](README.ja.md)

This package provides a `Shutdowner` interface that abstracts `fx.Shutdowner` from `go.uber.org/fx`. It wraps the `fx.Shutdowner` obtained from the DI container to make it easier to use from application code and tests.

## Public API

|Type / Function|Description|
|---|---|
|`Shutdowner`|Interface abstracting `Shutdown(...fx.ShutdownOption) error`|
|`NewShutdowner(sd fx.Shutdowner)`|Create a concrete instance wrapping `fx.Shutdowner`|
|`Module()`|Return an `fx.Module` that registers `Shutdowner` in the DI container|

## Why Abstract?

- Using `fx.Shutdowner` directly couples application code to the fx framework
- The `Shutdowner` interface allows easy mock injection in tests
- Keeps the fx dependency confined to the DI layer

## Notes

- The wrapper is extremely thin — it simply holds `fx.Shutdowner` and delegates `Shutdown` calls
- `Shutdown` triggers process stop and cleanup, so callers should be aware of side effects
- `mock/` contains auto-generated mocks via `mockgen`
