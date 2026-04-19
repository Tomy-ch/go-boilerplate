# instrumentation

English | [日本語](README.ja.md)

`instrumentation` is a directory that groups **DI middleware modules for HTTP layer observability and request identification**.

It provides the foundation for Tracing / Logging / Metrics through **request identifier generation** and **trace integration middleware**.

## Modules

|Module|Type|Priority|Description|
|---|---|---|---|
|`RequestIDModule()`|Use|1|Generate unique Request ID per request|
|`LoggingModule()`|Use|—|Structured HTTP request/response logging|
|`ObservabilityModule()`|Use|2|OpenTelemetry tracing integration|

## Priority Order

RequestID (Priority 1) → Observability (Priority 2) ensures **ID assignment occurs before trace start**.

## Notes

- RequestID and Observability are applied as **UseMiddleware with Priority**
- Observability depends on `ApplicationConfig` — **behavior may differ between production and non-production**
- Observability responsibility stays within the controller layer — **must not leak into domain/usecase**
- When adding middleware or changing priorities, watch for Priority conflicts with other UseMiddleware
