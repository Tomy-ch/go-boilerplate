# instrumentation

English | [日本語](README.ja.md)

`instrumentation` is a directory that groups **DI middleware modules for HTTP layer observability and request identification**.

It provides the foundation for Tracing / Logging / Metrics through **request identifier generation** and **trace integration middleware**.

## Modules

|Module|Type|Priority|Description|
|---|---|---|---|
|`RequestIDModule()`|Use|1|Generate unique Request ID per request|
|`ObservabilityModule()`|Use|2|OpenTelemetry tracing integration|
|`LoggingModule()`|Use|8|Structured HTTP request/response logging|
|`HTTPRedMetricsModule()`|Use|9|HTTP RED metrics (request count / duration / status) via a Prometheus recorder|

## Priority Order

RequestID (Priority 1) → Observability (Priority 2) ensures **ID assignment occurs before trace start**.

HTTPRedMetrics (Priority 9) runs after Observability so a trace context already exists, and is placed after Logging (8) / before Cookie (10). Priority 7 is **not** used because `forceJSON` (in `outbound`) already occupies it, and duplicate Use priorities are rejected at apply time.

## Notes

- RequestID / Observability / HTTPRedMetrics are applied as **UseMiddleware with Priority**
- Observability depends on `ApplicationConfig` — **behavior may differ between production and non-production**
- Observability responsibility stays within the controller layer — **must not leak into domain/usecase**
- `HTTPRedMetricsModule()` registers the recorder against the `prometheus.Registerer` provided by the DB module, so the metrics are exposed on the same `/metrics` endpoint; ops paths (`/metrics`, `/health`, etc.) are excluded from measurement
- When adding middleware or changing priorities, watch for Priority conflicts with other UseMiddleware
