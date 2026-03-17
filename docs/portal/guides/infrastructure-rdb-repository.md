# Repository Implementation Guide

[English](README.md) | 日本語

## Responsibility

Repository is the layer that **implements the Domain persistence abstraction (Repository Interface) in Infrastructure**.

The responsibilities of this layer are strictly limited to the following three:

1. Executing queries using sqlc
2. Converting DB Rows → Domain entities
3. Normalizing DB errors

Repository **must not contain business logic**.

```txt
Usecase
   ↓
Repository (Infra)
   ↓
sqlc
   ↓
Database
```

Repository simply **implements the Repository Interface defined by the Domain**.

## Architectural Position

Repository implementations should be placed in:

```txt
internal/infrastructure/rdb/repository/<aggregate>/
```

Example:

```txt
repository/
 ├ user/
 │   └ user_repository.go
 └ prefecture/
     └ prefecture_repository.go
```

The Repository Interface must be placed in the **Domain layer**.

```txt
internal/domain/<aggregate>/repository.go
```

Infrastructure **only implements** this interface.

## Repository Method Responsibility

Repository methods perform only the following processing:

```txt
Query
 ↓
sqlc
 ↓
Row
 ↓
Domain Entity
 ↓
return
```

Repository must not implement:

- Business rules
- Aggregation logic
- DTO creation
- Usecase logic

These belong to the **Usecase / Domain layers**.

## Using sqlc

Repository must use **sqlc generated query code**.

```go
rows, err := db.ListUsers(ctx, &gen.ListUsersParams{...})
```

With sqlc:

- SQL execution becomes type-safe
- Queries are verified at compile time

Generated code is located at:

```txt
internal/infrastructure/rdb/sqlc/gen
```

## Row → Domain Conversion

Structures returned by sqlc are **Infrastructure-only types**.

Therefore Repository must always convert them to **Domain entities**.

```go
user, err := user.New(
    uuid.FromPrimitive(row.Users.ID),
    row.Users.FirstName,
    row.Users.LastName,
    ...
)
```

Important rules:

- Never return sqlc Rows to upper layers
- Always use Domain constructors

## Domain Constructor Errors

If Domain entity creation fails,  
the error **must be returned as-is**.

```go
user, err := user.New(...)
if err != nil {
    return nil, err
}
```

Reasons include:

- Domain invariant violation
- DB data inconsistency
- Migration mistakes

These must be treated as **Domain-layer responsibility**.

## UUID Conversion

The database uses primitive UUID values.

Domain uses `pkg/uuid.UUID`.

Therefore conversion is required.

```go
uuid.FromPrimitive(row.ID)
uuid.ToPrimitiveUniqueList(ids)
```

UUID operations must use:

```txt
pkg/uuid
```

wrapper utilities.

## Nullable Conversion

Nullable DB values are represented using `sql.Null*`.

Repository should convert them using:

```txt
internal/infrastructure/rdb/conv
```

Example:

```go
conv.StringPtrFromNull(row.Users.Building)
conv.NullStringFromPtr(user.Building())
```

This centralizes conversions such as:

```txt
sql.NullString ⇔ *string
sql.NullTime   ⇔*time.Time
```

## sqlc Helpers

Helper utilities for LIKE search and similar tasks are located in:

```txt
internal/infrastructure/rdb/sqlc
```

Example:

```go
escaped := sqlc.EscapeForLike(keyword, sqlc.DefaultLikeEscapeChar)
pattern := sqlc.WrapContainsLikePattern(escaped)
```

Deleted state example:

```go
DeletedState: sqlc.BoolPtrToDeletedState(active)
```

Purpose:

- Prevent LIKE injection
- Standardize search patterns
- Centralize state conversion

## LoggingDBProvider

Repository does not directly use `driver.DatabaseDriver`.

Instead it generates the sqlc Querier using:

```go
db := gen.New(r.provider.NewLoggingDB(ctx))
```

`loggingdb.DBProvider` provides:

- SQL logging
- Transparent DB / Tx switching
- Context-based connection resolution

Repository therefore **does not need to know the DB connection state**.

## Error Normalization

PostgreSQL errors are normalized via:

```txt
internal/infrastructure/rdb/postgres/pgerror
```

```go
return pgerror.NormalizeError(err)
```

Typical mappings:

```txt
sql.ErrNoRows      → ErrNotFound
unique violation   → ErrConflict
connection error   → ErrUnavailable
others             → ErrInternal
```

## Transactions

Transaction boundaries are **managed by the Usecase layer**.

Repository must never start transactions.

Queries are executed using:

```go
gen.New(provider.NewLoggingDB(ctx))
```

This transparently switches between:

