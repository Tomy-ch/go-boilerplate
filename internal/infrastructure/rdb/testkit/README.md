## `testkit` Package

English | [日本語](README.ja.md)

Overview: **A utility package that provides helpers for tests using an RDB.**

It is primarily used for:

- Easily creating a test `DatabaseDriver`
- Initializing Infrastructure including `LoggingDBProvider`
- Running tests inside transactions that are always rolled back

This package is a **helper tool to simplify writing Repository and Infrastructure tests**.

## Purpose

Tests that use an RDB often face the following problems:

- DB initialization code becomes repetitive and lengthy
- Transaction management becomes complicated
- Test data cleanup is required after each test

`testkit` solves these problems by providing:

- Test DB initialization
- LoggingDBProvider creation
- Automatic rollback transactions

## Architectural Position

```txt
Repository Test
     ↓
 testkit
     ↓
 driver / loggingdb
     ↓
 PostgreSQL
```

`testkit` acts as a **test-only Infrastructure helper layer**.

## Provided APIs

### NewTestDB

```go
func NewTestDB(t *testing.T) driver.DatabaseDriver
```

Creates a test `DatabaseDriver`.

---

### NewTestLoggingProvider

```go
func NewTestLoggingProvider(t *testing.T) loggingdb.DBProvider
```

Creates a `LoggingDBProvider`.

Primary use cases:

- Repository tests
- QueryService tests

Internally, it performs:

```txt
MockConfigForTest
↓
DatabaseConfig
↓
Logging / Tracer initialization
↓
loggingdb.NewLoggingDBProvider
```

---

### NewTestTransactionManager

```go
func NewTestTransactionManager(t *testing.T) TransactionRunner
```

Creates a transaction manager for testing.

Internally it uses:

```txt
config.MockConfigForTest
↓
driver.NewTransactionManager
```

---

## Transaction Execution

### TransactionRunner

```go
type TransactionRunner interface {
    WithinTx(fn func(ctx context.Context))
}
```

---

### WithinTx

```go
func (t *testTxManager) WithinTx(fn func(ctx context.Context))
```

Executes the provided function **inside a transaction**.

Execution flow:

```txt
Transaction Begin
↓
fn(ctx) execution
↓
return rollbackForTestError
↓
Rollback
```

Internally, it uses `tx.Manager.Do`:

```txt
Do(fn)
↓
returning an error triggers rollback
```

---

## Rollback Mechanism

`WithinTx` performs rollback using the following mechanism:

```txt
execute fn
↓
return rollbackForTestError
↓
tx.Manager performs rollback
```

This special error is defined as:

```go
var rollbackForTestError = xerrors.New("rollback for test")
```

This error is treated as **success in tests**.

---

## Parallel Execution

Repository tests can be executed in parallel using `t.Parallel()`.

However, transactions are serialized internally.

```txt
Test execution     → Parallel
Transactions       → Serialized
```

This behavior is guaranteed by:

```go
txLock sync.Mutex
```

```go
txLock.Lock()
defer txLock.Unlock()
```

This ensures:

- Prevention of DB state conflicts
- Isolation between tests

---

## DB Instance

The test DB is managed as a singleton.

```go
var (
    testDB driver.DatabaseDriver
    dbOnce sync.Once
)
```

```txt
A single DB instance is shared within the process
```

This enables:

- Reduced connection cost
- Faster test execution

---

## Usage Examples

### Transaction-Based Test

```go
txm := testkit.NewTestTransactionManager(t)

txm.WithinTx(func(ctx context.Context) {
    repo.Create(ctx, ...)
})
```

---

### Repository Test

```go
provider := testkit.NewTestLoggingProvider(t)

repo := repository.NewRepository(provider)
```

---

## Test Design Policy

Using `testkit` enables the following design:

```txt
Repository Test
↓
Real DB
↓
Transaction Rollback
```

In other words:

```txt
Use a real database
+
Restore the state after the test
```

This enables **safe integration tests**.

---

## Notes

### Connects to a Real Database

This package connects to a **real PostgreSQL instance**.

Therefore you must configure:

- a test database
- a CI database

---

### Design of WithinTx

`WithinTx` does not return a value from the function.

```go
fn(ctx)
return rollbackForTestError
```

Therefore, assertions in tests should be written using:

```go
require / assert
```

---

### Transactions Always Roll Back

When using `WithinTx`, the transaction will **always be rolled back**.

Therefore it cannot be used for tests that require:

```txt
persistent data
```

---

## Summary

`testkit` provides:

```txt
Real DB
+
Automatic rollback
+
Parallel safety
```

as a **testing infrastructure package**.
