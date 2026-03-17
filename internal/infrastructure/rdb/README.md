# RDB Infrastructure Guide (`internal/infrastructure/rdb`)

[English](README.md) | 日本語

## Responsibility

`internal/infrastructure/rdb` is a **subsystem for using RDB (PostgreSQL)** within the Infrastructure layer.

This directory is responsible for:

- PostgreSQL connection management
- SQL execution (via sqlc)
- Repository / QueryService implementations
- SQL logging and tracing (Observability)
- PostgreSQL error normalization
- Conversion between DB nullable types and Go types

Domain / Usecase can use persistence and querying **without being aware of RDB implementation details**.

## RDB Architecture

This directory is structured into the following layers:

```txt
Usecase
   ↓
Repository / QueryService
   ↓
loggingdb
   ↓
driver
   ↓
PostgreSQL
```

Each layer has the following responsibilities:

|Layer|Responsibility|
|---|---|
|Repository|Aggregate persistence (implements Domain Repository Interface)|
|QueryService|Read-only queries (implements Usecase Interface)|
|loggingdb|SQL logging / tracing (Observability wrapper)|
|driver|DB connection and transaction management|
|PostgreSQL|Actual database|

Supporting components:

|Component|Responsibility|
|---|---|
|sqlc|Type-safe query execution generated from SQL|
|conv|Conversion between nullable types and Go types|
|pgerror|PostgreSQL error → application error normalization|
|testkit|RDB test utilities (real DB + rollback)|

## Directory Structure

```txt
internal/infrastructure/rdb

 ├ repository/        Repository implementations
 ├ query_service/     QueryService implementations
 ├ driver/            DB connection / transaction
 │   └ loggingdb/     SQL logging / tracing wrapper
 ├ sqlc/              sqlc generated code + SQL helpers
 ├ conv/              nullable conversion utilities
 ├ postgres/
 │   └ pgerror/       PostgreSQL error normalization
 └ testkit/           RDB test utilities
```

## Repository

Repository implements the **Domain Repository Interface**.

Responsibilities:

- Execute sqlc queries
- Convert Row → Domain entities
- Normalize DB errors

Important:

```txt
Repository must not contain business logic
```

See details:

[repository README](repository/README.ja.md)

## QueryService

QueryService provides **read-only query operations** such as search and listing.

While Repository handles aggregate persistence,  
QueryService is specialized for querying.

Responsibilities:

- Execute search queries
- Convert Row → Domain / DTO
- Normalize DB errors

Important:

```txt
Search logic must be implemented in QueryService, not Repository
```

See details:

[query_service README](query_service/README.ja.md)

## sqlc

`sqlc` generates Go code from SQL.

This directory provides:

- Generated query code
- LIKE search helpers
- Enum / state conversion helpers

Generated code is located at:

```txt
internal/infrastructure/rdb/sqlc/gen
```

See details:

[sqlc README](sqlc/README.md)

## conv

`conv` provides **conversion utilities between nullable types and Go pointer types**.

```txt
sql.NullString ⇔ *string
sql.NullTime   ⇔*time.Time
```

Used in Repository / QueryService implementations.

See details:

[conv README](conv/README.md)

## driver

`driver` is the **lowest layer providing DB connection and transaction management**.

Main features:

- DatabaseDriver abstraction
- Connection pool management
- Transaction management (`tx.Manager`)
- DBTX interface for sqlc

Important:

```txt
Transaction boundaries are managed by the Usecase layer
```

See details:

[driver README](driver/README.md)

## loggingdb

`loggingdb` is an **Observability wrapper that adds SQL logging and tracing**.

```txt
Repository / QueryService
   ↓
loggingdb
   ↓
driver
```

Main features:

- SQL logging
- OpenTelemetry spans
- Query latency measurement
- Slow query detection

Important:

```txt
loggingdb does not execute DB operations (pure wrapper)
```

See details:

[loggingdb README](driver/loggingdb/README.md)

## PostgreSQL Error Normalization

`postgres/pgerror` converts **PostgreSQL-specific errors into application errors**.

Repository / QueryService should use:

```go
pgerror.NormalizeError(err)
```

Typical mappings:

```txt
sql.ErrNoRows      → ErrNotFound
unique violation   → ErrConflict
connection error   → ErrUnavailable
others             → ErrInternal
```

See details:

[pgerror README](postgres/pgerror/README.md)

## testkit

`testkit` provides **utilities for testing with RDB**.

Main features:

- Test DB initialization
- LoggingDBProvider creation
- Transaction-based tests (automatic rollback)

Test characteristics:

```txt
Real DB
+
Transaction rollback
+
No parallel execution (Tx-based)
```

See details:

[testkit README](testkit/README.md)

## Design Principles

This RDB subsystem is based on the following principles:

### 1. Hiding DB Implementation

Domain / Usecase do not depend on:

```txt
sql
pgx
sql.DB
```

### 2. Separation of Responsibility (Repository / QueryService)

```txt
Write / persistence → Repository
Read / query        → QueryService
```

### 3. Centralized Transaction Boundary

```txt
Usecase manages transactions
Infrastructure does not start transactions
```

### 4. Error Normalization

PostgreSQL-specific errors are converted via:

```txt
pgerror
```

into application-level errors.

### 5. Type-Safe SQL

All SQL execution goes through:

```txt
sqlc
```

### 6. Observability

SQL logging and tracing are handled by:

```txt
loggingdb
```

### 7. Test Strategy (Integration-first)

Repository / QueryService tests use:

```txt
real DB + rollback
```

This is safely implemented using:

```txt
testkit
```
