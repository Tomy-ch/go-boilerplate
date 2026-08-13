# Repository DML

English | [日本語](README.ja.md)

SQL for Aggregate persistence and simple single-Aggregate reads (CRUD, plus simple filter / list / count by the Aggregate's own attributes).

## Purpose

- Centralize INSERT / UPDATE / DELETE / SELECT queries -- including simple filter / list / count by a single Aggregate's own attributes -- for persisting and retrieving domain models.
- Keep SQL simple and free of business logic -- cross-Aggregate reads, aggregation, and complex joins belong in QueryService.
- Provide type-safe repository implementations via sqlc code generation.

## Infrastructure Mapping

Implementation: `internal/infrastructure/rdb/repository/`

## Directory Structure

One subdirectory per aggregate or concern; each holds that unit's `.sql` files.

- `prefecture/` — Prefecture master reads
- `product/` — Product CRUD, stock updates, and image rows
- `product_category/` — Product category master reads
- `product_status/` — Product status master reads
- `purchase/` — Purchase reads, row locks, and lifecycle status updates
- `user/` — User CRUD, keyword search, row locks, and purge candidates
- `user_identity/` — Resolving a user from an external identity

## Naming Convention

- Files: verb + target (e.g., `select_user_by_id.sql`)
- `-- name:` annotation required on all queries

## Code Generation

```sh
make gen-query
```
