---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [persistence, cqrs]
---

# ADR-0030: Introduce system_cqrs as a fourth DML category outside the CQRS split

## Status

accepted

## Context

The DML directory organizes SQL source files by their architectural role:

| Category | Purpose |
| --- | --- |
| `repository/` | Aggregate CRUD (domain layer interface) |
| `query_service/` | Use-case-specific read queries (usecase layer interface) |
| `command_service/` | Write-side commands (usecase layer, reserved) |

Some queries do not belong to any of these three roles. Health checks, idempotency key
lookups, and outbox row polling are infrastructure-level operations that:

- Are not driven by a user-facing use case
- Do not correspond to a domain aggregate
- Must exist in every deployment regardless of business features

Forcing these queries into `repository/` or `query_service/` is misleading — they have no
aggregate owner and no use-case interface. A dedicated category makes the distinction
explicit.

## Decision

`database/dml/system_cqrs/` is a **fourth DML category** for system operational queries
(health verification, idempotency enforcement, outbox delivery). Its implementations live
in `internal/infrastructure/rdb/system_cqrs/` and are registered under a dedicated
`system_cqrs` sub-module in `persistenceModule`
(`internal/di/module/persistence.go`).

The `system_cqrs` category is explicitly outside the CQRS read/write split described in
[ADR-0029](0029-lightweight-cqrs.md). It serves infrastructure concerns, not application
business logic.

`make gen-query` processes all four categories through the same merge-dml and sqlc pipeline
(see [ADR-0025](0025-merged-dml-schema-as-sqlc-input.md)), so system_cqrs participates
in the same type-safe code generation as the other categories.

## Consequences

### Positive Consequences

- System operational queries are isolated from application DML; there is no risk of mixing
  domain persistence with health checks or idempotency writes.
- Infrastructure implementations (health check, idempotency, outbox) are cleanly registered
  in their own DI sub-module without polluting Repository or QueryService modules.
- The four-category model maps directly to four sub-modules in `persistenceModule`,
  providing a consistent structure for both DML and DI.

### Negative Consequences

- Developers must be aware of the four-category model when placing new queries, adding
  cognitive overhead compared to a two-category or flat model.
- The name `system_cqrs` sounds read-only but includes idempotency writes and outbox
  inserts; "infrastructure-operational" would be more precise but longer.

## Alternatives Considered

### Merge system queries into repository/

Simple structure, but `repository/` maintains a strict 1:1 structure with domain aggregates,
so system operational concerns that have no aggregate owner must not live there. It conflates
domain-aggregate concerns with infrastructure-operational concerns. Health checks and
idempotency keys have no aggregate; placing them in `repository/` is misleading and makes the
Repository concept less coherent.

### A single infrastructure/ category

More general — all non-CQRS queries go in one place. Loses the per-role clarity that the
four-category model provides and obscures which SQL is domain-related vs
infrastructure-related.

### No dedicated DML category (inline SQL in Go files)

Removes the extra directory but foregoes sqlc type safety for system queries. The
compile-time guarantees sqlc provides are worth the additional structure.

## Notes

- Source: [`database/dml/README.md`](../../database/dml/README.md) § "Directory
  Structure" and § "Subdirectory Mapping to Onion Architecture".
- Source: [`database/dml/system_cqrs/README.md`](../../database/dml/system_cqrs/README.md).
- DI registration: [`internal/di/module/persistence.go`](../../internal/di/module/persistence.go).
- Related: [ADR-0029](0029-lightweight-cqrs.md) (CQRS split that system_cqrs falls
  outside of); [ADR-0025](0025-merged-dml-schema-as-sqlc-input.md) (merge-dml pipeline).
