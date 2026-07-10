# SystemQuery DML

English | [日本語](README.ja.md)

System operational queries for health checks, metrics collection, and infrastructure monitoring.

## Purpose

- Provide queries for system health verification and operational metrics.
- Separate system-level concerns from application business logic.
- Generate type-safe Go code via sqlc for compile-time parameter and scan validation.

## Infrastructure Mapping

Implementation: `internal/infrastructure/rdb/system_cqrs/`

## Directory Structure

```text
system_cqrs/
├── health_check/
│   ├── select_system_health.sql
│   └── ...
├── metrics/
│   ├── select_system_metrics.sql
│   └── ...
└── ...
```

## Naming Convention

- Files: verb + target (e.g., `select_system_health.sql`)
- `-- name:` annotation required on all queries

## Code Generation

```sh
make gen-query
```
