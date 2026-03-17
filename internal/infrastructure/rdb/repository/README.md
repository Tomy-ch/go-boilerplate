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

Instead, it generates the sqlc Querier using:

```go
db := gen.New(r.db.NewLoggingDB(ctx))
```

`loggingdb.DBProvider` provides the following features:

- SQL log output
- Transparent DB / Tx switching
- Context‑based connection resolution

This allows Repository implementations to remain **agnostic to the current DB connection state**.

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

## Repository Struct

Repository implementations depend on the following components:

- `driver.DatabaseDriver` — a pure DB connection driver without logging
- `loggingdb.DBProvider` — a DB connection provider with logging support

```go
type repository struct {
    db     loggingdb.DBProvider
    tracer observability.LayerTracer
}
```

Constructor:

```go
func New(
    db loggingdb.DBProvider,
    tf observability.TracerFactory,
) user.Repository {
    return &repository{
        db:     db,
        tracer: tf.Infra(),
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

Repository tests can call `t.Parallel()` and run concurrently.

However, the transaction manager created via:

```go
txm := testkit.NewTestTransactionManager(t)
```

**serializes transaction execution internally.**

The actual execution model is:

```txt
Test execution       → Parallel
Transaction execution → Serialized
```

Each test runs inside `WithinTx`:

```txt
BEGIN
 ↓
test
 ↓
ROLLBACK
```

Because transactions are serialized, even if multiple tests run at the same time,
**database state conflicts and cross‑test interference are prevented.**

This allows Repository tests to safely use `t.Parallel()`.

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

## Repository Anti‑Patterns

There are several **common incorrect implementation patterns** in the Repository layer.
These break architectural boundaries and **must not be implemented.**

### 1. Writing Business Logic

Repository is a **persistence layer**.
Business rules must not be implemented here.

Bad example:

```go
func (r *repository) Create(ctx context.Context, user *user.User) error {
    if user.IsPremium() {
        // ❌ business rule
    }
}
```

Correct responsibility:

```txt
Repository
 ├ Query execution
 ├ Row → Domain conversion
 └ Error normalization
```

Business rules belong to the **Domain / Usecase layers**.

---

### 2. Creating DTOs

Repository **must not create DTOs**.

Bad example:

```go
return UserDTO{
    Name: row.Users.Name,
}
```

Repository must return **Domain entities only**.

```go
return user.New(...)
```

DTO transformation belongs to **Usecase / Presenter layers**.

---

### 3. Returning sqlc Rows Directly

sqlc Row types are **Infrastructure‑specific types**.

Bad example:

```go
return rows
```

Rows must always be converted to Domain entities.

```go
u, err := user.New(...)
```

Reason:

```txt
Do not leak sqlc types to upper layers
```

---

### 4. Implementing QueryService in Repository

Repository represents an **aggregate persistence abstraction**.

Therefore methods like:

```txt
FindByKeyword
SearchUser
AggregateSearch
```

must **not** be implemented in Repository.

Search functionality must be implemented in a dedicated:

```txt
QueryService
```

layer.

---

### 5. Starting Transactions

Repository **must not start transactions**.

Bad example:

```go
tx, _ := db.Begin()
```

Transaction boundaries belong to the **Usecase layer**.

Repository executes queries via:

```go
gen.New(r.db.NewLoggingDB(ctx))
```

which transparently switches between:

```txt
Tx / DB
```

---

### 6. Referencing Controller Types

Repository must not depend on the **HTTP layer**.

Bad example:

```go
func (r *repository) Create(ctx echo.Context)
```

Repository must use **pure Go interfaces**.

```go
func (r *repository) Create(ctx context.Context, user *user.User)
```

---

### 7. Defining Domain Interfaces in Infrastructure

Repository interfaces must be defined in the **Domain layer**.

Bad example:

```txt
internal/infrastructure/repository/user_repository_interface.go
```

Correct location:

```txt
internal/domain/user/repository.go
```

Infrastructure should only **implement Domain interfaces**.

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

// repository name is fixed
type repository struct {
    db     loggingdb.DBProvider
    tracer observability.LayerTracer
}

// constructor name is fixed
func New(
    db loggingdb.DBProvider,
    tf observability.TracerFactory,
) user.Repository {
    return &repository{
        db:     db,
        tracer: tf.Infra(),
    }
}

func (r *repository) FindAll(ctx context.Context, limit, offset int32) (user.Users, error) {

    // Start and end span
    ctx, endSpan := r.tracer.Start(ctx)
    defer endSpan()

    // Using ResolveDriverWithLog automatically outputs SQL logs
    // If logging is unnecessary, ResolveDriver can be used instead
    db := gen.New(r.db.NewLoggingDB(ctx))

    // Call DML generated by sqlc
    rows, err := db.ListUsers(ctx, &gen.ListUsersParams{
        OffsetParam: offset,
        LimitParam:  limit,
    })

    if err != nil {
        // Normalize and return DB errors
        // Error classification is handled by the pgerror package
        return nil, pgerror.NormalizeError(err)
    }

    // Convert rows to Domain entities
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
