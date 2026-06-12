# server DI Module

English | [日本語](README.ja.md)

This directory is the **server module layer responsible for Echo server initialization, startup, and DI management**.

Built around three `fx.Module` functions, it provides HTTP server creation, middleware aggregation, and lifecycle hook registration.

## Structure

```text
internal/di/server/
├── server.go       # Module / HookModule / MiddlewareModule
├── extension/      # Middleware and configurator DI registration
└── hook/           # Server lifecycle hooks (HTTP start/stop, DB close)
```

## Public API

|Function|Description|
|---|---|
|`Module()`|Provide `*echo.Echo` via `server.NewAppServer`|
|`HookModule()`|Register server lifecycle hooks (HTTP start/stop)|
|`MiddlewareModule()`|Aggregate all HTTP stack middleware and configurators|

### MiddlewareModule Composition

`MiddlewareModule()` aggregates the following sub-modules:

|Category|Modules|
|---|---|
|inbound|`IPExtractorModule`, `URIModule`, `OpenAPIModule`|
|outbound|`ErrorHandlerModule`, `ForceJSONModule`, `RecoveryModule`|
|security|`Module`, `CORSModule`, `CookieModule`|
|instrumentation|`RequestIDModule`, `LoggingModule`, `ObservabilityModule`|
|nonprod|`DebugModeModule`|

Additionally, `extension.ApplyExtends` is provided to apply all collected middleware and configurators to the Echo instance.

## Application Startup Order

```mermaid
flowchart LR
    Module["Module()"] --> MiddlewareModule["MiddlewareModule()"]
    MiddlewareModule --> HookModule["HookModule()"]
    HookModule --> Start["Server Start"]
```

1. `Module()` — Create Echo instance
2. `MiddlewareModule()` — Apply all middleware and configurators
3. `HookModule()` — Register start/stop hooks (server starts here)

## Subdirectories

|Directory|Description|Details|
|---|---|---|
|`extension/`|Middleware and configurator DI registration with Priority management|[README](extension/README.md)|
|`hook/`|Server lifecycle hooks (HTTP, DB close)|[README](hook/README.md)|

## Notes

- `Module()` must be loaded before `MiddlewareModule()` — Echo instance is required for middleware application
- `HookModule()` must be loaded last — server starts after middleware and configurators are applied
- `NewAppServer` has side effects and must not be referenced from domain/usecase
- Extensions are applied in the order **MiddlewareModule → HookModule**
