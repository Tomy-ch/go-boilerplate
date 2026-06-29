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

```text
repository/
├── user/
│   ├── insert_user.sql
│   ├── select_user_by_id.sql
│   └── ...
├── prefecture/
│   ├── ...
│   └── ...
└── ...
```

## Naming Convention

- Files: verb + target (e.g., `select_user_by_id.sql`)
- `-- name:` annotation required on all queries

## Code Generation

```sh
make gen-query
```
