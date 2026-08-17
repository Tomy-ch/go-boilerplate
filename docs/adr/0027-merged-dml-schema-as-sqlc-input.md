---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [persistence, codegen, tooling]
---

# ADR-0027: Use merged DML and a dumped schema as sqlc's single input

## Status

accepted

## Context

sqlc requires two inputs: a schema (table definitions) and SQL query files. The schema
evolves through migrations applied to a live database, and DML queries are spread across
per-category subdirectories (`repository/`, `query_service/`, `command_service/`,
`system_cqrs/`). Pointing sqlc directly at raw migration files is impractical because sqlc
does not understand migration sequencing — it would need all DDL statements merged and
applied in order. Pointing it at scattered DML directories without merging would require
sqlc to be aware of the directory layout.

A pre-processing step that produces a single, self-contained input set in a known location
keeps sqlc configuration simple and the generation pipeline deterministic.

## Decision

Two build steps produce a unified input set for sqlc under `database/gen/` before `sqlc
generate` runs:

1. **merge-dml** (`go run ./cmd/ merge-dml --type=$(type) --work-dir=$(work-dir)`)
   concatenates all SQL files from each DML category directory into `database/gen/`,
   producing one merged file per category.
2. **dump-schema** (`go run ./cmd/ dump-schema --work-dir=$(work-dir)`) dumps the schema of
   the live, migrated database into `database/gen/schema.gen.sql`.

`sqlc.yaml` then points at exactly these two artifacts:

```yaml
schema: database/gen/schema.gen.sql
queries: database/gen/
```

`make gen-query` runs both steps followed by `sqlc generate`. `database/gen/` is a
generated artifact and must not be edited manually.

## Consequences

### Positive Consequences

- sqlc always sees the schema that reflects the actual applied-migration state, not raw DDL
  files that could be out of order or partially applied.
- DML files remain organized per category under `database/dml/` but are merged before
  generation, keeping both human organization and tooling simplicity.
- The generated Go code (`internal/infrastructure/rdb/sqlc/gen/`) is fully reproducible
  from the same migrated DB state.
- The committed `schema.gen.sql` snapshot lets reviewers audit the generated Go code without
  re-running the migration pipeline: the runtime source of truth is the SQL files, but the
  generation-input state exists only on the generator's local DB at generation time, so the
  snapshot makes it available for review without reconstruction.

### Negative Consequences

- A live, migrated database must be available before running `make gen-query`; the schema
  dump cannot be produced from migration files alone. In practice this is harmless because a
  local container is always running during normal development.
- `database/gen/` must be committed or regenerated in CI, adding a dependency on the DB
  container during generation.

## Alternatives Considered

### Point sqlc directly at migration files

sqlc does not understand migration ordering. All DDL statements from all migration files
would need to be applied in sequence, which is exactly what `dump-schema` already captures
from the live DB. Rejected because it replicates the migration engine's job.

### Maintain a hand-written schema.sql

A manually curated schema file would drift from the actual applied migrations over time.
The dump-schema approach derives the schema from the applied state, eliminating drift.
Rejected.

### One sqlc invocation per DML category

Running sqlc separately for each category (`repository`, `query_service`, etc.) with
category-specific config would produce separate generated packages and complicate the Go
import graph. The single merged input produces one coherent generated package. Rejected.

## Notes

- Source: [`database/README.md`](../../database/README.md) — Data Lifecycle diagram.
- Related: [ADR-0025](0025-sql-first-data-access.md) (SQL-first data access),
  [ADR-0026](0026-sqlc-type-safe-sql.md) (sqlc as code generator).
- `make gen-query` is the single command that runs merge-dml, dump-schema, and sqlc in
  order.
