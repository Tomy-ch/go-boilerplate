---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [observability, di]
---

# ADR-0073: Observability providers are lifecycle-independent (ProviderShutdowner)

## Status

accepted

## Context

The OTel SDK providers (`TracerProvider`, `MeterProvider`, `LoggerProvider`) each have a
`Shutdown` method that must be called on graceful exit to flush buffered spans/metrics/logs.
The natural registration point for this teardown is the DI lifecycle hook.

The straightforward approach — having the provider constructors register their own `OnStop`
hooks — would require `internal/observability` to import `internal/di/lifecycle`. This
reverses the intended dependency direction: `observability` is a substrate that the DI
layer wires; it must not depend on DI internals. An import of `di/lifecycle` from
`observability` would create a circular or layering-violating dependency.

## Decision

Provider constructors (`NewTracerProvider`, `NewMeterProvider`, `NewLoggerProvider`) are
**lifecycle-agnostic**. Each returns the concrete SDK provider type (e.g.
`*sdktrace.TracerProvider`, `*sdkmetric.MeterProvider`) so the concrete `Shutdown` method
is available to the caller, but the constructors do not register any shutdown hook
themselves.

The `ProviderShutdowner` type (in `shutdown.go`) provides an otel-agnostic shutdown handle
that the DI module composes from the concrete providers. The DI layer
(`hook.RegisterObservabilityShutdownHooks`, in `internal/di`) owns shutdown registration.
This keeps `internal/observability` free of any `internal/di/lifecycle` dependency.

## Consequences

### Positive Consequences

- `internal/observability` has no dependency on `internal/di`; the dependency direction
  remains clean (DI wires observability, never the reverse).
- The concrete provider types expose `Shutdown` directly to the DI hook without an
  additional adapter; no wrapper interface is needed in the provider package.
- Shutdown order is controlled entirely by the DI layer, which can sequence hook
  registration consistently with other subsystems.

### Negative Consequences

- Provider construction and shutdown registration are split across two packages
  (`observability` constructs; `di` registers), so understanding the full lifecycle
  requires reading both.
- The concrete provider types are exposed as return values rather than the narrower
  `trace.TracerProvider` / `metric.MeterProvider` interfaces; callers that only need the
  interface use the adapter functions `ProvideTracerProvider` / `ProvideMeterProvider`.

## Alternatives Considered

### Provider constructors register their own lifecycle hooks

Rejected: requires `internal/observability` to import `internal/di/lifecycle`, inverting
the dependency and coupling the observability substrate to the DI framework.

### A lifecycle interface injected into provider constructors

Rejected: would thread a DI-framework concern into the observability API signature,
coupling the constructor to the lifecycle abstraction while solving the import-cycle only
partially.

### Rely on process exit (no explicit Shutdown)

Rejected: without `Shutdown`, buffered spans and metrics are silently dropped on graceful
exit, producing incomplete traces and metric gaps.

## Notes

- Design invariant (source): `docs/design/observability.md` §1 "Role theory", line 30:
  "the providers are lifecycle-agnostic — they return the concrete SDK providers (which
  expose `Shutdown`) and let the DI hook own shutdown registration, so `observability`
  never imports `di/lifecycle`."
- Implementation: `internal/observability/shutdown.go` (`ProviderShutdowner`),
  `internal/observability/provider.go` (`NewTracerProvider`, `NewMeterProvider`,
  `ProvideTracerProvider`, `ProvideMeterProvider`),
  `internal/observability/log_provider.go` (`NewLoggerProvider`).
- Parent: [ADR-0069](0069-config-driven-observability-gating.md).
