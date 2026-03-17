# `conv` Package

English | [日本語](README.ja.md)

Overview:  
The `conv` package provides utility functions to convert between `database/sql` nullable types (`sql.NullString`, `sql.NullInt16`, `sql.NullInt64`, `sql.NullBool`, `sql.NullFloat64`, `sql.NullTime`) and Go pointer types (`*string`, `*int64`, etc.), as well as `googleUUID.NullUUID`.

This package is primarily used as an **Infrastructure-layer utility combined with sqlc-generated code**.

## Responsibility

This package acts as a bridge between **nullable database values** and **pointer values used in application code**.

Primary responsibilities:

- Convert nullable values retrieved from the database into pointer types that are easier to use in application code
- Convert pointer values created in the application into `sql.NullXxx` types for database writes
- Encapsulate NULL-handling logic into reusable functions to prevent duplication
- Clearly separate database-specific types from application-level types

This design allows **NULL handling to be centralized in a single place**.

## Supported Types

This package handles the following nullable types.

### database/sql

- `sql.NullString`
- `sql.NullInt16`
- `sql.NullInt64`
- `sql.NullBool`
- `sql.NullFloat64`
- `sql.NullTime`

### UUID

- `github.com/google/uuid.NullUUID`

UUID values are converted to and from the application's `pkg/uuid.UUID`.

## Conversion Patterns

For each supported type, **symmetric conversion APIs** are provided.

```txt
NullXxx  → *T
*T       → NullXxx
T        → NullXxx
```

Example (string):

```txt
StringPtrFromNull
NullStringFromPtr
NewNullString
```

This symmetric design allows the same rules to be used for both reads and writes.

## Usage (Behavior-Based)

### Read Path

Convert nullable values retrieved from the database into pointer types.

```go
name := conv.StringPtrFromNull(row.Name)
```

Conversion rules:

```txt
NULL  → nil
value → *value
```

Application code can check for NULL using `nil`.

### Write Path

Convert pointer values into nullable database types.

```go
row.Name = conv.NullStringFromPtr(namePtr)
```

Conversion rules:

```txt
nil   → NULL
value → Valid=true
```

### Creating Nullable Values

Convert non-null values into nullable types.

```go
row.Name = conv.NewNullString("alice")
```

## UUID Conversion

UUID helpers convert between `googleUUID.NullUUID` and the application's `uuid.UUID`.

```go
UUIDPtrFromNull
NullUUIDFromPtr
NewNullUUID
```

Note:

- `UUIDPtrFromNull` performs UUID parsing and therefore **returns an error**.

Example:

```go
u, err := conv.UUIDPtrFromNull(row.UserID)
if err != nil {
    return err
}
```

## Pointer Safety

Pointers generated from nullable types reference a **copy of the internal value**.

That means they behave like:

```txt
&ns.String
```

They **do not reference driver-managed memory**, so they are safe to use as normal pointer values.

## Relationship with sqlc

This package is primarily used as a **support utility for sqlc-generated code**.

Role:

```txt
sqlc generated code
        ↓
conv utilities
        ↓
application code
```

This ensures clear separation between:

- Database types
- Application types

## Testing

Unit tests are provided for each function (`nullable_test.go`).

The tests use `testify/require` and verify:

- `Valid` flags are correctly set
- NULL → nil conversions
- nil → NULL conversions

## Necessity

### Production

Not strictly required, but **strongly recommended**.

Reasons:

- Centralized NULL handling
- Reduced duplicate logic
- Lower risk of bugs

### Development / Testing

Recommended.

Reasons:

- Simplifies expression of NULL / non-NULL cases
- Improves readability of test code

## Impact if Not Used

Disabling this package usually does not cause direct runtime failures, but it may lead to:

- NULL handling scattered throughout the codebase
- Increased duplicated logic
- Higher risk of subtle bugs

## Notes

This package provides **conversion utilities only**.

Responsibilities it does **not** handle:

- Database query execution
- Transaction management
- Domain logic

Those should be implemented in:

- Repository
- Usecase

or other appropriate layers.

## Extension Rules

When adding support for a new nullable type, follow the naming convention below.

```txt
XxxPtrFromNull
NullXxxFromPtr
NewNullXxx
```

Example:

```txt
DecimalPtrFromNull
NullDecimalFromPtr
NewNullDecimal
```

Following this rule maintains API consistency.
