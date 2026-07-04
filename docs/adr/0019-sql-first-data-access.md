---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [persistence, data]
---

# ADR-0019: SQL-first data access

## Status

accepted

## Context

Data access needs predictable performance characteristics and explicit, auditable query
behavior. Hiding SQL behind an ORM obscures what actually runs against the database, which
conflicts with the template's goals of type safety, structural clarity, and long-term
operability.

## Decision

Treat **SQL as an explicit contract**: queries are written directly in SQL rather than
generated implicitly by an ORM. SQL is the first-class artifact of the data-access layer;
Go code is derived from it (see [ADR-0020](0020-sqlc-type-safe-sql.md)).

## Consequences

### Positive Consequences

- Full control over every query.
- Clear, predictable performance characteristics.
- Explicit, reviewable data-access patterns.

### Negative Consequences

- More boilerplate than an ORM's convenience methods; each access path is written as SQL.
- No automatic cross-database portability that an ORM abstraction might provide.

## Alternatives Considered

### Full ORM

Rejected: convenient, but obscures query behavior and performance, working against the
explicit-data-access goal.

### Query builder

Rejected: reduces SQL visibility and adds an abstraction layer that increases complexity
without a matching benefit here.

## Notes

- The concrete type-safe generation mechanism is [ADR-0020](0020-sqlc-type-safe-sql.md) (sqlc).
- Enforced in part by the Repository / QueryService rules in [`docs/rules.md`](../rules.md#repository--queryservice-rules).
- Migrated from `docs/decisions.md` (§ "Why SQL-first").
