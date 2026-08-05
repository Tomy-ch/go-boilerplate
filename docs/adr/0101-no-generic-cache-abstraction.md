---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [exclusion, setup-review]
---

# ADR-0101: Do not provide a generic Cache abstraction

## Status

accepted

## Context

Caching is a common performance optimization, and it is natural to expect a template to
supply a `Cache` interface analogous to the domain `Repository` interfaces. Such an interface
would appear to support the lock-in avoidance principle from [ADR-0001](0001-avoid-lock-in.md)
by hiding the cache technology behind a seam.

However, a neutral `Cache` interface degrades to a lowest-common-denominator contract — in
practice, a TTL-backed get/set map. That contract:

- Leaks implementation semantics: a TTL that is meaningful for an in-process map is a
  different concept from a Redis key expiry or a Memcached slab eviction.
- Discards technology-specific capabilities: Redis pipelines, Lua scripts, pub/sub channels,
  and sorted-set operations cannot be expressed through a generic interface without exposing
  them as extension methods that break the abstraction anyway.

This is precisely the lock-in trade-off examined in [ADR-0001](0001-avoid-lock-in.md): a
neutral seam can cost technology-specific capabilities. In the case of a cache, the cost is
high enough that the seam does more harm than good.

The domain `Repository` interface already provides the correct seam. Caching is a
read-path optimization over repository access, not a separate domain concept.

## Decision

We deliberately do NOT provide a generic `Cache` interface or cache abstraction layer. When
caching is needed, adopters implement it as a **decorator** that satisfies the existing
domain `Repository` interface. Domain and usecase layers remain unaware of caching because
the decorator is wired at the infrastructure / dependency-injection layer — domain code
calls the same repository method; the decorator decides whether to serve from cache or
delegate to the real repository.

The `Repository` interface is the swap seam; no additional abstraction is required.

**The seam is whichever inner-layer interface already exists — not the `Repository`
specifically.** The paragraph above names `Repository` because it is the commonest cached seam,
but the principle is about the shape: cache by decorating an interface an inner layer already
declares, never by introducing a generic cache abstraction beside it. It therefore applies
unchanged to an outbound `usecase/boundary` port — a decorator satisfying that `Gateway`
interface, wired at the infrastructure / DI layer, leaves usecase and domain exactly as unaware
of caching as the `Repository` case does, and the wall-clock dependency a TTL introduces stays
inside the infrastructure decorator instead of reaching an inner layer. The subject differs; the
seam principle is identical. What a given decorator's TTL is, and what staleness that implies for
callers, is feature content and belongs with the feature.

## Consequences

### Positive Consequences

- Domain and usecase layers stay unaware of caching — no cache-specific types leak inward.
- Technology-specific capabilities (pipelines, Lua, pub/sub) are available in the decorator
  without compromising the inner-layer interface.
- Adding a cache decorator is an infrastructure concern, consistent with the onion model
  (see [ADR-0002](0002-onion-architecture.md)).
- No lowest-common-denominator abstraction that silently discards cache-backend features.

### Negative Consequences

- Adopters must write the decorator themselves; no off-the-shelf generic cache helper is
  provided.
- Each cached repository requires its own decorator, which increases per-repository
  boilerplate compared to a single shared cache wrapper.
- Without a common interface, tooling or instrumentation that operates on "all caches" must
  be implemented individually per decorator.

## Alternatives Considered

### Generic `Cache[K, V]` interface with TTL get/set

Provides a consistent seam and makes caching testable via a stub. Rejected because the
interface collapses to a TTL-backed map that cannot express the capabilities of modern cache
backends (pipelines, atomics, pub/sub, sorted sets). Adopters who need those capabilities
would have to add backend-specific methods, breaking the interface and negating its value.
See [ADR-0001](0001-avoid-lock-in.md) on the cost of neutral seams.

### Technology-specific cache client injected directly into usecase

Would expose Redis or Memcached types to the usecase layer, violating the dependency rule
(usecase must not depend on infrastructure). Rejected by the onion architecture constraints
([ADR-0002](0002-onion-architecture.md)).

## Notes

- Source: [`docs/project/out-of-scope.md`](../project/out-of-scope.md) lines 49–57.
- Related: [ADR-0001](0001-avoid-lock-in.md) (lock-in avoidance — neutral seams can cost
  technology-specific capabilities; this is a deliberate exception).
- Related: [ADR-0002](0002-onion-architecture.md) (onion architecture — cache decorator
  wired at the infrastructure layer).
- Full ADR set and ordering: [the ADR log](README.md).
