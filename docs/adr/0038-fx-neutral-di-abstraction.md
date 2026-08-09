---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [di, architecture]
---

# ADR-0038: Contain fx behind a neutral DI abstraction (Registrar / Shutdowner)

## Status

accepted

## Context

[ADR-0037](0037-uber-fx-di.md) adopts Uber Fx as the DI and lifecycle container.
Without an explicit containment boundary, `fx.Lifecycle` and `fx.Shutdowner` spread
across all layers: the HTTP server, observability components, workers, and the outbox
relay would all import `go.uber.org/fx` directly. This couples inner layers to the
framework, violates the onion architecture dependency rule (see
[ADR-0002](0002-onion-architecture.md)), and makes unit tests cumbersome because Fx
types require a running container.

The same replaceability principle that governs third-party library selection
([ADR-0001](0001-avoid-lock-in.md)) applies to the DI framework itself: fx should be
swappable without touching components that merely need to register start/stop hooks or
trigger a graceful shutdown.

## Decision

Define two narrow, neutral interfaces in dedicated packages within `internal/di/`:

- **`lifecycle.Registrar`** — wraps `fx.Lifecycle`, centralizing Start/Stop hook
  registration. All components (HTTP server, TracerProvider, workers, DB connection,
  etc.) receive `Registrar` rather than `fx.Lifecycle`.
- **`shutdowner.Shutdowner`** — wraps `fx.Shutdowner`, providing a typed `Shutdown()`
  method. Application code that needs to trigger a programmatic stop depends on this
  interface, not on `fx.Shutdowner` directly.

Both wrappers are implemented in the DI layer; the fx dependency is confined there.
Inner layers reference only the interface types.

`lifecycle` also provides `SupervisedRunner`, a shared primitive that standardizes the
"start a background goroutine, cancel it on stop, wait for drain" pattern used by
workers, jobs, and the outbox relay — removing duplicated Start/Stop plumbing across
those hooks.

## Consequences

### Positive Consequences

- The fx dependency is confined to the DI layer; no inner layer imports
  `go.uber.org/fx` for lifecycle or shutdown purposes.
- Tests can inject a no-op `Registrar` stub without standing up an `fx.App`.
- `SupervisedRunner` eliminates copy-pasted Start/Stop goroutine plumbing.
- Swapping the DI framework requires changing only the DI layer implementations, not
  every component that registers hooks.

### Negative Consequences

- Two additional thin wrapper packages exist solely to contain a framework type; the
  indirection is real even if the implementation is trivial.
- Lifecycle hook execution in integration tests still requires starting and stopping an
  `fx.App` (the abstraction does not remove that need when hooks must fire).

## Alternatives Considered

### Use fx.Lifecycle and fx.Shutdowner directly

Each component imports `go.uber.org/fx` for lifecycle registration and shutdown
signalling. Rejected: fx types propagate into every layer, violating the containment
principle and making unit tests dependent on the container.

### Single global hook registry (not DI-injected)

A package-level registry avoids the fx import but introduces global mutable state,
complicating parallel tests and making hook order implicit. Rejected in favor of an
explicit, injected interface.

## Notes

- Parent decision: [ADR-0037](0037-uber-fx-di.md) (adopting Uber Fx).
- Lock-in avoidance principle: [ADR-0001](0001-avoid-lock-in.md).
- Source: `internal/di/lifecycle/README.md`, `internal/di/shutdowner/README.md`.
