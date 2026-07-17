---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [observability, config]
---

# ADR-0062: Config-driven observability gating

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
  the three exporter settings is a non-empty, non-`none` value. Deriving from the exporter
  settings is intentional: a bare enable flag paired with a dead exporter is meaningless;
  this design forces operators to be conscious of export configuration.
- Gating is applied at **construction time**: a disabled signal builds no exporter / batcher
  / reader / runtime collector (no network, no goroutines), the Echo otelecho middleware
  degrades to pass-through, and the otelzap log core is not Tee'd into the logger. The SDK
  provider shells remain (cheap, inert) — this is runtime disabling, not build-time removal.
- The same config-driven gate also governs **per-log trace correlation**. `trace_id` /
  `span_id` are injected by the `Logger` itself (ctx-native API — `Info(ctx, msg, ...)`),
  never appended by callers, on each log line whose `ctx` carries a valid span. The gate —
  observability enabled **and** the ctx carries a valid span — is consolidated in a single
  `observability.NewTraceExtractor(obsCfg)` closure that is DI-injected into the `Logger` as a
  `logging.TraceExtractor`. `logging` depends only on that abstraction and never imports
  `observability`, so the gate stays config-driven without inverting the inward-only
  dependency direction (see [ADR-0003](0003-interface-based-decoupling.md)).

## Consequences

### Positive Consequences

- One typed source of truth, consistent with every other subsystem; no second control plane.
- Lightweight environments pay no observability cost (no exporters, readers, collector, or
  per-request spans).
- Portability preserved: any OTLP backend is still just `OBS_*_EXPORTER=otlp` + an endpoint.
- The `trace_id` / `span_id` gate lives in one place (the injected extractor); callers pass
  `ctx`, not `trace_id` / `span_id`. The one exception is `parent_span_id`, which cannot be
  derived from `ctx` and is therefore gated directly by `obsCfg.Enabled()` in
  `BuildSQLEndFields`.

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

- Vendor-neutral OTLP-only export and the official-semconv stance are recorded separately (see the observability ADRs in [the ADR log](README.md)).
- Design reference: `docs/design/observability.md`. The ctx-native `Logger` and the injected `TraceExtractor` that carries the per-log trace gate are described in `internal/logging/README.md`.
- Migrated from `docs/decisions.md` (§ "Why config-driven observability gating").
