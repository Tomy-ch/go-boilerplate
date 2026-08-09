---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [persistence, codegen]
---

# ADR-0024: Generate type-safe SQL access with sqlc

## Status

accepted

## Context

Given the SQL-first stance ([ADR-0023](0023-sql-first-data-access.md)), the handwritten SQL
still needs to reach Go as **compile-time type-safe** code, without reintroducing an ORM's
runtime abstraction or implicit query generation.

## Decision

Use **sqlc** to generate Go code from the SQL queries. The developer writes SQL; sqlc emits
typed Go functions and row structs that the infrastructure layer wraps.

## Consequences

### Positive Consequences

- Compile-time type safety between SQL and Go.
- SQL stays explicit and readable as the source artifact.
- Minimal runtime abstraction — generated code is thin.

### Negative Consequences

- A code-generation step (`make gen-query`) is required in the workflow; generated files
  are not edited by hand.
- Query capabilities are bounded by what sqlc can parse and map.

## Alternatives Considered

### GORM

Rejected: a convenient ORM, but it reintroduces ORM abstraction and implicit query
generation — exactly what the SQL-first decision avoids.

### Ent

Rejected: a schema-first approach that imposes a different development workflow than
authoring SQL directly.

## Notes

- Builds on [ADR-0023](0023-sql-first-data-access.md) (SQL-first).
- Generated code is not edited by hand — see the Generated Code rules in [`docs/rules.md`](../rules.md#generated-code-rules).
- Migrated from `docs/decisions.md` (§ "Why sqlc").
