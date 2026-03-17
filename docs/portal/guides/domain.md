# Domain Layer (`internal/domain`) Guide

## Role in Onion Architecture

- The **core of the business**. It represents **essential rules** such as entities, value objects, domain services, and domain events.
- It has **no concern for external systems** (HTTP / DB / UI) and defines behavior using **pure domain models and language**.
- It is the **most stable layer**. The design assumes that **as long as this layer does not break, the product remains maintainable**.

## Role in this boilerplate

- Place **Entity / ValueObject / DomainService / Repository (interface)** under:

```txt
internal/domain/<bounded-context>/<aggregate>/
```

Example: `internal/domain/user/`

```txt
user_domain.go       ← Aggregate Root (User)
value.go             ← Value Objects
service.go           ← Domain Service
user_repository.go   ← Repository Interface
error.go             ← Domain errors
constant.go          ← Validation constants
```

- Business rules should be expressed using **pure functions without side effects** whenever possible.  
  I/O, time retrieval, and random generation should be **injected as arguments**.

- State changes must be done through **entity methods**, and must not access external resources.

- Types should follow an **effectively immutable** approach.

  - private fields + getters
  - defensive copy (`ptr.Copy`)
  - setters are prohibited
  - state changes occur through **behavior methods**

- Dependencies should be **injected through constructors**.

- External libraries must not be imported directly; they should be used **through pkg wrappers**.

Examples:

- UUID → `pkg/uuid`
- Decimal → `pkg/decimal`
- Error → `pkg/xerrors`

## Domain boundaries

The Domain layer is responsible for **expressing business rules and state transitions**.

Domain responsibilities:

- Invariants
- State transitions
- Value consistency
- Business rules

Domain is **not responsible for**:

- Search specifications
- DB optimizations
- SQL structures
- External API calls
- Aggregation processing

These belong to other layers:

- Usecase
- QueryService
- ReadModel

Repositories should **only provide persistence abstractions**.

Simple queries are acceptable in practice.

Allowed examples:

- `FindByXXX`
- `FindAll`
- `CountByXXX`

## Implementation notes

### Naming / structure

- Struct names should match the **domain concept**
- Slice types may be defined when needed

```go
type Users []*User
```

- Repository interface name should be `Repository`
- Package name should be the domain name
- Constructor name should be `New`

### Do not set fields outside the constructor

- Invariants must be guaranteed in `New(...)`
- setters are prohibited
- state transitions occur through **behavior methods**

### Access via getters

- Fields must not be exported

```txt
ID()
FirstName()
Email()
```

- Pointer types must use **defensive copy**

```go
ptr.Copy(...)
```

### Do not use struct tags

The Domain layer must not know the outside world.

Forbidden:

```txt
json
db
validate
```

These belong to DTO or Infrastructure.

### Handling time and ID

- `time.Now()` must not be used in Domain
- UUID generation must not be done in Domain

Generation must happen in:

- Controller
- Usecase

Domain only receives **typed values**

```txt
uuid.UUID
time.Time
```

### Validation

#### Format validation

Prefer **Value Objects**

Example:

```go
NewEmail(...)
```

Lightweight domains may allow primitive types.

#### Boundary validation

Boundary values should be defined in `constant.go`

```txt
minLength
maxEmailLength
```

#### Errors

Errors must be **specific errors**

```txt
ErrInvalidEmail
ErrInvalidPostalCode
```

Abstract errors should not be returned directly.

```go
if !stringkit.InRange(email, minLength, maxEmailLength) {
    return nil, xerrors.Wrap(ErrInvalidEmail, ...)
}
```

### Domain Invariants

Entities must **always satisfy invariants**.

Examples:

- `updatedAt >= createdAt`
- `deletedAt >= createdAt`
- `deletedAt >= updatedAt`

Invariant enforcement points:

- `New(...)`
- state transition methods

Usecase / Repository **must not enforce invariants**.

## Aggregate Design

In this boilerplate, **Aggregate is the design unit**.

```txt
internal/domain/<bounded-context>/<aggregate>/
```

### Aggregate Root

Each aggregate has **one Root**.

Responsibilities:

- integrity guarantee
- external operation entry point
- persistence target

```go
type User struct {
    id uuid.UUID
}
```

Repositories are defined **for the root**

```go
type Repository interface {
    CreateUser(ctx context.Context, user *User) error
}
```

### Aggregate integrity

All changes must go **through the root**

