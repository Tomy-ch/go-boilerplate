---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [persistence, cqrs, architecture]
---

# ADR-0025: Adopt lightweight CQRS — Repository for writes, QueryService for reads

## Status

accepted

## Context

In a pure onion architecture all persistence is mediated by Repository interfaces defined
in the domain layer. This is appropriate for aggregate-level writes and simple reads (fetch
by ID, count by own attributes) because Repositories guard domain invariants and reconstruct
full aggregates.

The model breaks down for use-case-specific read requirements:

- **Cross-aggregate joins**: a user list API must return the prefecture name alongside each
  user. Joining `users` and `prefectures` cannot be expressed as a single-aggregate
  Repository method without leaking the join detail into the domain.
- **View-shaped DTOs**: API responses often need a subset or reshaping of aggregate data
  (pagination metadata, nested objects). Returning full aggregates and mapping in the
  usecase layer is correct but wasteful for large read sets.
- **Complex filtering and full-text search**: keyword search across multiple columns, GIN
  index queries, and paginated results require query patterns that do not fit cleanly in a
  domain interface.

Forcing these queries into Repository pollutes domain interfaces with view-specific methods,
erodes aggregate encapsulation, and makes the domain depend on presentation concerns. The
opposite extreme — full CQRS with a separate read database, event projections, and eventual
consistency — introduces significant infrastructure and operational complexity that is not
warranted at the current scale. The project needs a middle path.

## Decision

Adopt **lightweight CQRS** on the same PostgreSQL instance, splitting persistence into
three responsibilities:

### Repository (command/write path)

- Interface defined in the **domain layer** (`internal/domain/<aggregate>/repository.go`).
- Responsible for aggregate persistence and simple single-aggregate reads: fetch by ID,
  simple filter/list/count by the aggregate's own attributes.
- Does not handle cross-aggregate joins, aggregation, or keyword search.
- Implementation lives in `internal/infrastructure/rdb/repository/<aggregate>/`.

### QueryService (query/read path)

- Interface defined in the **usecase layer** (`internal/usecase/<aggregate>/query/`), not
  the domain — because the read model is a usecase concern, not an aggregate invariant.
- Handles reads that cross aggregate boundaries, require multi-table JOINs, pagination,
  full-text search, or return DTOs shaped per API response.
- Returns DTOs, not full domain entities.
- Implementation lives in `internal/infrastructure/rdb/query_service/<aggregate>/`.

### Command Service (placeholder)

- A reserved sub-module in `persistenceModule` for future write-side complexity that does
  not fit a domain Repository (e.g., bulk operations, write-optimized commands against
  non-domain tables). Currently empty.

Both Repository and QueryService are registered in `persistenceModule` in
`internal/di/module/persistence.go` and injected via Uber Fx (see
[ADR-0030](0030-uber-fx-di.md)). This is not full CQRS: there is no separate read store,
event sourcing, or eventually-consistent projection pipeline.

See [`docs/rules.md`](../../rules.md) § "Repository / QueryService Rules" for the
day-to-day boundary enforcement rules.

## Consequences

### Positive Consequences

- Repository stays aggregate-focused; domain interfaces do not accumulate view-specific
  methods, preserving domain purity per [ADR-0002](0002-onion-architecture.md).
- QueryService can freely optimize queries (joins, pagination, full-text search, GIN index)
  without touching domain logic or exposing domain entities to the read path.
- The usecase layer owns the QueryService interface: the read model is a usecase concern, so
  its interface belongs there rather than in the domain.
- Both sides remain behind interfaces and are injected via DI, keeping them replaceable per
  [ADR-0001](0001-avoid-lock-in.md).
- No new infrastructure dependency; both paths run against the same PostgreSQL instance.

### Negative Consequences

- Two persistence abstractions (Repository and QueryService) require developers to decide
  which to use for a given read. The boundary is documented in `docs/rules.md` but requires
  understanding.
- QueryService interfaces in the usecase layer are further from the domain, which can make
  intent less obvious when reading domain code in isolation.
- The "no complex reads in Repository" boundary must be maintained by review; there is no
  compiler enforcement for the distinction.

## Alternatives Considered

### Fat Repository (all reads in Repository)

Put all reads — including joins, pagination, and keyword search — in domain Repository
interfaces. Simple to understand and requires only one persistence abstraction.

Rejected because it pollutes domain interfaces with view-specific queries, couples the
domain to presentation requirements, and undermines aggregate encapsulation over time. The
domain should not know how the API shapes its response.

### Full CQRS with a separate read store

Maintain a dedicated read database (e.g., Elasticsearch, read-replica with materialized
views) updated via event projections. Provides strong read scalability and enables NLP-grade
search.

Rejected as premature: the current dataset and query complexity do not require a separate
store. Eventual consistency and projection maintenance would add operational overhead not
yet warranted.

### All reads via QueryService (abolish Repository reads)

Eliminate read methods from Repository entirely; route all reads through QueryService.
Simplifies the boundary but forces QueryService overhead onto trivial single-aggregate
lookups (e.g., fetch user by ID for a write precondition check). Rejected — the Repository
is the natural home for reads that are integral to aggregate lifecycle.

### CQRS at the usecase level only (no QueryService abstraction)

Have usecases call Repository methods directly for complex reads, applying in-memory joins.
Avoids a new abstraction but transfers N+1 query and performance concerns to the application
layer. Rejected for performance and correctness reasons.

## Notes

- Source: [`internal/infrastructure/rdb/query_service/README.md`](../../../internal/infrastructure/rdb/query_service/README.md)
  § "Relationship to CQRS" and § "When to Use QS Over Repository".
- Source: [`docs/rules.md`](../../rules.md) § "Repository / QueryService Rules".
- DI registration: [`internal/di/module/persistence.go`](../../../internal/di/module/persistence.go).
- Related: [ADR-0026](0026-system-query-dml-category.md) (system_query as a fourth, non-CQRS
  category); [ADR-0028](0028-in-database-full-text-search.md) (QueryService used for FTS).
