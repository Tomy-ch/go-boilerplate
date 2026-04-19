# Query Service Implementation Guide

English | [日本語](README.ja.md)

## Position of Query Service in Onion Architecture

In Onion Architecture, persistence abstractions are defined as **Repository interfaces** in the Domain layer. Repository handles per-Aggregate CRUD and guards Domain invariants.

Query Service (QS), on the other hand, is an **intentional exception to this principle**.

```mermaid
flowchart TB
    subgraph "Domain Layer"
        RepoIF["Repository interface"]
    end
    subgraph "Usecase Layer"
        QSIF["QueryService interface"]
    end
    subgraph "Infrastructure Layer"
        RepoImpl["Repository impl"]
        QSImpl["QueryService impl"]
    end

    RepoImpl -. implements .-> RepoIF
    QSImpl -. implements .-> QSIF
```

### Why QS Interface Lives in Usecase, Not Domain

|Aspect|Repository|Query Service|
|---|---|---|
|Concern|Aggregate persistence|Usecase-specific search|
|Granularity|Per Aggregate|Per screen / API response|
|Return type|Domain Entity|DTO (display projection)|
|Invariants|Guaranteed by Domain|Not involved|
|Interface placement|Domain layer|Usecase layer|

QS returns **projections that a usecase needs, not complete Aggregate reconstructions**. This is a Usecase concern, not a Domain concern, so the interface is placed in the Usecase layer (`internal/usecase/<aggregate>/query`).

### Relationship to CQRS

Introducing QS is a **lightweight CQRS (Command Query Responsibility Segregation)** approach.

- **Command (write)**: Goes through Repository, guarding Domain Entity invariants
- **Query (read)**: Goes through QS, executing performance-optimized search queries directly

This is not full CQRS (separate DB / event sourcing), but a **practical design that separates read/write responsibilities on the same DB**.

### When to Use QS Over Repository

Consider QS instead of Repository when:

- Searches requiring multi-table JOINs
- Paginated list retrieval
- Full-text or keyword search
- Queries requiring aggregation or grouping
- Reads that don't need full Aggregate reconstruction

Conversely, simple queries like single retrieval by ID or count can remain in Repository.

## Role

Query Service is a layer that provides **read-only queries such as search and list retrieval**.

```mermaid
flowchart TB
    Controller --> Usecase --> QS["QueryService"] --> Sqlc["sqlc"] --> DB["Database"]
```

The responsibilities of Query Service are as follows:

1. Execute SQL search
2. Convert Row → Domain Entity / DTO
3. Normalize DB errors

Query Service **does not contain business logic.**

## Architecture Position

QueryService implementations are placed in the following location.

`internal/infrastructure/rdb/query_service/<aggregate>/`

Example

```txt
query_service/
 └ user/
     └ user_query_service.go
```

QueryService interfaces are placed in the **Usecase layer**.

`internal/usecase/<aggregate>/query`

Example

```txt
internal/usecase/user/query/user_query_service.go
```

Infrastructure **only implements this interface**.

## QueryService Responsibilities

QueryService is responsible for **search-specific data retrieval**.

```mermaid
flowchart TB
    Query --> Sqlc --> Row --> Domain["Domain Entity or DTO"] --> Ret["return"]
```

QueryService does not perform the following:

- Business rules
- Usecase logic
- Controller processing
- Transaction management

## sqlc Usage

QueryService uses **sqlc generated queries**.

```go
rows, err := db.SearchUsers(ctx, &gen.SearchUsersParams{...})
```

## SQL Splitting Design

Search conditions (active / deleted / all) are not branched within SQL, but implemented as separate queries.

Example:

- SearchUsers
- SearchActiveUsers
- SearchDeletedUsers

Reasons:

- Improve SQL readability
- Improve index efficiency
- Simplify sqlc generated code

## LIKE Search Helper

For keyword search, use helpers from `internal/infrastructure/rdb/sqlc`.

Example

```go
escaped := sqlc.EscapeForLike(keyword, sqlc.DefaultLikeEscapeChar)
pattern := sqlc.WrapContainsLikePattern(escaped)
```

Purpose:

- Prevent LIKE injection
- Standardize search patterns

Keyword search is executed with OR conditions (ILIKE ANY).

## Deleted State Control

Filtering by deleted state is handled by branching in Go and calling dedicated queries.

```go
switch {
case filter.Active == nil:
    // all
case *filter.Active:
    // active
case !*filter.Active:
    // deleted
}
```

With this design:

- Prevent SQL complexity
- Maintain index efficiency
- Improve readability

## Row → Return Type Conversion

