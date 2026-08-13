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

One subdirectory per aggregate or concern; each holds that unit's `.sql` files.

- `dashboard/` — Sales totals and purchase-status counts for the operational dashboard
- `product/` — Product ranking projection
- `purchase/` — Purchase detail and summary projections across aggregates

## Naming Convention

- Files: verb + target (e.g., `list_published_products.sql`)
- `-- name:` annotation required on all queries

## Code Generation

```sh
make gen-query
```
