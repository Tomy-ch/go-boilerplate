# Ready Handler (`internal/controller/handler/ready`)

English | [日本語](README.ja.md)

## Role

`ready` exposes the readiness-probe endpoint **`GET /ready`**.

A readiness probe answers "can this instance serve traffic right now?". Unlike
the liveness probes ([`health`](../health) / [`healthz`](../healthz)), which
return a static `{ "status": "ok" }` without touching any dependency, `ready`
actually verifies the critical downstream dependency — the database — before
reporting ready.

## What it checks

`GetReady` delegates to the `healthcheck` usecase's `CheckHealth`, which:

1. Reads the application's current time from the injected `clock` boundary
   (returned as `applicationTime`).
2. Runs a lightweight database health-check query
   (`SELECT 1`, `GetDBHealthCheck`) via the RDB system query, measuring the
   round-trip latency and the time the DB last responded successfully.

If the database query fails, `CheckHealth` returns an error and the handler
propagates it (no response body); the error is translated to an HTTP status by
the shared [apperror](../../../apperror/README.md) mapping (e.g. a DB failure
surfaces as `503` via `apperror.ErrUnavailable`). On success the usecase reports
`status: ok`.

## Standard handler pattern

This is a permanent handler that follows the standard pattern documented in the
[parent handler guide](../README.md): a `server` struct built by `BindHandler`
through `gen.NewStrictHandler` / `gen.RegisterHandlers`, with a tracer span
wrapping the handler body and a single usecase call.

```go
func BindHandler(
    e *echo.Echo,
    tf observability.TracerFactory,
    healthUsecase healthcheckuc.Usecase,
)
```

- `healthUsecase healthcheckuc.Usecase` performs the actual readiness check.
- `tf observability.TracerFactory` yields the controller-layer `LayerTracer`.

Because the handler propagates the request context into the usecase, it re-binds
`ctx` from `ctx, endSpan := s.tracer.Start(ctx)`.

`BindHandler` is wired in the controller DI module
([`internal/di/module/controller.go`](../../../di/module/controller.go)) with
`fx.Invoke(ready.BindHandler)`.

## Response

`GetReady` returns `gen.ReadyResponse` (`GetReady200JSONResponse`):

| Field | Source |
| --- | --- |
| `Status` | usecase status (`ok` / `degraded` / `unhealthy`) |
| `ApplicationTime` | `clock.Now()` at check time |
| `DbLatencyMs` | DB health-check round-trip latency (ms) |
| `DbRespondedAt` | time the DB last responded successfully |

`Status` is the `ReadyResponseStatus` enum; its members (`ok`, `degraded`,
`unhealthy`) mirror the usecase's status constants.