Row structs returned by sqlc are **Infrastructure-specific types**.

However, in this project, type conversions are applied at generation time using sqlc override:

- nullable → pointer type
- UUID → `pkg/uuid` type

Therefore, QueryService can pass generated types directly to Domain constructors or DTOs with minimal additional conversion.

```go
u, err := user.New(
    row.ID,
    row.FirstName,
    row.LastName,
    ...
)
```

Important rules:

- Do not return sqlc Row directly to upper layers
- Convert to Domain Entity or DTO

## About UUID

In this project, sqlc override aligns UUID in DB and `pkg/uuid` used in Domain.

Therefore, explicit UUID conversion in QueryService is generally unnecessary.

```go
row.Users.ID // usable as-is
```

Use the `pkg/uuid` wrapper for UUID generation, comparison, and helper operations.

## About Nullable

Nullable values are handled as pointer types via sqlc override.

Therefore, additional conversion in QueryService is unnecessary.

```go
row.Users.Building   // *string
row.Users.DeletedAt  //*time.Time
```

## LoggingDBProvider

QueryService typically uses `loggingdb.DBProvider` to access the DB.

```go
db := gen.New(s.db.NewLoggingDB(ctx))
```

`loggingdb.DBProvider` provides:

- SQL logging
- Transparent switching between DB / Tx
- Context-based connection retrieval

QueryService is designed to **not be aware of DB connection state**.

## Direct driver Usage

If logging is unnecessary, non-logging DB access can be used.

```go
db := gen.New(s.db.NewDB(ctx))
```

Use cases:

- Suppress log noise in high-frequency processing
- Simple processing without logging
- Benchmarking or minimal path verification

Principles:

- Normally use `NewLoggingDB(ctx)`
- Use `NewDB(ctx)` only when there is a clear reason

## Error Normalization

PostgreSQL errors are normalized in `internal/infrastructure/rdb/postgres/pgerror`.

```go
return pgerror.NormalizeError(err)
```

Main mappings:

```mermaid
flowchart TB
    A["sql.ErrNoRows"] --> B["ErrNotFound"]
    C["unique violation"] --> D["ErrConflict"]
    E["connection error"] --> F["ErrUnavailable"]
    G["others"] --> H["ErrInternal"]
```

## Observability (Tracing)

QueryService uses:

`observability.LayerTracer`

```go
ctx, endSpan := s.tracer.Start(ctx)
defer endSpan()
```

QueryService is responsible only for:

- span start
- span end

### About span name

Span names are uniformly assigned by LayerTracer, so QueryService does not need to explicitly specify them.

### Design Intent

- Ensure tracing consistency
- Separate responsibilities across layers
- Eliminate direct dependency on OpenTelemetry

## DI (Dependency Injection) Mechanism (Query Service)

Query Service is created using **DI with Uber Fx**.  
Like Repository, it is implemented in the Infrastructure layer and injected into the Usecase layer interface.

### Overall Structure

Query Service is registered via `fx.Provide` and injected into Usecase.

```mermaid
flowchart TB
    Module["InfrastructureModule"]
    Provide["fx.Provide(userqs.New)"]
    IF["query.UserQueryService (interface)"]
    Usecase["Usecase"]

    Module --> Provide --> IF --> Usecase
```

### Role of internal/di/module/infrastructure.go

```go
func InfrastructureModule() fx.Option {
    return fx.Module("infrastructure",
        fx.Module("query_service",
            fx.Provide(
                userqs.New,
            ),
        ),
    )
}
```

- `fx.Provide`
  - Registers the Query Service constructor
- Return value is the **interface defined in the Usecase layer**
  - Example: `query.UserQueryService`

### Query Service Constructor Design

```go
func New(
    db loggingdb.DBProvider,
    tf observability.TracerFactory,
) query.UserQueryService {
    return &service{
        db:     db,
        tracer: tf.Infra(),
    }
}
```

Key points:

- Return value must be an **interface (Usecase definition)**
- All dependencies must be received as arguments (new is prohibited)
- External dependencies such as DB / Tracer are confined to Infrastructure

### DI Flow

```mermaid
flowchart TB
    Provide["fx.Provide(userqs.New)"]
    IF["query.UserQueryService"]
    Usecase["Usecase (dependency)"]

    Provide --> IF --> Usecase
```

On the Usecase side:

```go
type service struct {
    qs query.UserQueryService
}
```

is used to receive via interface.

### Difference from Repository (DI perspective)

||Repository|Query Service|
|---|---|---|
|interface definition location|domain|usecase|
|return type|domain.Repository|query.QueryService|
|purpose|persistence|search|

### Why return Usecase interface

