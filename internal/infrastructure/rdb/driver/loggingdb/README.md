# loggingdb

English | [日本語](README.ja.md)

Overview: **An observability wrapper layer that adds SQL execution logging and trace information to DB access. Actual query execution is delegated to the driver layer, while loggingdb adds log formatting and tracing integration.**

`loggingdb` is an **observability adapter positioned above the driver layer**.

## Architectural Position

```txt
Usecase
   ↓
Repository
   ↓
loggingdb
   ↓
driver
   ↓
PostgreSQL
```

`loggingdb` **does not perform database execution itself.**

It wraps `driver.DBTX` and adds the following behavior during SQL execution:

- SQL logging
- OpenTelemetry trace integration
- Query execution latency measurement
- Slow query detection

## Responsibility

This directory provides **observability for SQL execution**.

Primary responsibilities:

- Provide a logging wrapper for `driver.DBTX`
- Emit SQL execution logs
- Create OpenTelemetry spans
- Measure query execution latency
- Detect slow queries
- Structure logging fields

With this design, upper layers (`repository / usecase / handler`) do not need to handle:

- logging implementation
- tracing implementation

## DBTX Wrapper

The core implementation of loggingdb is a **DBTX wrapper**.

```txt
driver.DBTX
     ↓ wrap
loggingdb (dbWithLogging)
     ↓
SQL logging + tracing
```

`dbWithLogging` wraps `driver.DBTX` and performs the following steps around SQL execution:

1. Start an OpenTelemetry span
2. Execute the SQL query
3. Measure execution time
4. Emit SQL logs
5. Evaluate error conditions

## SQL Log Contents

The emitted logs include the following fields:

```txt
Query
Args
Latency
Error
TraceID
SpanID
ParentSpanID
```

This enables tracing both:

- API requests
- database queries

within the **same trace context**.

## Slow Query

`loggingdb` automatically detects slow queries.

The threshold is defined by:

```go
DBConfig().SlowQueryWarnThreshold()
```

The log level is determined using the following rules:

```txt
Error 발생 → ERROR
slow query → WARN
normal query → INFO
```

## Provider

`DBProvider` is a **DI adapter** that aggregates dependencies required by loggingdb.

Provided dependencies:

- `DatabaseDriver`
- `Logger`
- `LogFieldBuilder`
- `DatabaseConfig`
- `LayerTracer`

Through this design, loggingdb avoids directly depending on:

- logging implementation
- tracing implementation

and instead receives them via DI.

## Necessity

### Production

Recommended.

Reasons:

- detection of slow queries
- investigation of DB errors
- correlation between API requests and DB queries

These capabilities are **highly valuable for operational monitoring**.

However, in extremely high-throughput environments you may consider:

```txt
sampling
slow query only logging
```

to reduce log volume.

### Development / Testing

Strongly recommended.

Reasons:

- verify issued SQL queries
- debug sqlc query behavior
- assist database-related testing

Logging during development is especially useful for debugging.

## Notes

### loggingdb does not perform DB I/O

`loggingdb` is a **pure wrapper**.

All actual SQL execution is delegated to the driver layer.

```txt
loggingdb
   ↓ delegate
driver
```

### Always propagate Context

Trace information is stored in:

```txt
context.Context
```

Therefore you must always propagate:

```txt
ctx
```

to lower layers.

### Log volume with high query counts

Logs are emitted for each DB operation:

```txt
ExecContext
QueryContext
QueryRowContext
```

If a process performs a large number of queries, the resulting log volume may increase significantly.
