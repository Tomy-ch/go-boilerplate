---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [observability, metrics]
---

# ADR-0072: Metrics travel two paths — OTLP push and Prometheus scrape

## Status

accepted

## Context

The observability subsystem produces two categories of metrics with fundamentally
different collection shapes:

- **Operational subsystem metrics** (outbox lag, worker RED, idempotency results, DB
  spans, Go runtime) are event-driven: they accumulate as the application processes work
  and are best emitted on a push schedule by the SDK.
- **Process/broker identity metrics** (build info, broker queue depth) are pull-oriented:
  build identity is resolved once at wiring time and never changes; queue depth is best
  read on-demand per scrape rather than buffered and pushed periodically.

Routing both categories through the same OTLP push path would force pull-oriented metrics
into an awkward push pattern (polling in a background goroutine just to push them over
OTLP), or would require disabling OTLP metrics entirely to keep a Prometheus scrape
endpoint.

## Decision

Expose metrics via **two deliberate and independent paths**:

| Path | Instruments | How it leaves the process |
| --- | --- | --- |
| **OTLP push** | `outbox` / `worker` / `idempotency` / `httpclient` OTel meters + Go runtime + `otelpgx` DB metrics | `MeterProvider` `PeriodicReader` pushes to Collector; active only when `MetricsEnabled()` |
| **Prometheus scrape** | `app_build_info` (buildinfo), `worker_queue_depth` (queue) | registered to the default Prometheus registry, served at `/metrics` via `promhttp`; independent of `OBS_METRICS_EXPORTER` |

The scrape path is always present regardless of whether the OTLP signal is enabled. The
two paths coexist: a deployment may enable both, only the scrape endpoint, or only OTLP
push, depending on the monitoring stack.

## Consequences

### Positive Consequences

- Pull-oriented metrics (build identity, live queue depth) are collected efficiently on
  demand rather than pushed on a fixed interval.
- The scrape endpoint works without any Collector and without enabling `OBS_METRICS_EXPORTER`,
  making lightweight / local deployments still queryable.
- Each path is independently operated: the OTLP push path and the scrape path do not
  share state or introduce coupling between them.

### Negative Consequences

- Operators must be aware of both paths to get the full metric picture; a Prometheus
  scrape endpoint and an OTLP Collector are two distinct ingestion points to configure
  and monitor.
- Metrics that belong to one path cannot trivially be queried through the other without
  Collector-side bridging (e.g., Prometheus remote write).

## Alternatives Considered

### OTLP push only (bridge Prometheus collectors into the OTel MeterProvider)

Rejected: queue depth is a live pull (sampling the broker state per scrape is the
intended model for `worker.QueueStatsProvider`); buffering it into a push pipeline
introduces a stale-reading window and an extra background goroutine purely for
instrumentation reasons.

### Prometheus scrape only

Rejected: OTel meter instruments integrate natively with the SDK pipeline (batch export,
exemplars, resource attribution), which Prometheus instruments do not.

### Unified through a Prometheus-to-OTLP bridge in the Collector

Not adopted here: that is a Collector-side concern. The application does not prescribe
Collector configuration.

## Notes

- Source: `docs/design/observability.md` §3.2 "Two metric exit paths".
- Parent: [ADR-0069](0069-config-driven-observability-gating.md) (config-driven gating).
- Implementation: `internal/observability/metrics/buildinfo/` (buildinfo collector),
  `internal/observability/metrics/queue/` (queue depth collector),
  OTel meter instruments in `outbox_metrics.go`, `worker_metrics.go`,
  `idempotency_metrics.go`, `http_client_metrics.go`.