- Query is a Usecase concern (per usecase)
- Not placed in Domain because it is not aggregate-based
- Allows flexible handling of search specification changes

### Rules for AI / Developers

- Query Service constructor must always be defined as `New`
- Return value must be the Usecase interface
- Do not new dependencies inside Query Service
- Register DI in `internal/di/module/infrastructure.go`

## QueryService Structure

QueryService has the following dependencies.

```go
type service struct {
    db     loggingdb.DBProvider
    tracer observability.LayerTracer
}
```

constructor

```go
func New(
    db loggingdb.DBProvider,
    tf observability.TracerFactory,
) query.UserQueryService {
    return &service{
        db:     db,
        tracer: tf.Infra(),
    }
}
```

## Difference from Repository

Repository and QueryService have different roles.

||Repository|QueryService|
|---|---|---|
|purpose|Aggregate persistence|search|
|operation|CRUD|search|
|responsibility|Aggregate unit|search-specific|
|placement|domain interface|usecase interface|

## Anti-Patterns

### 1. Writing search in Repository

Search processing should be implemented in QueryService, not Repository.

However, the following simple filters are allowed in Repository:

- Retrieval by ID / foreign key
- Simple condition filtering
- Count retrieval (COUNT)

More complex searches (multiple conditions, full-text search, etc.) should be implemented in QueryService.

### 2. Writing business logic

QueryService is **data retrieval only**.

NG

```go
if user.IsPremium() {
}
```

### 3. Returning sqlc Row

NG

```go
return rows
```

Always convert to Domain.

## Implementation Example

```go
// service is the implementation of QueryService.
// It is responsible for DB access and tracing.
type service struct {
    db     loggingdb.DBProvider
    tracer observability.LayerTracer
}

// New is the constructor for QueryService.
// All dependencies are injected externally; no new is performed internally.
func New(
    db loggingdb.DBProvider,
    tf observability.TracerFactory,
) query.UserQueryService {
    return &service{
        db:     db,
        tracer: tf.Infra(),
    }
}

// FindByFilter searches users based on keywords and deleted state.
// - Keywords are converted to LIKE patterns
// - Deleted state is branched in Go
// - SQL uses dedicated queries
func (s *service) FindByFilter(ctx context.Context, filter*query.UserSearchFilter, limit, offset int32) (query.UserSearchResults, error) {
    ctx, endSpan := s.tracer.Start(ctx)
    defer endSpan()

    // Convert keywords to LIKE search patterns
    tokens := make([]string, len(filter.Keywords))
    for i, kw := range filter.Keywords {
        escaped := sqlc.EscapeForLike(kw, sqlc.DefaultLikeEscapeChar)
        tokens[i] = sqlc.WrapContainsLikePattern(escaped)
    }

    // Acquire DB connection using loggingDB
    db := gen.New(s.db.NewLoggingDB(ctx))

    // Switch queries based on deleted state
    switch {
    case filter.Active == nil:
        return fetchSearchAll(ctx, db, &gen.SearchUsersParams{
            PatternsParam: tokens,
            LimitParam:    limit,
            OffsetParam:   offset,
        })
    case *filter.Active:
        return fetchSearchActive(ctx, db, &gen.SearchActiveUsersParams{
            PatternsParam: tokens,
            LimitParam:    limit,
            OffsetParam:   offset,
        })
    case !*filter.Active:
        return fetchSearchDeleted(ctx, db, &gen.SearchDeletedUsersParams{
            PatternsParam: tokens,
            LimitParam:    limit,
            OffsetParam:   offset,
        })
    default:
        panic("unreachable: invalid active")
    }
}

// fetchSearchAll is a helper function to search all users.
// It separates logic from QueryService and clarifies responsibilities.
func fetchSearchAll(
    ctx context.Context,
    db *gen.Queries,
    params*gen.SearchUsersParams,
) (query.UserSearchResults, error) {
    rows, err := db.SearchUsers(ctx, params)
    if err != nil {
        return nil, pgerror.NormalizeError(err)
    }

    // Row → DTO conversion
    results := make(query.UserSearchResults, len(rows))
    for i, row := range rows {
        results[i] = &query.UserSearchResult{
            FirstName:      row.FirstName,
            LastName:       row.LastName,
            Email:          row.Email,
            Phone:          row.Phone,
            PostalCode:     row.PostalCode,
            PrefectureName: row.PrefectureName,
            City:           row.City,
            Street:         row.Street,
            Building:       row.Building,
            RegisteredAt:   row.CreatedAt,
            DeletedAt:      row.DeletedAt,
        }
    }

    return results, nil
}
```
