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

One subdirectory per aggregate or concern; each holds that unit's `.sql` files.

- `health_check/` — Liveness and readiness probes
- `idempotency/` — Idempotency key lifecycle (claim, complete, expiry sweep)
- `outbox/` — Transactional outbox lifecycle (claim, publish, retry, dead-letter)

## Naming Convention

- Files: verb + target (e.g., `select_system_health.sql`)
- `-- name:` annotation required on all queries

## Code Generation

```sh
make gen-query
```
