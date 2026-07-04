---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [observability, config]
---

# ADR-0058: Config-driven observability gating

## Status

accepted

## Context

Observability (traces / metrics / logs) must be switchable per environment so that
lightweight environments construct **no** OpenTelemetry providers, exporters, or
instrumentation bridges. A separate control plane read behind the app's back (the
spec-standard `OTEL_*` environment via autoexport) would create a second source of truth
disconnected from the project's typed config.

## Decision

Make observability a single, typed, **config-driven** switch:

- Exporter settings live in the typed `OBS_*` config subsystem (`OBS_TRACES_EXPORTER` /
  `OBS_METRICS_EXPORTER` / `OBS_LOGS_EXPORTER` / `OBS_OTLP_ENDPOINT` / `OBS_OTLP_PROTOCOL`),
  not in `OTEL_*` env read by autoexport.
- There is **no dedicated enable flag**. Observability is *derived* as enabled when any of
  the three exporter settings is a non-empty, non-`none` value.
- Gating is applied at **construction time**: a disabled signal builds no exporter / batcher
  / reader / runtime collector (no network, no goroutines), the Echo otelecho middleware
  degrades to pass-through, and the otelzap log core is not Tee'd into the logger. The SDK
  provider shells remain (cheap, inert) — this is runtime disabling, not build-time removal.

## Consequences

### Positive Consequences

- One typed source of truth, consistent with every other subsystem; no second control plane.
- Lightweight environments pay no observability cost (no exporters, readers, collector, or
  per-request spans).
- Portability preserved: any OTLP backend is still just `OBS_*_EXPORTER=otlp` + an endpoint.

### Negative Consequences

- "Enabled" is derived, not explicit; operators read the exporter settings to know the state
  (this is the deliberate replacement for the removed `OBSERVABILITY_ENABLED` flag).
- Instrumentation dependencies remain linked (disabled at runtime, not removed at build).

## Alternatives Considered

### Keep `OTEL_*` + autoexport

Rejected: exporter settings would not be represented in typed config, leaving a second
source of truth read directly from the environment.

### A dedicated `OBSERVABILITY_ENABLED` flag alongside exporter settings

Rejected: redundant with "is any exporter configured", and prone to conflicting states
(`ENABLED=true` with no exporter). This unifies the two previously disconnected control
planes.

### Build-tag removal of the otel / bridge dependencies

Rejected for now: runtime disabling meets the lightweight goal; build-time removal would add
two wiring variants for hot-path-wired instrumentation (otelecho / otelpgx) without a current
requirement.

## Notes

- Vendor-neutral OTLP-only export and the official-semconv stance are recorded separately (see the observability ADRs in [`../adr-migration-plan.md`](../adr-migration-plan.md)).
- Design reference: `docs/design/observability.md`.
- Migrated from `docs/decisions.md` (§ "Why config-driven observability gating").
