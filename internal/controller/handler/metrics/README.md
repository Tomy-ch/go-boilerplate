# Metrics Handler (`internal/controller/handler/metrics`)

English | [日本語](README.ja.md)

## Role

`metrics` exposes the Prometheus scrape endpoint **`GET /metrics`**.

It is an **operational endpoint that is not defined in OpenAPI**, so it is the
deliberate exception to the standard handler pattern documented in the
[parent handler guide](../README.md). It has **no `gen/` package, no
`ServerInterface`, no `server` struct, no `gen.NewStrictHandler`, and no Usecase
call** — it simply mounts the Prometheus client's own HTTP handler on Echo.

## Implementation

```go
func BindHandler(e *echo.Echo, bav echomw.BasicAuthValidator) {
    e.GET("/metrics",
        echo.WrapHandler(promhttp.Handler()),
        echomw.BasicAuth(bav),
    )
}
```

- The route is registered directly as an `echo.HandlerFunc` via
  `echo.WrapHandler(promhttp.Handler())` — not through `gen.RegisterHandlers`.
- `BindHandler` is wired in the controller DI module
  ([`internal/di/module/controller.go`](../../../di/module/controller.go)) with
  `fx.Invoke(metrics.BindHandler)`, the same registration mechanism as the
  standard handlers.

## What is exposed

`promhttp.Handler()` serves the Prometheus **default registry**
(`prometheus.DefaultGatherer` / `prometheus.DefaultRegisterer`). This handler
package registers nothing itself; the metrics come from collectors that other
packages register on that default registry at startup, including:

- the client library's built-in **Go runtime and process collectors**;
- **`app_build_info`** — the build/version info gauge registered by
  [`internal/observability/metrics/buildinfo`](../../../observability/metrics/buildinfo)
  (`buildinfo.Register` → `prometheus.DefaultRegisterer`);
- **RDB pool / query metrics** from
  [`internal/infrastructure/rdb/metrics`](../../../infrastructure/rdb/metrics)
  (its `NewRegisterer` returns `prometheus.DefaultRegisterer`);
- **HTTP RED metrics** from the instrumentation middleware, registered against
  the same `prometheus.Registerer` provided by the DB module (see
  [instrumentation](../../../di/server/extension/instrumentation/README.md));
- the **worker queue stats collector**
  ([`internal/observability/metrics/queue`](../../../observability/metrics/queue),
  `RegisterStatsCollector`).

> OpenTelemetry metrics (`OBS_METRICS_EXPORTER`) are **pushed over OTLP**, not
> scraped here; this endpoint only serves the Prometheus-native default registry.

## Access control

The endpoint is protected with Echo's Basic auth middleware
(`echomw.BasicAuth(bav)`). The `BasicAuthValidator` is provided by DI from
`internal/controller/httpstack/basicauth` (driven by `config.MetricsConfig`).

## Difference from the standard handler pattern

| Standard OpenAPI handler | This endpoint |
| --- | --- |
| Route from `gen.RegisterHandlers` | Direct `e.GET("/metrics", …)` |
| `server` struct + `gen.NewStrictHandler` | Plain `echo.WrapHandler` |
| Parses request → calls Usecase → Presenter | No Usecase, no DTO conversion |
| `observability.LayerTracer` span | No span (nothing to trace) |

This carve-out is limited to non-OpenAPI operational endpoints; feature APIs must
follow the standard pattern in the [parent handler guide](../README.md).
