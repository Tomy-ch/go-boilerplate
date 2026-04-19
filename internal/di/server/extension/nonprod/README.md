# nonprod

English | [日本語](README.ja.md)

`nonprod` is a layer that provides **non-production environment-only server extensions** (development / staging / local) via DI.

It groups modules that safely apply behaviors (such as Echo debug mode) that should be disabled in production, based on environment settings.

## Modules

|Module|Type|Description|
|---|---|---|
|`DebugModeModule()`|Configurator|Enable Echo debug mode in non-production environments only|

## Notes

- **Must always reference ApplicationConfig and ensure it does not run in production**
- Debug mode is "non-production only" — leaking to production creates a security risk
- ServeCfg applies side effects directly to the Echo instance — **must not depend on domain/usecase**
- To add non-production settings, extend this `nonprod` directory with new modules
