---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [persistence, cqrs, architecture]
---

# ADR-0027: Adopt lightweight CQRS — Repository for writes, QueryService for reads

## Status

accepted

## Context

In a pure onion architecture all persistence is mediated by Repository interfaces defined
in the domain layer. This is appropriate for aggregate-level writes and simple reads (fetch
by ID, count by own attributes) because Repositories guard domain invariants and reconstruct
full aggregates.

The model breaks down for use-case-specific read requirements:

- **Cross-aggregate joins**: building a dashboard that connects many aggregates — or
  grouping and aggregating data across multiple aggregate boundaries — cannot be expressed
  as a single-aggregate Repository method without leaking join and aggregation details into
  the domain.
- **View-shaped DTOs**: API responses often need a subset or reshaping of aggregate data
  (pagination metadata, nested objects). Returning full aggregates and mapping in the
  usecase layer is correct but wasteful for large read sets.
- **Complex filtering and full-text search**: keyword search across multiple columns,
  full-text search, and paginated results require query patterns that do not fit cleanly in
  a domain interface.

Forcing these queries into Repository pollutes domain interfaces with view-specific methods,
erodes aggregate encapsulation, and makes the domain depend on presentation concerns. The
opposite extreme — full CQRS with a separate read database, event projections, and eventual
consistency — introduces significant infrastructure and operational complexity that is not
warranted at the current scale. The project needs a middle path.

## Decision

Adopt **lightweight CQRS** on the same PostgreSQL instance, splitting persistence into
three responsibilities:

### Repository (aggregate-scoped CRUD and some aggregation)

- Interface defined in the **domain layer** (`internal/domain/<aggregate>/<aggregate>_repository.go`).
- Responsible for aggregate persistence and simple single-aggregate reads: fetch by ID,
  simple filter/list/count by the aggregate's own attributes.
- Does not handle cross-aggregate joins, aggregation, or keyword search.
- Implementation lives in `internal/infrastructure/rdb/repository/<aggregate>/`.

### QueryService (query/read path)

- Interface defined in the **usecase layer** (`internal/usecase/<aggregate>/query/`), not
  the domain — because the read model is a usecase concern, not an aggregate invariant.
- Handles reads that cross aggregate boundaries, require multi-table JOINs, pagination,
  full-text search, or return a read-model projection that would be wasteful to reconstruct
  as a full aggregate (a subset/reshape of a heavy aggregate, or a joined view).
- Returns DTOs, not full domain entities.
- A read is *not* a QueryService case merely because its result is returned as a DTO — every
  read is eventually mapped to a response DTO. Simple single-aggregate reads (fetch by ID, and
  filter / list / count by the aggregate's own attributes, including an unfiltered full list)
  stay in Repository. Returning many rows is not "crossing aggregates"; crossing means spanning
  *different* aggregates.
- The split is **storage-agnostic**: what decides Repository vs QueryService is whether the read
  targets the aggregate's system-of-record state (the full aggregate is reconstructable →
  Repository) or a derived projection / read model — a search index, a separate search store, a
  denormalized / generated search column, or a cross-aggregate JOIN view (the aggregate cannot be
  reconstructed → QueryService). Full-text search is a *typical* QueryService case because it
  usually rides on a derived search projection, not because "search" is inherently a read-side
  concern; a plain filter on the aggregate's own columns stays in Repository.
- Field resolution across *independent* aggregates (e.g. attaching a separate top-level aggregate's
  name — a prefecture — to a list) is done in the **usecase layer** by batch-fetching the related
  aggregate through its own Repository (`FindByIDs`) and merging by key — keeping each read a
  single-aggregate Repository read — rather than via a QueryService SQL JOIN.
