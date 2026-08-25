# CommandService Implementation Guide

English | [日本語](README.ja.md)

## Why this directory exists even when it is empty

The data-access directories mirror the DML categories under `database/dml/` one for one, because a
category that exists in SQL and not in Go is a category nobody decided on
([`../README.md`](../README.md) § Directory Structure). The pairing is what decides the directory,
not whether an implementation happens to sit in it, so it holds only this guide until the first
CommandService is written.

## What belongs here

The criterion is not restated here. It lives in [`../README.md`](../README.md) § command_service,
which owns two rules this package cannot be understood without:

- **What may live here** — only a write that cannot be expressed as loading an aggregate and saving
  it. Anything that can be read-modify-saved belongs on the Repository.
- **Conditions are derived, never authored** — a guard enforced in SQL here must restate a domain
  invariant that already exists, and returns that same domain sentinel.

The decision records behind them are
[ADR-0032 (lightweight-cqrs)](../../../../docs/adr/0032-lightweight-cqrs.md) and
[ADR-0034 (commandservice-atomicity-criterion)](../../../../docs/adr/0034-commandservice-atomicity-criterion.md);
write ordering follows
[ADR-0036 (ordered-pessimistic-row-locks)](../../../../docs/adr/0036-ordered-pessimistic-row-locks.md).

## Placement

```txt
command_service/
 └ <aggregate>/
     └ <aggregate>_command_service.go
```

The interface lives in the **Usecase layer**, beside the QueryService interface rather than in the
Domain layer, for the reason [`../query_service/README.md`](../query_service/README.md) gives: the
shape is decided per usecase, not per aggregate.

```txt
internal/usecase/<aggregate>/command/<aggregate>_command_service.go
```

Infrastructure only implements that interface.

## Constructor

Dependencies arrive as arguments; nothing is constructed inside.

```go
func New(
    db driver.DatabaseDriver,
    tf observability.TracerFactory,
) command.CommandService {
    return &commandService{
        db:     db,
        tracer: tf.Infra(),
    }
}
```

## Transaction

A CommandService **never opens a transaction.** It executes on the one supplied through `ctx`,
which `driver.New(ctx, db)` picks up, because the Usecase owns the boundary and nests the write
under `idempotency.Run`. Opening one here would put a second boundary inside the first, and the
outer rollback would no longer cover this write.

```go
db := gen.New(driver.New(ctx, c.db))
```

## Outbox

A CommandService does **not** emit outbox events. That is a Usecase responsibility, served by the
`system_cqrs` category — keeping it out of here is what lets the same write be composed into a
workflow that decides for itself whether an event is warranted.

## Errors and observability

Every sqlc error is normalized with `pgerror.NormalizeError`, and the method opens an
infrastructure-layer span through the injected `TracerFactory`, both exactly as Repository and
QueryService do. See [`../pgerror/README.md`](../pgerror/README.md) and [`../README.md`](../README.md)
§ Observability.

## DI

Register the constructor in the `command_service` sub-module of `persistenceModule`
(`internal/di/module/persistence.go`). The provided type is the Usecase-layer interface, never the
concrete struct.

```go
fx.Module("command_service",
    fx.Provide(
        <aggregate>cmd.New,
    ),
),
```

## Tests

An integration test asserts the atomic write effect on every touched table via post-write `SELECT`,
the fail-closed guard using a stale lock value so the domain check passes but the DB predicate
rejects, and constraint-violation normalization. The full test strategy, including what the default
`testkit` helper cannot reproduce, is in [`../README.md`](../README.md) § Test Strategy.
