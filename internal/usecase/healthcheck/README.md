# healthcheck

English | [日本語](README.ja.md)

Usecase for reporting service health: it captures the application time from the
`Clock` boundary and probes the database, returning a single health DTO.

## Usecase — `Usecase`

`New(dbSystemQuery, tracerFactory, clock) Usecase`

- `CheckHealth(ctx) (*DTO, error)` reads the current time via `clock.Now()`, then
  calls `DBSystemQuery.CheckDBHealth(ctx)`. On success it returns a `*DTO` with
  `Status = Ok`, `ApplicationTime`, and the `DBHealthCheck` result. On a DB error
  it returns `nil` — the DTO must not be dereferenced.

`DTO` carries `Status`, `ApplicationTime`, and `DBHealthCheck` (`query.DBHealth`).
The status constants are `Ok`, `Degraded`, and `Unhealthy`; the current
implementation only emits `Ok` on success (a DB error surfaces as the returned
error, not as a `Degraded` / `Unhealthy` DTO).

## DB probe — `healthcheck/query`

The `query` subpackage is a thin leaf boundary: `DBSystemQuery` with a single
`CheckDBHealth(ctx) (DBHealth, error)`, where `DBHealth` reports `Ready`,
`RespondedAt`, and `Latency`. The concrete implementation lives in
`internal/infrastructure/rdb/system_cqrs/healthcheck/` and runs a lightweight
`SELECT 1` liveness probe (`database/dml/system_cqrs/health_check/`).

## Layout

| Concern | Path |
| --- | --- |
| usecase | `internal/usecase/healthcheck/` (this package) |
| DB probe boundary | `internal/usecase/healthcheck/query/` (`DBSystemQuery`) |
| clock boundary | `internal/usecase/boundary/clock/` (`Clock`) |
| infrastructure | `internal/infrastructure/rdb/system_cqrs/healthcheck/` |
| sqlc DML | `database/dml/system_cqrs/health_check/` |
