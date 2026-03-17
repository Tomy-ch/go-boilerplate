# Query Service Implementation Guide

## Role

Query Service is a layer that provides **read-only queries such as search and listing**.

While Repository represents **aggregate persistence abstraction**,  
Query Service provides **query-specific search operations**.

```txt
Controller
   ↓
Usecase
   ↓
QueryService
   ↓
sqlc
   ↓
Database
```

Query Service has the following responsibilities:

1. Execute SQL queries
2. Convert Row → Domain
3. Normalize DB errors

Query Service **must not contain business logic.**

## Architecture Position

QueryService implementations are placed in:

```txt
internal/infrastructure/rdb/query_service/<aggregate>/
```

Example

```txt
query_service/
 └ user/
     └ user_query_service.go
```

QueryService interfaces are defined in the **Usecase layer**.

```txt
internal/usecase/<aggregate>/query
```

Example

```txt
internal/usecase/user/query/user_query_service.go
```

Infrastructure only **implements this interface**.

## QueryService Responsibility

QueryService is responsible for **search-oriented data retrieval**.

```txt
Query
 ↓
sqlc
 ↓
Row
 ↓
Domain Entity or DTO
 ↓
return
```

QueryService does NOT:

- Implement business rules
- Contain usecase logic
- Handle controller concerns
- Manage transactions

## sqlc Usage

QueryService uses **sqlc generated queries**.

```go
rows, err := db.ListUsersByKeywords(ctx, ...)
```

sqlc provides:

- Type-safe SQL execution
- Compile-time SQL validation

Generated code:

```txt
internal/infrastructure/rdb/sqlc/gen
```

## LIKE Search Helper

For keyword search, use helpers from:

```txt
internal/infrastructure/rdb/sqlc
```

Example

```go
escaped := sqlc.EscapeForLike(keyword, sqlc.DefaultLikeEscapeChar)
pattern := sqlc.WrapContainsLikePattern(escaped)
```

Purpose:

- Prevent LIKE injection
- Standardize search patterns

## Deleted State Handling

Deleted-state filtering uses:

```go
DeletedState: sqlc.BoolPtrToDeletedState(active)
```

This enables:

```txt
active / inactive / all
```

state control.

## Row → Return Type Conversion

sqlc Rows are **Infrastructure types**.

QueryService must always convert them into **return types (Domain Entity or DTO)**.

```go
user, err := user.New(
    uuid.FromPrimitive(row.Users.ID),
    ...,
)
```

Important rule:

```txt
Do not return sqlc Row to upper layers
```

## UUID Conversion

DB uses primitive UUID.

Domain uses:

```txt
pkg/uuid.UUID
```

Conversion:

```go
uuid.FromPrimitive(row.ID)
```

## Nullable Conversion

Nullable DB values are converted using:

```txt
internal/infrastructure/rdb/conv
```

Example

```go
conv.StringPtrFromNull(row.Users.Building)
conv.TimePtrFromNull(row.Users.DeletedAt)
```

## LoggingDBProvider

QueryService does not directly use DB driver.

```go
db := gen.New(s.db.NewLoggingDB(ctx))
```

`loggingdb.DBProvider` provides:

- SQL logging
- DB / Tx switching
- Context binding

## Error Normalization

PostgreSQL errors are normalized via:

```txt
internal/infrastructure/rdb/postgres/pgerror
```

```go
return pgerror.NormalizeError(err)
```

Main mappings:

```txt
sql.ErrNoRows      → ErrNotFound
unique violation   → ErrConflict
connection error   → ErrUnavailable
others             → ErrInternal
```

## Observability (Tracing)

QueryService uses:

```txt
observability.LayerTracer
```

```go
ctx, endSpan := s.tracer.Start(ctx)
defer endSpan()
```

QueryService handles only:

```txt
Span start / end
```

## QueryService Structure

QueryService has the following dependencies:

```go
type service struct {
    db     loggingdb.DBProvider
    tracer observability.LayerTracer
}
```

Constructor:

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
|Purpose|Aggregate persistence|Search|
|Operation|CRUD|Search|
|Responsibility|Aggregate-based|Query-specific|
|Placement|domain interface|usecase interface|

## Anti-Patterns

### 1. Writing search logic in Repository

Search must not be implemented in Repository.

NG

```go
func (r *repository) FindByKeyword(...)
```

Search belongs to:

```txt
QueryService
```

---

### 2. Writing business logic

QueryService is **data retrieval only**.

NG

```go
if user.IsPremium() {
}
```

---

### 3. Returning sqlc Row

NG

```go
return rows
```

Always convert to Domain.

## Implementation Example

```go
// service name is fixed
type service struct {
  db     loggingdb.DBProvider
  tracer observability.LayerTracer
}

// New function name is fixed
func New(
  db loggingdb.DBProvider,
  tf observability.TracerFactory,
) query.UserQueryService {
  return &service{
    db:     db,
    tracer: tf.Infra(),
  }
}

func (s *service) FindByKeyword(ctx context.Context, keywords []string, active *bool, limit, offset int32) (user.Users, error) {
    // Start / end span
    ctx, endSpan := s.tracer.Start(ctx)
    defer endSpan()

    // Pre-processing for QueryService usage
    tokens := make([]string, len(keywords))
    for i, kw := range keywords {
        escaped := sqlc.EscapeForLike(kw, sqlc.DefaultLikeEscapeChar)
        tokens[i] = sqlc.WrapContainsLikePattern(escaped)
    }

    // Use logging-enabled DB driver
    // If not needed, use driver.ResolveDriver(ctx, r.db)
    db := gen.New(s.db.NewLoggingDB(ctx))

    rows, err := db.ListUsersByKeywords(ctx, &gen.ListUsersByKeywordsParams{
        PatternsParam: tokens,
        DeletedState:  sqlc.BoolPtrToDeletedState(active),
        LimitParam:    limit,
        OffsetParam:   offset,
    })
    if err != nil {
        // Normalize error before returning
        // Error classification is handled in pgerror package
        return nil, pgerror.NormalizeError(err)
    }

    // Map to Domain entity or DTO
    users := make(user.Users, len(rows))
    for i, row := range rows {
        u, err := user.New(
            uuid.FromPrimitive(row.Users.ID),
            row.Users.FirstName,
            row.Users.LastName,
            row.Users.PasswordHash,
            row.Users.Email,
            row.Users.Phone,
            uuid.FromPrimitive(row.Users.PrefectureID),
            row.Users.City,
            row.Users.Street,
            conv.StringPtrFromNull(row.Users.Building),
            row.Users.PostalCode,
            row.Users.CreatedAt,
            row.Users.UpdatedAt,
            conv.TimePtrFromNull(row.Users.DeletedAt),
        )
        if err != nil {
            return nil, err
        }
        users[i] = u
    }

    return users, nil
}
```
