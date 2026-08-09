---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [observability]
---

# ADR-0069: Vendor-neutral OTLP-only export (delegate backend to the Collector)

## Status

accepted

## Context

The observability subsystem must ship traces, metrics, and logs to a monitoring backend
without coupling the application binary to any specific vendor's SDK or proprietary
endpoint. The same service is deployed to environments with different backends (Grafana,
Datadog, New Relic, and others), and the binary must remain neutral across those choices. Embedding vendor-specific exporters in the application binary would require
code changes when switching providers, contradicting the lock-in avoidance principle in
[ADR-0001](0001-avoid-lock-in.md).

## Decision

Wire only the **vendor-neutral OTLP plumbing**. All three signals — traces, metrics, and
logs — use OTLP as the sole export transport (`http/protobuf` by default, `grpc`
optional). Vendor-specific routing, pipeline transformation, and backend authentication
live entirely in a Collector or Agent sidecar, never in the application code or typed
config.

The single `OBS_OTLP_ENDPOINT` is shared across all three signals; for HTTP the
per-signal path (`/v1/traces`, `/v1/metrics`, `/v1/logs`) is appended automatically when
the URL carries no path. No vendor SDK is imported; the only exporter dependency is the
OpenTelemetry OTLP exporter packages.

This decision is gated by [ADR-0068](0068-config-driven-observability-gating.md): a
signal that is disabled (empty or `none` exporter value) builds no OTLP exporter at all.

## Consequences

### Positive Consequences

- Any OTLP-compatible backend is reachable by pointing `OBS_OTLP_ENDPOINT` at a
  Collector without changing application code.
- No vendor SDK is compiled into the binary; dependency surface stays auditable and
  replaceable.
- Backend-specific concerns (authentication, pipeline routing, sampling rules) are
  handled by the Collector, separating operational concerns from the application.

### Negative Consequences

- Requires a Collector or Agent sidecar in every environment that enables export.
  Direct-to-vendor export without a Collector is not supported by this design.
- Two transport options (`http/protobuf` and `grpc`) must be maintained; misconfigured
  protocol produces a startup-time error at provider construction (`errInvalidOTLPProtocol`).

## Alternatives Considered

### Vendor-native exporters (e.g., Datadog agent SDK, New Relic SDK)

Rejected: importing a vendor SDK couples the binary to one provider's API and authentication
model. Switching backends would require code and dependency changes in the application
itself, violating [ADR-0001](0001-avoid-lock-in.md).

### OTel autoexport (reading `OTEL_*` env vars directly)

Rejected: autoexport reads the environment outside the typed config system and would
create a second control plane. [ADR-0068](0068-config-driven-observability-gating.md)
records this rejection in detail.

### Console exporter for local development

Rejected: the no-op fallback (disabled signal builds no exporter, no goroutine) is a
sufficient local development mode. A console exporter would be a third code path for
negligible benefit.

## Notes

- Parent principle: [ADR-0001](0001-avoid-lock-in.md) (lock-in avoidance).
- Config-driven gating: [ADR-0068](0068-config-driven-observability-gating.md).
- Source: `docs/design/observability.md` §1 "Role theory", `internal/observability/README.md`
  "Configuration boundary".
- Implementation: `internal/observability/provider.go` (OTLP-only exporter construction
  in `newSpanExporter` / `newMetricExporter`).
