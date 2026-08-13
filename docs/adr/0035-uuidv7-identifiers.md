---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [persistence, identifiers]
---

# ADR-0035: Use UUIDv7 (time-ordered) identifiers for all entity primary keys

## Status

accepted

## Context

Entity primary keys must satisfy several constraints:

- **Global uniqueness** — no coordination with a central authority required.
- **Domain-layer generation** — IDs must be creatable in the domain without a database
  round-trip (e.g., to check out an aggregate before persisting it).
- **Index performance** — random primary keys (UUIDv4) cause frequent B-tree page splits
  because each new row is inserted at a random position in the index. At scale this
  increases write amplification and index fragmentation.
- **No information leak** — sequential integers leak the total record count and insertion
  order, which is undesirable for externally exposed IDs.

UUIDv7 (RFC 9562) embeds a millisecond-precision Unix timestamp in the most-significant
bits, producing time-ordered values. New rows append near the end of the B-tree index,
similar to auto-increment integers, while retaining global uniqueness and requiring no
central sequence generator.

## Decision

All entity primary keys use **UUIDv7**, generated via `pkg/uuid.New()`, which wraps
`github.com/google/uuid.NewV7()`.

The `sqlc.yaml` type override maps the PostgreSQL `uuid` column type to `pkg/uuid.UUID`
for both non-nullable and nullable columns:

```yaml
overrides:
  - db_type: uuid
    go_type:
      import: go-boilerplate/pkg/uuid
      package: uuid
      type: UUID
  - db_type: uuid
    nullable: true
    go_type:
      import: go-boilerplate/pkg/uuid
      package: uuid
      type: UUID
      pointer: true
```

This eliminates per-field UUID conversion throughout the codebase: sqlc-generated code
uses `pkg/uuid.UUID` directly, so QueryService and Repository implementations can pass
generated row fields to domain constructors without an intermediate conversion step.

## Consequences

### Positive Consequences

- Time-ordered inserts reduce B-tree page splits and fragmentation compared to UUIDv4,
  improving write throughput and index compactness at scale.
- IDs are generated in the domain layer without a database round-trip, keeping the domain
  self-contained.
- No central sequence generator is required; UUIDv7 generation is independent on every
  process and host.
- The sqlc type override eliminates per-field UUID conversion across QueryService and
  Repository, removing a repeated conversion step and the opportunity for conversion bugs.
- IDs are monotonically increasing within a millisecond, giving approximate insertion-order
  sortability for free.

### Negative Consequences

- UUIDv7 encodes a millisecond-precision timestamp in the high bits. Exposing UUIDs
  publicly reveals the approximate record creation time.
- Millisecond granularity means multiple records created within the same millisecond are
  not strictly ordered relative to each other.
- Requires `github.com/google/uuid` v1.6 or later (which introduced `NewV7`).

## Alternatives Considered

### UUIDv4 (random)

Widely supported and simple. However, random distribution across the full 128-bit space
causes frequent B-tree page splits, degrading write performance and increasing storage
overhead for the index as the table grows. Rejected in favor of time-ordered IDs.

### Auto-increment integer

Compact and cache-friendly for B-tree indexes. However: leaks record counts and insertion
order when exposed in APIs, requires a centralized sequence generator (database sequence),
and complicates distributed or parallel test scenarios where multiple writers must not
collide. Rejected.

### ULID (Universally Unique Lexicographically Sortable Identifier)

Time-ordered and lexicographically sortable like UUIDv7. Requires a separate library and
is not a UUID-compatible type (different wire format). `github.com/google/uuid`'s `NewV7`
provides equivalent time-ordering without an additional dependency. Rejected.

### UUIDv1

Time-ordered but embeds the MAC address of the generating host, which is a privacy concern
for externally exposed IDs, and the timestamp is encoded in a fragmented way (not
monotonically increasing in the string representation). Rejected.

## Notes

- Source: [`pkg/uuid/README.md`](../../pkg/uuid/README.md) — wraps `github.com/google/uuid`;
  generates UUIDv7.
- Source: [`sqlc.yaml`](../../sqlc.yaml) — `uuid` type override to `pkg/uuid.UUID`.
- Related: [ADR-0025](0025-sqlc-type-safe-sql.md) (sqlc type overrides are the mechanism
  that wires this UUID type into generated code).
