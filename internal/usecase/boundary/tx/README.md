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
