# Version Handler (`internal/controller/handler/version`)

English | [日本語](README.ja.md)

## Role

`version` exposes the build-information endpoint **`GET /version`**.

It returns the version metadata baked into the binary at build time (via
`ldflags`) plus the running service identity, so operators and clients can tell
exactly which build and environment they are talking to.

Unlike the liveness / readiness probes it is not a health check — it only
reports static build/identity facts and never touches the database.

## Standard handler pattern

This is a permanent handler that follows the standard pattern documented in the
[parent handler guide](../README.md): a `server` struct built by `BindHandler`
through `gen.NewStrictHandler` / `gen.RegisterHandlers`, with a tracer span
wrapping the handler body.

```go
func BindHandler(
    e *echo.Echo,
    tf observability.TracerFactory,
    loc *time.Location,
    bi system.BuildInfo,
    ac *config.ApplicationConfig,
)
```

- `bi system.BuildInfo` provides the `ldflags`-injected `Version()`,
  `Revision()`, and `BuildDate()` values.
- `ac *config.ApplicationConfig` provides the running `Env()` and `Name()`.
- `loc *time.Location` is the location used to render the build date.
- `tf observability.TracerFactory` yields the controller-layer
  `LayerTracer`.

`BindHandler` is wired in the controller DI module
([`internal/di/module/controller.go`](../../../di/module/controller.go)) with
`fx.Invoke(version.BindHandler)`.

## Response

`GetVersion` returns `gen.VersionResponse` (`GetVersion200JSONResponse`):

| Field | Source |
| --- | --- |
| `Version` | `system.BuildInfo.Version()` |
| `Revision` | `system.BuildInfo.Revision()` |
| `BuildDate` | `system.BuildInfo.BuildDate()` parsed to `loc` |
| `Environment` | `config.ApplicationConfig.Env()` |
| `Service` | `config.ApplicationConfig.Name()` |

`BuildDate` is parsed from the RFC 3339 UTC string via
`datetime.ParseRFC3339UTCToLocation` into the injected `*time.Location`. If the
injected build date is not a valid RFC 3339 UTC value, the handler returns an
error wrapping `apperror.ErrInternal` (`invalid build date`) — a broken build,
not a client error.

## Note

This handler has no downstream usecase call: since it does not propagate the
re-bound `ctx`, it starts the span with `_, endSpan := s.tracer.Start(ctx)` per
the parent guide's exception for probe-style handlers.