- **Reference-master clarification**: a *reference master* — fixed / standing lookup data with no
  independent write / transactional lifecycle, reached through a mandatory, uniquely-determined FK
  <!-- 撤去後にこの箇所へ自分の例を置くための指針。
       目的: 「参照マスタ」だけでは読者が自分のテーブルを当てはめられない。
       意義: 判定は独立した書き込みライフサイクルの有無で決まり、テーブルの名前や規模ではない。
       書き方: 列挙型に相当する固定テーブルを 1〜2 個挙げる。 -->
  <!-- sample-api:begin -->
  （サンプルでの例は `purchase_statuses` / `product_statuses`）
  <!-- sample-api:end -->
  — is part of the
  owning aggregate's semantic set, not a foreign aggregate. Projecting its display attribute (e.g. a
  status *name*) by JOINing it into the owner's read is a single-aggregate **Repository** read, not a
  cross-aggregate JOIN, and needs neither a QueryService nor a usecase merge. The criterion is the
  joined data's *nature*, not its Go modeling: a master resolved purely by JOIN with no domain type
  of its own qualifies just as one that also has a sub-package (`internal/domain/product/status`
  exists only for a master-list endpoint). The cross-aggregate merge rule above applies only to
  *independent* aggregates (own transactional lifecycle). See `docs/rules.md` § "Repository /
  QueryService Rules" for the discriminator.
- Implementation lives in `internal/infrastructure/rdb/query_service/<aggregate>/`.

### Command Service (command/write path)

- Interface defined in the **usecase layer** (`internal/usecase/<workflow>/command/`). Ownership of a
  cross-aggregate write port is decided on the **workflow** axis, not the aggregate axis. Trying to
  pick an owning aggregate does not generalize: one real write (a coupon redemption, say) can enforce
  the invariants of user, product and purchase at once, so "the aggregate whose invariant it enforces"
  is a relation, not a function. A transaction, by contrast, always has exactly one initiator, and
  [`docs/rules.md`](../rules.md) already gives the usecase layer ownership of transaction boundaries.
  `internal/usecase/purchase/command/` names a workflow, not an aggregate, so "why purchase and not
  product?" does not arise.
- Domain Service and CommandService therefore diverge deliberately: a Domain Service is a *rule* and
  owns no transaction; a CommandService is a *transaction tool* and is owned by whoever opens the
  transaction. That every condition a CommandService enforces is derived from a domain invariant is
  guaranteed by the Derivation rule below, not by where the file sits.
- A CommandService method **receives the decided aggregate** — `CreateX(ctx, *x.X)` — symmetric to
  how a Repository returns one, rather than a decomposed parameter bag. Passing the aggregate keeps
  the decided write unit intact; a parameter bag scatters the write intent across a signature and
  breaks that symmetry. This is not a DTO-boundary violation: infrastructure legitimately handles
  domain entities, since repositories already map rows to and from them, and the DTO-boundary rule
  targets what the *controller* is exposed to, not what infrastructure receives.
- After executing a write operation, the Usecase calls back through the Repository for the
  affected aggregate to validate correctness, preserving domain integrity.
- The Usecase return value is a DTO, not a domain entity.
- Implementation lives in `internal/infrastructure/rdb/command_service/<aggregate>/`.

> **Implementation status**: the `command_service` sub-module in `persistenceModule`
> (`internal/di/module/persistence.go`) holds exactly one provider, and it belongs to the sample
> purchase feature. Removing the samples empties the sub-module and leaves this section describing
> an intended design with no occupant — which is the state a fork starts from. The occupant is kept
> because the eligibility bar below is only legible against a concrete case that meets it.

Repository, QueryService, and CommandService are all registered in `persistenceModule` in
`internal/di/module/persistence.go` and injected via Uber Fx (see
[ADR-0035](0035-uber-fx-di.md)). This is not full CQRS: there is no separate read store,
event sourcing, or eventually-consistent projection pipeline.

See [`docs/rules.md`](../rules.md) § "Repository / QueryService Rules" for the
day-to-day boundary enforcement rules.

## Consequences

### Positive Consequences

- Repository stays aggregate-focused; domain interfaces do not accumulate view-specific
  methods, preserving domain purity per [ADR-0002](0002-onion-architecture.md).
- QueryService can freely optimize queries (joins, pagination, full-text search) without
  touching domain logic or exposing domain entities to the read path.
- Both Service interfaces are owned on the same axis — the workflow that opens the transaction and
  shapes the read model — so they live together in the usecase layer, and the domain layer keeps
  exactly one persistence contract: the Repository.
