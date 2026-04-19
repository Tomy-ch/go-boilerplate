# CommandService DML

English | [日本語](README.ja.md)

Write SQL for state changes (INSERT / UPDATE / DELETE), designed as a future extension point.

## Purpose

- Centralize write operations that mutate application state, separate from repository CRUD and read-only queries.
- Ensure transactional integrity for multi-table updates and complex state transitions.
- Generate type-safe Go code via sqlc for compile-time parameter validation.

## Infrastructure Mapping

Implementation: `internal/infrastructure/rdb/command_service/` (future)

## Directory Structure

```text
command_service/
├── user/
│   ├── insert_user.sql
│   ├── update_user_email.sql
│   └── ...
├── product/
│   ├── publish_product.sql
│   └── ...
└── ...
```

## Naming Convention

- Files: verb + target (e.g., `update_user_email.sql`)
- `-- name:` annotation required on all queries

## Code Generation

```sh
make gen-query
```
