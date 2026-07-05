---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [observability, exclusion, setup-review]
---

# ADR-0061: Use only official OpenTelemetry semantic conventions; do not invent custom semconv or put vendor keys in typed config

## Status

accepted

## Context

OpenTelemetry publishes a versioned semantic conventions package (`semconv`) that defines
standard attribute names for service identity, resource attributes, and telemetry signals.
There is recurring pressure to add vendor-specific keys (e.g., Datadog tags, Grafana
labels) or project-specific attribute names directly to the typed `ObservabilityConfig`,
so that vendor routing or enrichment can be driven from the typed config layer. There is
similarly pressure to add non-standard resource attributes beyond the official schema when
the standard does not yet cover a desired label.

## Decision

We deliberately do **NOT** invent custom semantic convention keys or place vendor-specific
OTLP attribute keys in the typed `ObservabilityConfig`.

All resource attributes in `NewResource` use only the official semconv package
(`go.opentelemetry.io/otel/semconv/v1.37.0`): `semconv.ServiceName`,
`semconv.DeploymentEnvironmentName`, and `semconv.ServiceVersion`. The two build-time
identity fields that have no direct semconv counterpart (`service.revision`,
`service.build_date`) are expressed as plain `attribute.String` calls with stable,
non-vendor key names — not as vendor-proprietary keys. Vendor-specific enrichment and
routing live in the Collector, not in the application config.

The setup reviewer must confirm that any new resource attribute or config field introduced
during initial project setup still respects this boundary.

## Consequences

### Positive Consequences

- Resource attributes remain portable across any OTLP-compatible backend; no
  vendor-specific mapping is needed in application code.
- Typed config stays free of OTLP-specific or vendor-specific keys, keeping
  `ObservabilityConfig` readable and auditable.
- Upgrading the semconv version is a single import path change, not a search across a
  mixed custom/official key namespace.

### Negative Consequences

- When the official semconv does not yet cover a needed attribute, the team must either
  wait for the standard or accept a plain `attribute.String` key under a clearly
  namespaced, non-vendor name; convenience vendor keys are not available as a shortcut.

## Alternatives Considered

### Add vendor-specific keys to `ObservabilityConfig`

Rejected: it would embed backend routing decisions in the typed config layer, creating
coupling to a specific vendor and undermining the vendor-neutral stance of
[ADR-0060](0060-vendor-neutral-otlp-export.md). Vendor enrichment belongs in the
Collector pipeline.

### Invent project-local semconv-like keys

Rejected: inventing a parallel key namespace causes attribute name drift when the official
semconv eventually covers the same concept, and produces non-portable telemetry that
requires backend-specific remapping.

## Notes

- Complements: [ADR-0060](0060-vendor-neutral-otlp-export.md) (vendor-neutral export).
- Parent gating: [ADR-0059](0059-config-driven-observability-gating.md).
- Implementation evidence: `internal/observability/provider.go` line 23 (semconv import),
  lines 58–60 (`semconv.ServiceName` / `semconv.DeploymentEnvironmentName` /
  `semconv.ServiceVersion` in `NewResource`); `internal/observability/README.md` lines
  46–51 ("no OTLP-specific keys leak into the typed config").
- Setup reviewer action: when customizing the project scaffold, confirm that no new
  `ObservabilityConfig` field carries a vendor-specific OTLP key name.
