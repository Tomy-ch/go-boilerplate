# driver

English | [日本語](README_ja.md)

Overview: **A foundational driver layer for RDB (PostgreSQL) connectivity. Provides connection management, transaction boundaries, and sqlc execution adapters.**

This package is the **lowest-level DB access infrastructure** in the Infrastructure layer.

Repository implementations access the database through this driver.

## Architectural Position

```txt
Usecase
   ↓
Repository
   ↓
Driver (this package)
   ↓
PostgreSQL
```

The driver acts as the **lowest-level adapter for RDB connections**.

## Responsibilities

This directory provides the following capabilities:

- **DatabaseDriver abstraction** wrapping `sql.DB`
- **Transaction management (`tx.Manager`)**
- **DBTX interface for sqlc execution**
- **Connection pool configuration**
- **Database connectivity check at startup (fail fast)**

With this design, the Repository layer does not directly depend on concrete database implementations such as:

```txt
sql.DB
sql.Tx
pgx driver
```

## DB Initialization

`NewDB()` initializes the database connection.

```go
func NewDB(...) (DatabaseDriver, error)
```

The process includes:

1. Initialize connection with `sql.Open()`
2. Configure connection pool settings

```txt
MaxOpenConns
MaxIdleConns
ConnMaxLifetime
ConnMaxIdleTime
```

1. Verify connectivity using `PingContext`

If the ping fails, an **error is returned at startup (fail fast)**.

## DatabaseDriver

`DatabaseDriver` is an interface that abstracts `sql.DB`.

```go
type DatabaseDriver interface {
    DBTX

    BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
    PingContext(ctx context.Context) error
    Close() error
}
```

Purpose:

- Avoid direct dependency on `sql.DB`
- Allow mocking during tests
- Abstract transaction initialization

The implementation is provided by `dbDriver`.

## DBTX

`DBTX` is the **minimal interface required by sqlc**.

```go
type DBTX interface {
    ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
    PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
    QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
    QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}
```

This interface allows sqlc to execute queries using either:

```txt
*sql.DB
*sql.Tx
```

without changing the generated code.

## Transaction Transparent Layer

`connection.go` provides a **transaction-transparent adapter** via `New()`.

```go
func New(ctx context.Context, db DatabaseDriver) DBTX
```

Behavior:

```txt
If Tx exists in context
    ↓
return *sql.Tx

If Tx does not exist
    ↓
return DatabaseDriver
```

This allows the Repository layer to execute queries without worrying about whether it is using:

```txt
DB
Tx
```

## Transaction Management

`tx.Manager` provides transaction boundaries for the Usecase layer.

```go
err := tx.Do(ctx, func(ctx context.Context) error {
    ...
})
```

Internally it performs:

1. Check if a Tx exists in context
2. If present → **reuse existing transaction**
3. If absent → **start a new transaction**
4. Execute the function
5. Success → commit
6. Error → rollback

This design allows **safe handling of nested transactions**.

## Important Notes

### Always propagate Context

Transactions are stored in:

```txt
context.Context
```

Therefore you must always propagate:

```txt
ctx
```

to lower layers.

### Repository must use `driver.New()`

In the Repository layer:

```go
driver.New(ctx, db)
```

should be used to obtain `DBTX`.

This enables transparent switching between:

```txt
Tx
DB
```

## Necessity

### Production

Required.

Reason:

- DB connection management
- Transaction boundaries
- sqlc query execution

all depend on this layer.

### Development / Testing

Recommended.

Reason:

- `DatabaseDriver` is an interface
- Allows mocking for testing
