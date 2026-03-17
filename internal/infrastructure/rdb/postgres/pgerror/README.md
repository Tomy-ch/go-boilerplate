# pgerror Package

English | [日本語](README.ja.md)

Overview: **An Infrastructure-layer component that normalizes PostgreSQL-specific errors into application-wide errors and determines database connectivity failures. It acts as a translation layer that hides database-specific error semantics from upper layers.**

## Architectural Position

```txt
Repository
   ↓
pgerror
   ↓
PostgreSQL driver (pgx / pgconn)
```

`pgerror` is the layer responsible for **converting errors returned by the DB driver into application-level errors**.

By introducing this layer:

- Usecase
- Domain
- Handler

do not need to understand **PostgreSQL-specific error semantics**.

## Responsibility

This package normalizes PostgreSQL-specific errors into a common application error format.

Primary responsibilities:

- Convert PostgreSQL SQLSTATE codes into `apperror`
- Detect database connectivity failures
- Convert `sql.ErrNoRows` into `NotFound`
- Standardize error contracts between Infrastructure and Usecase layers

This enables **separating DB-specific error logic from application code**.

## Error Normalization

`NormalizeError` converts PostgreSQL errors into application-level errors.

```go
func NormalizeError(err error) error
```

Processing flow:

```txt
DB error
   ↓
NormalizeError
   ↓
AppError (apperror)
```

This function is recommended to be the **single normalization point for errors returned from the Infrastructure layer to the Usecase layer**.

## SQLSTATE Mapping

The following PostgreSQL SQLSTATE codes are mapped to application errors.

|SQLSTATE|Meaning|AppError|
|---|---|---|
|23505|unique violation|Conflict|
|23503|foreign key violation|InvalidArgument|
|23502|not null violation|InvalidArgument|
|23514|check violation|InvalidArgument|
|22001|string too long|InvalidArgument|
|22P02|invalid text representation|InvalidArgument|
|42501|insufficient privilege|PermissionDenied|

PostgreSQL errors that do not match these cases are mapped to:

```txt
Internal
```

## Special Handling

### sql.ErrNoRows

`sql.ErrNoRows` is not a PostgreSQL SQLSTATE and therefore handled separately.

```txt
sql.ErrNoRows
   ↓
NotFound
```

This allows the Repository layer to simply write:

```go
return NormalizeError(err)
```

and still correctly produce a `NotFound` error.

## Connection Failure Detection

`IsUnavailable` determines whether an error represents a database connectivity failure.

```go
func IsUnavailable(err error) bool
```

The following errors are treated as connectivity failures.

```txt
context.DeadlineExceeded
net.Error (timeout)
driver.ErrBadConn
PostgreSQL SQLSTATE 08XXX
```

This detection can be used for:

- retry mechanisms
- circuit breakers
- failover handling

## Error Wrapping

`pgerror` preserves the original database error while converting it to an application error.

The resulting structure is:

```txt
apperror
+
original error message
```

Errors are wrapped using `xerrors.Wrap`, allowing the system to retain:

- the application-level error classification
- the original database error

## Necessity

### Production

Required.

Reasons:

- Convert database constraint violations into correct HTTP responses
- Detect database connectivity failures
- Prevent exposure of raw database errors

This makes the layer essential for production systems.

### Development / Testing

Required.

Reasons:

- Hide database error semantics from tests
- Stabilize error handling in CI environments
- Improve readability of sqlc / repository tests

## Notes

### Use NormalizeError at a Single Boundary

`NormalizeError` should be applied **only once at the Infrastructure → Usecase boundary**.

Applying it multiple times may corrupt the error structure.

### PostgreSQL-Specific Behavior

SQLSTATE codes starting with `08XXX` represent PostgreSQL connection errors.

If the application migrates to another database (e.g., MySQL / TiDB), the implementation of:

```txt
pgerror
```

must be replaced accordingly.

### Avoid DB-Specific Logic in Upper Layers

Do not write DB-specific logic in Usecase or Domain layers such as:

```txt
SQLSTATE
pgconn
pgx
```

Database-specific error handling must remain inside the Infrastructure layer.
