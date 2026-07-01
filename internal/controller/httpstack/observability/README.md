# observability

English | [日本語](README.ja.md)

OpenTelemetry tracing middleware for Echo.

## Role

Distributed tracing must wrap every request uniformly to be useful, and threading span creation through each handler would be repetitive and error-prone. Isolating the trace entry point as a middleware starts a server span for every request in one place, so downstream layers inherit propagated trace context automatically and handlers stay free of instrumentation boilerplate.

## Notes

- `Middleware(serviceName)` returns the OTel (`otelecho`) tracing middleware; `PassthroughMiddleware()` returns a no-op middleware that simply forwards to the next handler. The DI layer selects the passthrough when tracing is disabled (`ObservabilityConfig.TracesEnabled()` is false), so the middleware slot is always filled without a conditional at the registration site.
