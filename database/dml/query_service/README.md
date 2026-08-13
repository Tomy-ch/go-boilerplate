# QueryService DML

English | [日本語](README.ja.md)

Read-only SQL for search and list optimization, bypassing the domain layer.

## Purpose

- Provide optimized read-only queries with JOINs, aggregation, and filtering at the SQL level.
- Separate read concerns from write operations and transaction management.
- Generate type-safe Go code via sqlc for compile-time parameter and scan validation.

## Infrastructure Mapping

Implementation: `internal/infrastructure/rdb/query_service/`

## Directory Structure

One directory per read model, named after the aggregate the projection is read from.

## Naming Convention

- Files: verb + target (e.g., `list_published_products.sql`)
- `-- name:` annotation required on all queries

## Code Generation

```sh
make gen-query
```
