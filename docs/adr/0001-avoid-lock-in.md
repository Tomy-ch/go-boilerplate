---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [foundational, architecture, dependencies]
---

# ADR-0001: Adopt lock-in avoidance as a design principle

## Status

accepted

## Context

This is a long-lived **template** repository whose stated design goals include replaceable
infrastructure and long-term operability. Two forms of captivity threaten those goals:

- **Vendor lock-in** — binding the application to a proprietary SaaS (a specific APM,
  managed queue, or cloud primitive) so that switching provider means rewriting code.
- **Library lock-in** — a dependency that spreads a framework's types and idioms across
  layers so it can no longer be swapped without a wide blast radius.

Downstream users fork this template into environments whose vendors we cannot predict, so
captivity is a concrete, not theoretical, risk.

## Decision

Treat **lock-in avoidance (replaceability)** as a first-order design principle that the
more specific decisions inherit:

- Prefer vendor-neutral, OSS, standards-based components; keep vendor specifics behind a
  seam (domain `Repository` interfaces, `usecase/boundary` ports, OTLP + Collector for
  observability) rather than in inner layers.
- Every third-party library maps to a single, nameable, replaceable responsibility; a
  library that would straddle two upstreams is treated as an explicit, bounded exception.
- **Wrap the vendor behind `pkg/`.** Where a library supplies a general-purpose *value type*
  that would otherwise spread through every layer, the concrete vendor is wrapped by a thin
  `pkg/` package and the rest of the repository depends only on that wrapper (`pkg/uuid`,
  `pkg/xerrors`, `pkg/decimal`). A direct import of the wrapped vendor from anywhere other than
  its seam is forbidden, which is checkable by grep and by depguard, so the library can be
  swapped by editing one package instead of the whole tree. The wrapper carries no business
  semantics — that is what keeps it admissible in `pkg/` at all — and where the type crosses the
  database or the HTTP boundary, the wrapper owns those conversions too, so even generated code
  never names the vendor type. Applying this pattern to a particular library is an instance of
  this principle and does not warrant an ADR of its own.

This principle is the parent of the library-selection policy, the SQS opt-in isolation, the
OTLP-only export, and the vendor-neutral deploy skeleton (each recorded as its own ADR).

## Consequences

### Positive Consequences

- Infrastructure and providers are swappable behind stable interfaces; no SaaS captivity.
- The dependency surface stays auditable — each library has one replaceable job.
- Forks can retarget cloud, broker, or telemetry backends without touching domain/usecase.

### Negative Consequences

- Neutral seams can cost technology-specific capabilities (the trade-off examined per case,
  e.g. the deliberately rejected generic cache abstraction).
- More indirection than binding directly to a vendor SDK.

## Alternatives Considered

### Bind directly to a chosen vendor / SaaS

Rejected: fastest initially, but couples business code to one provider's API and pricing,
defeating the template's replaceable-infrastructure goal.

### No explicit principle (decide per case)

Rejected: without a stated parent principle, lock-in creeps in incrementally through
convenient SDK imports and framework types leaking across layers.

## Notes

- Sources: `docs/architecture.md` (vendor-neutral / OSS-replaceable), `docs/project/policy.md`
  (vendor neutrality), `docs/project/policy.md` § "Library Selection Policy".
- Enforced in part by the layer/`pkg` import rules in [`docs/rules.md`](../rules.md).
- Full ADR set and ordering: [the ADR log](README.md).
