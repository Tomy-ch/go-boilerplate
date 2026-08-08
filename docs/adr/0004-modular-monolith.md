---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [foundational, architecture]
---

# ADR-0004: Adopt a modular monolith (microservices are a non-goal)

## Status

accepted

## Context

This repository targets backend services for business systems expected to be operated and
evolved over a long period — from PoC to early-scaling phases — by teams with Tech
Lead-level technical judgment. The primary design goals are maintainability, structural
safety, and long-term operability, not independent scalability of individual capabilities.

Two deployment philosophies were considered:

1. **Modular monolith** — a single deployable binary with clear internal module boundaries
   enforced by layer rules.
2. **Microservices** — multiple independently deployable services, each owning its own data
   store and domain.

Microservices introduce significant operational complexity: distributed tracing, network
failure modes, cross-service transactions, and independent deployment pipelines. For the
class of systems targeted here, that complexity is premature. At the same time, the
architecture should not become a trap: the strict layer separation and interface-defined
boundaries (see [ADR-0002](0002-onion-architecture.md) and
[ADR-0003](0003-interface-based-decoupling.md)) should make future service extraction a
deliberate and tractable refactor.

## Decision

Adopt a **modular monolith** as the deployment architecture. The system runs as a single
deployable application. Internal module boundaries are enforced by layer separation and
depguard rules. Microservices are explicitly a non-goal for this system.

## Consequences

### Positive Consequences

- Single deployment artifact simplifies CI/CD, observability, and local development.
- No distributed transaction complexity; transaction boundaries remain within a single
  process (Usecase layer owns them).
- Clear module boundaries at the Go package level make future service extraction a deliberate
  refactor rather than an accident.
- Suitable for the primary target: PoC-to-early-scaling backend services under long-term
  maintenance.

### Negative Consequences

- Not appropriate for systems designed as microservices from inception.
- Not appropriate for ultra-low-latency workloads where abstraction overhead must be
  minimized.
- A single deployable means all components scale together; fine-grained per-capability
  autoscaling requires architectural refactoring.

## Alternatives Considered

### Microservices from day one

Independent services per domain capability. Rejected for the primary target:
operational overhead is disproportionate for systems in the PoC-through-early-scaling scope.
A system that needs microservices from the outset should treat this structure as an
extraction source rather than a starting point.

### Flat monolith (no enforced module boundaries)

A single package or lightly structured package tree with no enforced layer rules. Rejected:
business logic, infrastructure, and transport become entangled over time, making maintenance
and future decomposition difficult.

## Notes

- Source: `docs/architecture.md` § "Modular Monolith Strategy" and § "Non-Goals".
- Source: `docs/project/scope.md` § "Architectural Assumptions" and § "Non-Target Use Cases".
- Related layer shape: [ADR-0002](0002-onion-architecture.md).
