---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [foundational, architecture, dependencies]
---

# ADR-0003: Define boundaries with interfaces for loose coupling (DIP)

## Status

accepted

## Context

[ADR-0002](0002-onion-architecture.md) defines the layer shape — dependencies point inward
and infrastructure cannot be imported by domain or usecase. The mechanism that makes this
rule enforceable and testable in practice is the Dependency Inversion Principle (DIP): every
cross-layer boundary is expressed as a Go interface **owned by the inner layer**, and the
outer layer provides the concrete implementation.

Without this constraint, outer layers would depend on inner-layer concrete types, and the
inner layers would need to be aware of the outer layers — either direction introduces
coupling that makes infrastructure replacement and isolated unit testing impractical.

The forces driving this decision:

- Domain code must be testable without a database or external system present.
- Infrastructure (database, external APIs) must be replaceable without touching business logic
  (consistent with [ADR-0001](0001-avoid-lock-in.md)).
- Usecase must orchestrate domain behavior without knowing which infrastructure adapter
  satisfies each Repository or Boundary interface.
- DI resolution (see ADR-0039) wires implementations to interfaces at startup, so the
  cross-layer seam must be an interface for the DI container to inject.
- Defining interfaces as cross-layer contracts allows each layer to own its processing
  logic independently, unbound from the implementation details behind the interface.
  Without interface definitions — coupling layers through concrete types — callers must
  track callee internals, reducing refactorability across the boundary.

## Decision

Define every cross-layer boundary as a Go interface owned by the inner layer; outer layers
implement those interfaces and inject them via DI.

Concretely:

- **Repository interfaces** are declared in the domain layer; infrastructure provides the
  implementations.
- **Boundary interfaces** (e.g. clock, auth, outbox, idempotency, tx) are declared in
  `internal/usecase/boundary`; infrastructure provides the implementations.
- Usecase depends on domain interfaces and boundary interfaces — never on infrastructure
  packages directly.
- Controller depends on usecase interfaces — never on domain entities or infrastructure types.

## Consequences

### Positive Consequences

- Domain and usecase layers are unit-testable with mock implementations; no real database
  or HTTP client is required.
- Infrastructure adapters (database engine, queue broker, external API client) are swappable
  without modifying business logic.
- The interface boundary is the natural seam for depguard enforcement: a forbidden import
  (e.g., usecase importing infrastructure) fails CI.

### Negative Consequences

- Every cross-layer concern requires an interface declaration plus at least one concrete
  implementation, increasing file count.
- Mapping between layer-local types (domain entity to DTO, DTO to OpenAPI response) adds
  repetitive conversion code at each boundary crossing.

## Alternatives Considered

### Direct concrete-type coupling

Outer layers import inner-layer concrete structs directly. Simpler initially, but test
isolation becomes impractical and infrastructure changes ripple across layers.

### Generic / type-parameter seams

Go generics could express some boundaries without explicit interface types. Rejected: adds
complexity without meaningful benefit for the Repository and Boundary use cases targeted here.

## Notes

- Layer dependency rules and the forbidden import table: [`docs/rules.md`](../rules.md)
  §§ "Layer Dependency Rules", "Usecase Dependency Rules", "Infrastructure Implementation
  Rules".
- The DI container that wires implementations to interfaces at startup: ADR-0039.
- Related layer shape: [ADR-0002](0002-onion-architecture.md).
- Source: `docs/architecture.md` § "Dependency Inversion"; `docs/rules.md` §§ "Layer
  Dependency Rules", "Usecase Dependency Rules", "Infrastructure Implementation Rules".
