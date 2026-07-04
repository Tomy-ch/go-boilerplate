---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [persistence, migration, data]
---

# ADR-0024: Ship master data via migration; keep transactional seed out of production

## Status

accepted

## Context

Two categories of initial data exist in the project:

1. **Master data** — reference tables that must be present for the application to function
   correctly in any environment (prefectures, product statuses, purchase statuses). These
   rows are stable, enumerable, and production-required.
2. **Transactional seed data** — users, products, demo orders needed only in
   development/test environments. These rows must never reach production.

Mixing both categories into a single seeding mechanism creates deployment risk: a seed step
that runs in production could insert test data. Alternatively, omitting a dedicated seed
step forces operators to manually insert master data after every fresh deployment.

## Decision

**Master data** is inserted inside its corresponding migration file alongside the table DDL,
using `INSERT ... ON CONFLICT (id) DO NOTHING` for idempotency. Examples:

- `000003_create_prefectures.up.sql` — inserts all 47 Japan prefectures alongside the
  `CREATE TABLE` statement.
- `000005_create_product_statuses.up.sql` — inserts the initial product status rows.
- `000008_create_purchase_statuses.up.sql` — inserts the initial purchase status rows.

`make migrate-up` is the only command required to bring a new environment to a fully
functional state; no separate data step is needed.

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

### Negative Consequences

- Correcting a master data value (e.g., a misspelled prefecture name) requires a new
  migration file rather than a simple row update, consistent with
  [ADR-0022](0022-append-only-immutable-migrations.md).
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
- Related: [ADR-0022](0022-append-only-immutable-migrations.md) (immutability);
  [ADR-0023](0023-sequential-migration-ids.md) (numbering).
