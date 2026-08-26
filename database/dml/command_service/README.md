# CommandService DML

English | [日本語](README.ja.md)

Write SQL for state changes (INSERT / UPDATE / DELETE) that do not fit the load-mutate-save shape of a Repository.

## Purpose

- Centralize write operations that mutate application state, separate from repository CRUD and read-only queries.
- Ensure transactional integrity for multi-table updates and complex state transitions.
- Generate type-safe Go code via sqlc for compile-time parameter validation.

## Infrastructure Mapping

Implementation: `internal/infrastructure/rdb/command_service/`

## Directory Structure

One directory per aggregate whose writes need this category, named after it.

## Naming Convention

- Files: verb + target (e.g., `update_user_email.sql`)
- `-- name:` annotation required on all queries

## Code Generation

```sh
make gen-query
```
