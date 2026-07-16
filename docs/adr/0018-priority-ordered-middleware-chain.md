---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [http, middleware]
---

# ADR-0018: Build the middleware chain as a priority-ordered, data-driven list

## Status

accepted

## Context

The inbound HTTP middleware chain must execute in a specific, invariant order — request-ID
injection must precede observability (so spans carry the ID), observability must precede
recovery (so panics are traced), authentication must precede handlers, and so on. The current
order is: uri-pre (1), requestID (1), observability (2), recovery (3), cors (4), security
(5), openapi (6), forcejson (7), httpredmetrics (8), logging (9), cookie (10).

The project uses [ADR-0032](0032-uber-fx-di.md) (Uber fx) for dependency injection. Each
middleware lives in its own `*_di.go` file under `internal/di/server/extension/`, meaning
there is no single location where the call sequence is written down. If each `*_di.go` called
`e.Use(...)` directly, the effective chain order would be an emergent consequence of Go module
initialisation and fx wiring order — neither explicit nor guaranteed.

Call-site ordering (`e.Use` in a single setup function) is an alternative, but it is fragile
under a multi-file, multi-contributor setup: adding a middleware in the wrong position silently
reorders the chain, and there is no machine-checked signal that the order is correct. Auditing
the chain requires reading every `e.Use` call in sequence.

## Decision

Each middleware carries an **explicit integer `Priority` field** declared in its `*_di.go`.
The `UseMiddleware` and `PreMiddleware` value-group structs in `internal/di/server/extension`
hold the priority alongside the `echo.MiddlewareFunc`. At server startup, `ApplyExtends`
collects all contributed middleware entries, sorts them by priority (ascending, lower number
applied first), validates that no two entries in the same kind (`Pre` / `Use`) share the same
priority value, and applies them to Echo in order.

The chain order is therefore **data**, not call-site ordering: each owner declares a
priority number, the engine enforces uniqueness and applies the sorted list, and the effective
order is observable from the priority values alone without reading multiple files end-to-end.

## Consequences

### Positive Consequences

- Chain order is auditable from priority integers alone; a new contributor can see the full
  order without tracing every wiring file.
- Priority conflicts (two middleware claiming the same slot) are detected at startup and
  produce an explicit error, rather than silently applying in an arbitrary order.
- Each `*_di.go` is fully self-contained: it declares the middleware and its priority without
  needing to know where other middleware sit in a shared list.
- The `httpstack/` directory stays free of Echo-instance and fx dependencies; middleware
  implementations remain independently testable.

### Negative Consequences

- A contributor adding a new middleware must choose a priority number that does not collide
  with any existing value in the same kind, requiring awareness of the existing assignment
  table.
- The integer-based slot scheme is a flat namespace: inserting a middleware between priority 7
  and 8 when 7.5 is not an integer requires renumbering neighbours.

## Alternatives Considered

### Hardcoded call-site ordering in a single setup function

Straightforward: one function lists all `e.Use(...)` calls in sequence. Order is immediately
visible in that function. Rejected because it does not compose with the fx value-group pattern
— each `*_di.go` would need to call into a shared setup function or the setup function would
need to import every middleware package, coupling everything to a single registration site.

### Named ordered slice in a configuration struct

Define a fixed ordered slice of middleware names in a config struct; each middleware registers
itself by name. Provides a single canonical list. Rejected because it requires maintaining a
list in two places (the config and the `*_di.go`) and introduces string-matching indirection
that is harder to type-check.

### fx ordered-group annotation (soft ordering)

Uber fx supports `group:"...,soft"` to express ordering hints. Not used because fx groups do
not guarantee stable ordering under recompilation, and the chain order here is a hard
correctness requirement, not a hint.

## Notes

- Extension engine implementation: `internal/di/server/extension/extension.go`.
- Middleware implementations: `internal/controller/httpstack/`.
- Priority assignment table: see `docs/design/rest.md` § "Glossary" (priority entry).
- source: `docs/design/rest.md` § "Role theory" (design principle: "Ordered middleware by
  priority") and Glossary (priority / extension engine entries).
