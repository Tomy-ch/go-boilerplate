---
status: accepted
date: 2026-07-22
deciders: [maintainers]
tags: [architecture, http]
---

# ADR-0098: Cache the exchange-rate gateway with a TTL decorator on the boundary seam

## Status

accepted

## Context

`GET /v1/exchange-rates` fetches rates from an external service (Frankfurter, ECB-derived)
through the `boundary.Gateway` seam. The rate source updates at most daily, so repeating the
outbound HTTP call on every request is wasteful and couples request latency to an external
dependency. A backend cache is needed.

[ADR-0096](0096-no-generic-cache-abstraction.md) already rules out a generic `Cache`
abstraction and prescribes a **decorator** on an existing seam. Its decision text, however,
names the domain `Repository` interface as the swap seam. The exchange-rate cache target is
`internal/usecase/boundary/exchangerate.Gateway`, an outbound gateway boundary — not a
`Repository`. A decision is needed on (a) where the cache lives given ADR-0096's Repository
wording, and (b) the TTL value and the freshness tolerance it implies.

## Decision

We cache the exchange-rate gateway with an in-memory TTL **decorator that satisfies
`boundary.Gateway`**, placed in the infrastructure layer
(`internal/infrastructure/webapi/exchangerate/cache.go`) and wired in dependency injection
(`internal/di/module/webapi.go`), which wraps the raw gateway before it reaches the usecase.
Usecase and domain remain unaware of caching.

We explicitly read ADR-0096's decorator principle — *cache via a decorator on the existing
inner-layer seam, never a generic abstraction* — as applying to the `Gateway` boundary, not
only to `Repository`. The subject differs; the seam principle is identical.

The TTL is a fixed `const rateTTL = 24 * time.Hour`, not an environment/config value. Because
the source updates daily, 24h is a value whose rationale can be stated. Crossing the ECB
publication time (≈16:00 CET) may serve a rate up to ~24h old; this is acceptable because the
cached value feeds only a **non-persistent reference display**, and its `rateDate` is exposed
on the response so callers can judge freshness.

## Consequences

### Positive Consequences

- Usecase and domain stay unaware of caching; the time dependency of the TTL lives entirely
  in the infrastructure decorator (consistent with onion, [ADR-0002](0002-onion-architecture.md)).
- Consistent with [ADR-0096](0096-no-generic-cache-abstraction.md): no generic cache
  abstraction; the existing seam is reused.
- Removes per-request outbound HTTP calls for repeated currency pairs.

### Negative Consequences

- The Gateway-boundary read of ADR-0096 is an extension of its Repository-scoped wording; the
  reasoning is recorded here so the extension is explicit, not implicit.
- A fixed 24h TTL can serve a stale rate across the daily publication boundary; mitigated by
  exposing `rateDate` and by the reference being advisory only.

### Neutral Consequences

- Errors are not cached: a failed fetch re-hits the source on the next request, so a transient
  outage does not poison the cache.

## Alternatives Considered

### In-memory cache inside the usecase

Rejected: the usecase would depend on wall-clock time and mutable state, requiring a clock
abstraction and muddying the onion boundary. Caching is an infrastructure concern.

### A dedicated cache boundary interface

Rejected by [ADR-0096](0096-no-generic-cache-abstraction.md): it reintroduces the generic
cache abstraction that ADR explicitly excludes.

### Making the TTL a config value

Rejected: there is no environment-specific reason to vary it, which would violate the config
principle of not placing values whose per-environment rationale cannot be stated. If a
concrete operational need arises, the constant can become config with an ADR revision.

## Notes

- Related: [ADR-0096](0096-no-generic-cache-abstraction.md) (decorator seam principle, here
  read onto the `Gateway` boundary).
- Related: [ADR-0002](0002-onion-architecture.md) (decorator wired at the infrastructure layer).
- Rounding of the reference amount this cache feeds: [ADR-0099](0099-reference-amount-half-up-rounding.md).
