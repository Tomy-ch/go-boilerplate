# healthcheck

Usecase for reporting service health: it captures the application time from the
`Clock` boundary and probes the database, returning a single health DTO.

## Usecase — `Usecase`

`New(dbSystemQuery, tracerFactory, clock, probes) Usecase`

- `CheckHealth(ctx) (*DTO, error)` reads the current time via `clock.Now()`, then
  calls `DBSystemCqrs.CheckDBHealth(ctx)`. On a DB error it returns `nil` — the DTO
  must not be dereferenced. On success it runs every `Probe` and returns a `*DTO`
  with `ApplicationTime`, the `DBHealthCheck` result, one `DependencyStatus` per
  probe, and `Status = Ok` when they all pass or `Degraded` when any fails.

`DTO` carries `Status`, `ApplicationTime`, `DBHealthCheck` (`query.DBHealth`) and
`Dependencies`. The status constants are `Ok`, `Degraded`, and `Unhealthy`;
`Unhealthy` is not emitted — a database that cannot answer surfaces as the returned
error, which the shared `apperror` mapping turns into `503`.

## Degradable dependencies — `Probe`

A `Probe` is a `Name` plus a `Check(ctx) error`. The database is not one of them: it
is the reason this endpoint exists, so its failure is an error rather than a degraded
status. What belongs here is a dependency the instance can keep serving ordinary HTTP
without — Realtime Delivery is the first — and a failing probe therefore never turns
into an error. Returning `503` for one would drop the instance out of the load
balancer and stop the traffic that is still healthy.

This package never learns which subsystems exist. Probes arrive as a value group
(`readiness.probes`) that the owning DI module contributes to, so the dependency
points from the subsystem to this package and never back
(`internal/di/module/README.md`).

## DB probe — `healthcheck/query`

The `query` subpackage is a thin leaf boundary: `DBSystemCqrs` with a single
`CheckDBHealth(ctx) (DBHealth, error)`, where `DBHealth` reports `Ready`,
`RespondedAt`, and `Latency`. The concrete implementation lives in
`internal/infrastructure/rdb/system_cqrs/healthcheck/` and runs a lightweight
`SELECT 1` liveness probe (`database/dml/system_cqrs/health_check/`).

## Layout

| Concern | Path |
| --- | --- |
| usecase | `internal/usecase/healthcheck/` (this package) |
| DB probe boundary | `internal/usecase/healthcheck/query/` (`DBSystemCqrs`) |
| clock boundary | `internal/usecase/boundary/clock/` (`Clock`) |
| infrastructure | `internal/infrastructure/rdb/system_cqrs/healthcheck/` |
| sqlc DML | `database/dml/system_cqrs/health_check/` |
