---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [di, lifecycle]
---

# ADR-0030: Adopt Uber Fx for dependency injection and lifecycle

## Status

accepted

## Context

The application wires many components across layers and needs structured dependency
resolution plus managed application lifecycle (ordered start-up and graceful shutdown).
Manual wiring works at small scale but becomes hard to manage as the component graph grows,
and a compile-time-only DI tool would not cover runtime lifecycle.

## Decision

Adopt **Uber Fx** as the dependency injection container and application lifecycle manager.

## Consequences

### Positive Consequences

- Explicit dependency wiring organized into modules.
- Application lifecycle management (ordered start/stop hooks).
- A clear, modular structure for growing the component graph.

### Negative Consequences

- A DI framework dependency at the composition root; its reflective wiring is a learning
  cost and is kept contained behind neutral DI abstractions (see the fx-containment ADR).

## Alternatives Considered

### Manual DI

Rejected: effective for small systems, but becomes difficult to manage as the dependency
graph grows.

### Google Wire

Rejected: provides compile-time DI, but does not offer runtime application lifecycle
management, which this application needs.

## Notes

- fx is confined behind neutral DI abstractions (Registrar / Shutdowner) — recorded separately (see the DI ADRs in [`../adr-migration-plan.md`](../adr-migration-plan.md)).
- Migrated from `docs/decisions.md` (§ "Why Fx").
