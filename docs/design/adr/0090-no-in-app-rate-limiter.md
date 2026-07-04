---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [exclusion, setup-review]
---

# ADR-0090: Do not provide an in-application rate limiter

## Status

accepted

## Context

Rate limiting is a standard production-hardening concern, and it is natural to reach for an
in-process, in-memory counter when adding it to an API. However, this template targets
cloud-native, multi-instance deployments. In that environment, each application instance
maintains its own counter independently: counters are not shared across replicas, so an
attacker distributing requests across pods bypasses any per-instance threshold. An in-memory
limiter would therefore give a false sense of protection while adding complexity.

The pressure to include one is real — frameworks and libraries (e.g. Echo middleware,
`golang.org/x/time/rate`) make it trivial to add a limiter in a few lines, and it appears to
work correctly in single-instance or development environments, which is where it is typically
tested.

## Decision

We deliberately do NOT provide an in-application rate limiter. Rate limiting is delegated to
the infrastructure edge — the API gateway, load balancer, reverse proxy, or service mesh that
sits in front of the fleet. These components enforce limits against the true global request
rate, regardless of how many application instances are running.

Adopters who require rate limiting must configure it at the infrastructure layer appropriate
to their deployment (e.g. AWS API Gateway usage plans, NGINX `limit_req`, Envoy
`ratelimit` filter, Kubernetes Gateway API).

## Consequences

### Positive Consequences

- No false security from a per-instance counter that can be bypassed by distributing load.
- No in-process lock contention or memory overhead from a shared limiter data structure.
- Rate-limiting policy lives in infrastructure configuration, where it can be tuned without
  deploying application code.

### Negative Consequences

- Adopters must configure rate limiting at the infrastructure layer; there is no fallback if
  none is configured.
- Local development and testing do not exercise rate-limiting behavior unless the edge
  component is included in the dev environment.

## Alternatives Considered

### In-memory token bucket / sliding window (e.g. `golang.org/x/time/rate`)

Works correctly for a single instance but fails silently in a multi-replica deployment —
each pod gets its own independent bucket, making the effective limit `n × per-instance-limit`
where `n` is the replica count. Rejected because it provides incorrect enforcement at scale.

### Distributed in-app limiter backed by Redis or similar

Shares state across replicas, but requires an additional infrastructure dependency
(a Redis cluster), couples the application to that specific technology, and replicates
functionality already available at the edge layer. Rejected as over-engineering for a
template; the edge layer is the appropriate place.

## Notes

- Source: [`docs/project/out-of-scope.md`](../../project/out-of-scope.md) lines 11–16.
- Full ADR set and ordering: [`../adr-migration-plan.md`](../adr-migration-plan.md).
