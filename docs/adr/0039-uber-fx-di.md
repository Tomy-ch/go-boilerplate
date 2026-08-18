---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [di, lifecycle]
---

# ADR-0039: Adopt Uber Fx for dependency injection and lifecycle

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
graph grows. Specific problems at this system's scale:

- The composition root grows proportionally with each added component; every new
  dependency requires explicit wiring, causing combinatorial growth.
- Multiple shared resources (DB pool, logger, observability) must be threaded as
  arguments through the entire wiring chain, making file-level decomposition
  impractical.
- A bloated composition root increases the likelihood of merge conflicts as multiple
  contributors touch the same wiring site.
- Manual DI suits small systems, but past a certain threshold readability degrades
  acceleratingly; it is rarely rewritten once entrenched and becomes permanent
  technical debt.

### Google Wire

Rejected: provides compile-time DI, but does not offer runtime application lifecycle
management, which this application needs.

## Notes

- fx is confined behind neutral DI abstractions (Registrar / Shutdowner) — recorded separately (see the DI ADRs in [the ADR log](README.md)).
- Migrated from the former `docs/decisions.md`.