```txt
Usecase → Aggregate Root → Entity
```

### Aggregate boundary

Aggregates must remain **small**

Principle:

```txt
1 Aggregate = 1 Transaction Boundary
```

Avoid:

- huge aggregates
- direct DB schema mapping
- tightly coupled models

### Cross-aggregate references

References must be **by ID only**

```go
type Order struct {
    userID uuid.UUID
}
```

Forbidden:

```go
type Order struct {
    user *User
}
```

### Multi-aggregate rules

Rules across aggregates belong to:

- Domain Service
- Usecase

Example:

```txt
User cancellation → stop Subscription
```

### Query and Aggregate

Aggregates are **Write Models**

They must not handle:

- aggregation
- reporting
- complex search
- GROUP BY

These belong to:

- QueryService
- ReadModel

## Dependency inversion for infrastructure

Repositories are **persistence abstractions**

```go
type Repository interface {
    FindAll(ctx context.Context, limit, offset int32) (Users, error)
    CreateUser(ctx context.Context, user *User) error
    CountByActive(ctx context.Context, active*bool) (int64, error)
}
```

Implementation:

```txt
internal/infrastructure/persistence/postgres/
```

Mapping is done using `sqlc`.

### Allowed repository methods

- `FindAll`
- `FindByXXX`
- `CountByXXX`

Expected operations:

```txt
SELECT / WHERE / JOIN
```

### Methods that must not be in repository

- GROUP BY
- SUM / AVG
- WITH clauses
- cross-boundary joins

Place them in:

- Usecase
- QueryService
- ReadModel

## Callable layers

Called from:

- Usecase

Domain **must not call other layers**.

Cross-aggregate rules:

- Domain Service

Exception:

```txt
read-only aggregate reference
```

## Testing strategy

Domain tests are **pure unit tests**

Forbidden dependencies:

- DB
- HTTP
- environment variables
- time.Now()

### Constructor validation

`New(...)` must guarantee **invariants**

Examples:

- zero ID
- boundary values
- time consistency

```go
require.ErrorIs(t, err, ErrInvalidEmail)
```

### Getter contract tests

```txt
ID()
FirstName()
Email()
CreatedAt()
UpdatedAt()
```

### Immutable guarantee tests

Pointer fields:

```txt
*string
*time.Time
```

Verification:

1. mutate constructor pointer
2. mutate getter result

Entity must remain unchanged.

### Domain behavior tests

Example:

```go
FullName()
```

Expected:

```txt
firstName + " " + lastName
```

### Error classification tests

```go
require.ErrorIs(t, err, ErrInvalidEmail)
```

### Test design policies

#### Deterministic

```go
baseTime := time.Date(2025,1,1,0,0,0,0,time.UTC)
```

#### Parallel execution

```go
t.Parallel()
```

#### Fail fast

```go
require.NoError(t, err)
```

### Test fixture

Fixture usage is recommended.

Reasons:

- reduce duplication
- guarantee invariants
- simplify tests

```go
func newTestUser(t *testing.T)*User {
    baseTime := time.Date(2025,1,1,0,0,0,0,time.UTC)

    id := uuid.NewTestFromSalt(t,"user")
    prefectureID := uuid.NewTestFromSalt(t,"prefecture")

    user, err := New(
        id,
        "John",
        "Doe",
        "hashed_password",
        "john@example.com",
        "1234567890",
        prefectureID,
        "Shibuya",
        "1-2-3",
        nil,
        "1500001",
        baseTime,
        baseTime.Add(time.Hour),
        nil,
    )

    require.NoError(t, err)
    return user
}
```

### Invariant preservation tests

State transition test:

```go
Before → Behavior → After
```

Example:

```go
func TestUser_UpdateEmail(t *testing.T) {
    user := newTestUser(t)

    updatedAt := user.UpdatedAt().Add(time.Hour)

    err := user.UpdateEmail("new@example.com", updatedAt)

    require.NoError(t, err)
    require.Equal(t, "new@example.com", user.Email())
}
```

Invalid case:

```go
require.ErrorIs(t, err, ErrInvalidUpdatedAt)
```

## Do / Don’t

### Do

- guarantee integrity in constructor
- state transitions via behavior methods
- ensure consistency with Value Objects
- abstract persistence via Repository
- table-driven tests

### Don’t

Forbidden:

- http.*
- echo.*
- sqlc types
- json tags
- setters
- DB-driven design
- time.Now() in Domain
