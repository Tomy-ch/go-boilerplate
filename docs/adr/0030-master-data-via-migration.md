---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [persistence, migration, data]
---

# ADR-0030: Ship master data via migration; keep transactional seed out of production

## Status

accepted

## Context

Two categories of initial data exist in the project:

1. **Master data** — reference tables that must be present for the application to function
   correctly in any environment. These rows are stable, enumerable, and production-required.
2. **Transactional seed data** — demo/mock data needed only in development/test
   environments. These rows must never reach production.

Mixing both categories into a single seeding mechanism creates deployment risk: a seed step
that runs in production could insert test data. Alternatively, omitting a dedicated seed
step forces operators to manually insert master data after every fresh deployment.

## Decision

**Master data** is inserted inside its corresponding migration file alongside the table DDL,
using `INSERT ... ON CONFLICT (id) DO NOTHING` for idempotency.

`make migrate-up` is the only command required to bring a new environment to a fully
functional state; no separate data step is needed.

A row that references master data resolves the master row's identifier **in SQL, by the master's
stable business key**, at insert time (a sub-`SELECT` on its `code` column) — never by carrying
the master row's UUID as a constant in application code. The migration is then the single place
that identifier is decided; a copy in application code would be a second place, and two places
drift silently the first time only one of them moves. What application code holds instead is the
business key itself, which is part of the domain vocabulary and stays meaningful regardless of
which UUID the migration happened to assign.

**Transactional seed data** lives in `database/seed/` and is applied only via `make
db-seed` (or `./server db-seed`). This command is explicitly excluded from production use.
See [`database/seed/README.md`](../../database/seed/README.md) for the full policy.

## Consequences

### Positive Consequences

- A fresh production deployment is complete after `make migrate-up` alone; no separate
  seeding step is required or allowed.
- `ON CONFLICT DO NOTHING` makes master-data inserts idempotent — safe to re-run if a
  migration is applied more than once (e.g., during testing).
- The boundary between production-required data and development-only data is explicit and
  enforced by separate commands.
- A master row's identifier exists in exactly one place — the migration that inserts it — so
  application code cannot pin a UUID that a re-seeded or corrected master no longer carries.

### Negative Consequences

- Correcting a master data value (e.g., a misspelled prefecture name) requires a new
  migration file rather than a simple row update, consistent with
  [ADR-0028](0028-append-only-immutable-migrations.md).
- Migration files contain both DDL and DML, which slightly broadens their scope beyond pure
  schema definitions.

## Alternatives Considered

### All initial data via seed/

Both master and transactional data would be in `database/seed/`. Production deployments
would need a filtered seeding step to run only master-data files. This adds operational
complexity and risk of accidental test data leakage. Rejected.

### Separate master-data migration category

A dedicated directory or command for master-data migrations. Adds a new concept without
meaningful benefit over embedding the INSERT statements in the same migration file that
creates the table. Rejected.

## Notes

- Source: [`database/seed/README.md`](../../database/seed/README.md) § "Difference from
  Migrations" and § "Notes".
- Source: [`database/README.md`](../../database/README.md).
- Related: [ADR-0028](0028-append-only-immutable-migrations.md) (immutability);
  [ADR-0029](0029-sequential-migration-ids.md) (numbering).
