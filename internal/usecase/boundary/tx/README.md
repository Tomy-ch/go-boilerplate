# tx

English | [日本語](README.ja.md)

Provides a `Manager` interface for transaction boundary management and a generic helper for returning values from transactions.

## Design Intent

- Make Usecase aware of "the existence of transactions" without exposing DB details
- Completely hide DB driver dependencies (pgx, sql.Tx, etc.)
- Transaction boundaries are a Usecase responsibility — Infrastructure does not start transactions

## Implementation

`internal/infrastructure/rdb/driver/` provides the concrete implementation using pgx transactions.

## Notes

- `Manager.Do` supports nested calls — reuses existing transaction if one exists in context
- `DoWithResult` wraps `Manager.Do` to extract a typed return value
- Transaction scope should be kept minimal to avoid unnecessary locks
- **Retry on contention (H1)**: `Do` retries the whole transaction a bounded number of times on `serialization_failure` (40001) / `deadlock_detected` (40P01). Because `fn` may run **up to N times**, it must be **idempotent for non-DB side effects** (caller's responsibility). The recommended pattern is to write external side effects (event publish, email, ...) as **outbox rows within the same transaction**, so a rolled-back attempt discards them too and only the committed attempt's rows are delivered — making retries safe by construction. The nested path (reusing an existing transaction) is **not** retried and runs once.