```txt
Tx / DB
```

## Observability (Tracing)

Infrastructure uses:

```txt
observability.LayerTracer
```

```go
ctx, endSpan := r.tracer.Start(ctx)
defer endSpan()
```

Repository only knows about:

- span start
- span end

It does not depend directly on the OpenTelemetry SDK.

## Repository struct

Repository implementations use the following dependencies:

```go
type repository struct {
    db       driver.DatabaseDriver
    provider loggingdb.DBProvider
    tracer   observability.LayerTracer
}
```

Constructor:

```go
func New(
    db driver.DatabaseDriver,
    provider loggingdb.DBProvider,
    tf observability.TracerFactory,
) user.Repository {
    return &repository{
        db:       db,
        provider: provider,
        tracer:   tf.Infra(),
    }
}
```

## Test Strategy

Repository tests are implemented as **Infrastructure Integration Tests**.

Because Repository is responsible for SQL execution correctness,  
tests **must use a real database instead of mocks**.

Test structure:

```txt
Repository
   ↓
sqlc
   ↓
driver
   ↓
PostgreSQL
```

The entire layer is tested **using a real database**.

### Test Objectives

Repository tests verify:

- SQL queries execute correctly
- Row → Domain conversion works correctly
- Domain constructor errors propagate correctly
- PostgreSQL errors are normalized via `pgerror.NormalizeError`
- Pagination (limit / offset) works correctly

Repository tests **do not validate Domain logic**.

### Test Environment

Repository tests initialize the DB using `testkit`.

```go
db, provider := testkit.NewTestDBWithLoggingProvider(t)
```

This function provides:

- test DB connection
- loggingdb.DBProvider

### Transaction Tests

Each test runs **inside a transaction**.

```go
txm := testkit.NewTestTransactionManager(t)

txm.WithinTx(func(ctx context.Context) {
    // test logic
})
```

Internal flow:

```txt
BEGIN
 ↓
test
 ↓
ROLLBACK
```

Benefits:

- DB state remains clean
- test isolation is guaranteed

### Parallel Execution

Repository tests **must not run in parallel**.

Reason:

```txt
A shared database is used.
```

Concurrent execution may cause:

- data races
- unstable tests

Therefore the rule is:

```go
// do not use t.Parallel()
```

### Domain Error Verification

Repository uses Domain constructors for Row → Domain conversion.

If **invalid data exists in the DB**,  
Domain errors will occur.

Tests should verify these cases.

```go
require.ErrorIs(t, err, user.ErrInvalidLastName)
```

This validates cases such as:

```txt
DB data inconsistency
migration errors
Domain invariant violations
```

### Error Normalization in Tests

DB errors are converted using `pgerror.NormalizeError`.

Example mappings:

```txt
sql.ErrNoRows      → ErrNotFound
unique violation   → ErrConflict
connection error   → ErrUnavailable
others             → ErrInternal
```

Repository tests should verify **normalized results** such as:

```txt
ErrConflict
ErrNotFound
```

### Summary

Repository testing policy:

```txt
Repository
 ├ sqlc
 ├ driver
 ├ loggingdb
 └ PostgreSQL
```

The above stack is tested **using a real DB integration test**.

```txt
Domain          → Unit Test
Usecase         → Unit Test
Repository      → Integration Test
Controller      → Unit Test
Integration     → HTTP Test
```

This ensures verification of:

- SQL correctness
- Domain conversion
- error normalization

## Do / Don't

### Do

- Use sqlc generated code
- Convert Row → Domain
- Use conv for nullable conversion
- Use pgerror.NormalizeError
- Use LIKE helper utilities

### Don't

- Define Domain interfaces in Infrastructure
- Return sqlc types to upper layers
- Write business logic
- Reference Controller types
- Implement QueryService

## Minimal Snippet

```go
package user

type repository struct {
    db       driver.DatabaseDriver
    provider loggingdb.DBProvider
    tracer   observability.LayerTracer
}

func New(
    db driver.DatabaseDriver,
    provider loggingdb.DBProvider,
    tf observability.TracerFactory,
) user.Repository {
    return &repository{
        db:       db,
        provider: provider,
        tracer:   tf.Infra(),
    }
}

func (r *repository) FindAll(ctx context.Context, limit, offset int32) (user.Users, error) {
    ctx, endSpan := r.tracer.Start(ctx)
    defer endSpan()

    db := gen.New(r.provider.NewLoggingDB(ctx))

    rows, err := db.ListUsers(ctx, &gen.ListUsersParams{
        OffsetParam: offset,
        LimitParam:  limit,
    })

    if err != nil {
        return nil, pgerror.NormalizeError(err)
    }

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