- CommandService can freely optimize flexible updates, deletes, and other write operations
  without touching domain logic. Routing back through the Repository at the end prevents
  domain integrity from being compromised.
- All three abstractions remain behind interfaces and are injected via DI, keeping them
  replaceable per [ADR-0001](0001-avoid-lock-in.md).
- No new infrastructure dependency; all three paths run against the same PostgreSQL instance.

### Negative Consequences

- Three persistence abstractions (Repository, QueryService, and CommandService) require
  developers to decide which to use for a given operation. The boundary is documented in
  `docs/rules.md` but requires understanding.
- Both Service interfaces sit in the usecase layer, further from the domain, which can make intent
  less obvious when reading domain code in isolation. In particular, reading a domain package alone
  does not reveal that a write path exists which does not pass through its Repository; the
  eligibility bar and the Derivation rule below are what keep that path from becoming a general
  escape hatch.
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
lookups (e.g., fetch user by ID for a write precondition check).

Rejected as a DDD anti-pattern that produces **domain anemia**. Aggregate operations
(business rules, invariant checks, command preconditions) depend on reading aggregate state
through the Repository; eliminating those reads strips the domain of the data it needs to
reason about its own invariants, hollowing it into an anemic shell. This is not a mild
trade-off — it fundamentally compromises the onion architecture established in
[ADR-0002](0002-onion-architecture.md).

### CQRS at the usecase level only (no QueryService abstraction)

Have usecases call Repository methods directly for complex reads, applying in-memory joins.
Avoids a new abstraction but transfers N+1 query and performance concerns to the application
layer. Rejected for performance and correctness reasons.

### Route every write through the aggregate Repository (abolish CommandService)

Express all writes as "load the aggregate, mutate it, save it", so the Repository is the only
persistence seam and every write passes through the aggregate that owns the invariant. This is the
shape Evans describes, and it would remove the asymmetry of having a read seam and a write seam with
different owners.

Rejected because some writes cannot be expressed that way without changing their concurrency
properties. Restoring stock on cancellation is a relative update that takes no lock on the product
row at all; expressing it as "load the product, add back, save" would require introducing a lock the
cancel path does not currently take, adding contention and a deadlock surface that does not exist
today. The same holds for any set-based or counter-style write. The seam exists for that class of
write and for nothing else — the eligibility rule below is what keeps it from becoming a general
escape hatch.

### Decompose the aggregate into a parameter bag on the CommandService signature

Pass the decided values as individual parameters instead of the aggregate, on the reading that
infrastructure should not receive a domain entity. Rejected: it scatters the write intent across a
signature that grows with every field, and it breaks the Repository-symmetry that gives the write
seam its shape. The premise is also wrong — the DTO-boundary rule exists to keep domain entities out
of *controller* responses, not out of infrastructure, which already maps rows to and from entities.

**Eligibility.** A write belongs on CommandService only when it cannot be expressed as loading an
aggregate and saving it: relative updates, set-based operations, and operations that obtain atomicity
without taking a lock. Anything that can be read-modify-saved goes on the Repository. Without this
line the seam degrades into "where I put SQL I want to write directly".

**Derivation.** Any condition CommandService enforces must be *derived from* a domain invariant, never
authored independently. The stock guard in the decrement statement restates the domain's
insufficient-stock rule as a fail-closed second net ([ADR-0031](0031-ordered-pessimistic-row-locks.md));
it is downstream of that rule, so a change to the domain rule obliges a change here, and never the
reverse. Two independently written copies of one rule diverge silently the first time only one moves.

## Notes

- Source: [`internal/infrastructure/rdb/query_service/README.md`](../../internal/infrastructure/rdb/query_service/README.md)
  § "Relationship to CQRS" and § "When to Use QS Over Repository".
- Source: [`docs/rules.md`](../rules.md) § "Repository / QueryService Rules".
- DI registration: [`internal/di/module/persistence.go`](../../internal/di/module/persistence.go).
- Related: [ADR-0028](0028-system-cqrs-dml-category.md) (system_cqrs as a fourth, non-CQRS
  category).
