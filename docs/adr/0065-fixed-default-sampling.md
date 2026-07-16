---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [observability, exclusion, setup-review]
---

# ADR-0065: Fix the SDK default sampling; do not expose sampling as an env knob

## Status

accepted

## Context

The OTel SDK allows the sampler to be replaced at provider-construction time, and
operators of high-volume services commonly request a configurable sampling rate (e.g.,
`OBS_SAMPLE_RATE=0.1`) to reduce trace volume and Collector load in production. Adding
such a knob appears straightforward and is a recurring infrastructure request.

However, head-based probabilistic sampling — the most common form of configurable sampling
— produces incomplete trace populations (some requests are recorded, others are not),
which complicates latency analysis and incident investigation. It also introduces an
operational footgun: setting sampling to 0 in an incident scenario silently discards all
traces.

## Decision

We deliberately do **NOT** expose sampling as an environment variable or typed-config
field.

The sampler is fixed at the SDK default — `ParentBased(AlwaysSample)` — and is not
currently env-configurable. If sampling reduction is required in a high-volume deployment,
it should be delegated to the Collector (tail-based sampling, probabilistic sampling
processor), where the full trace is available before the sampling decision is made.

The setup reviewer must confirm that the initial project customization does not introduce
an `OBS_SAMPLE_RATE` or equivalent field into `ObservabilityConfig`.

## Consequences

### Positive Consequences

- All traces are complete by default; no sampling gaps introduce blind spots in latency
  histograms or error attribution.
- Collector-side tail-based sampling can make decisions on the complete span tree, which
  is more accurate than head-based sampling in the SDK.
- Operational surface of `ObservabilityConfig` stays minimal; no sampling footgun.

### Negative Consequences

- Without Collector-side sampling, high-traffic deployments will emit all spans over OTLP,
  increasing Collector and storage load. The adopter must add Collector-side sampling
  when needed.
- The sampling strategy cannot be changed without deploying a Collector configuration
  change (or a code change); there is no runtime knob in the application.

## Alternatives Considered

### Head-based configurable sampling via `OBS_SAMPLE_RATE`

Rejected: probabilistic head sampling produces incomplete trace populations that mislead
latency analysis. Setting the rate to zero in an incident silently discards all diagnostic
data. Tail-based sampling in the Collector is the preferred alternative.

### Ratio-based sampler hardcoded to a value other than 1.0

Rejected for the same reason as configurable head sampling: incomplete traces are less
useful than complete traces, and the trade-off is better made at the Collector level where
the full span tree is visible.

## Notes

- Source: `docs/design/observability.md` §2.2, line 80: "Sampling is the SDK default
  `ParentBased(AlwaysSample)`; it is **not** currently env-configurable."
- Implementation confirmation: `internal/observability/README.md` §1 `NewTracerProvider`
  ("Uses the SDK default sampler (`ParentBased(AlwaysSample)`); sampling is not currently
  env-configurable").
- Parent: [ADR-0060](0060-config-driven-observability-gating.md).
- Setup reviewer action: confirm that no `ObservabilityConfig` field for sample rate is
  added during project initialization.
