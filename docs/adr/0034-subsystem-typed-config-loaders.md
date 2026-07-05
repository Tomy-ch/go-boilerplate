---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [config]
---

# ADR-0034: Subsystem-scoped envPrefix typed config loaders

## Status

accepted

## Context

The application reads configuration from environment variables covering many concerns:
operating system settings, application identity, HTTP server timeouts, database
connection, observability, security, authentication, workers, and the outbox relay.

A flat, unscoped namespace for all variables (e.g., `HOST`, `PORT`, `TIMEOUT`) collides
as the number of variables grows and makes it impossible to infer which component a
variable belongs to from its name alone. Parsing all variables into a single struct
with no internal grouping makes the code difficult to navigate and ownership of each
value unclear.

## Decision

Each subsystem has its own **typed struct** in `internal/config/envspec.go`, and the
root `Loader` struct embeds all subsystems as named fields with a matching `envPrefix`
tag. Environment variable names follow `{SUBSYSTEM}_{NAME}` in `UPPER_SNAKE_CASE`.

The root `Loader` in `internal/config/envspec.go` embeds each subsystem as a named
field with a matching `envPrefix` tag. The illustrative shape is:

```go
// Abbreviated — see internal/config/envspec.go for the definitive list of subsystems.
type Loader struct {
    OS     OperatingSystem `envPrefix:"OS_"`
    Server Server          `envPrefix:"SERVER_"`
    // … one field per subsystem
}
```

For the full subsystem list and field details, see `internal/config/envspec.go` and the
loading flow in `internal/config/README.md`.

`env.ParseAs[Loader]()` (via `github.com/caarlos0/env/v11`) maps the prefixed
environment variables into the corresponding typed struct fields automatically.

After loading, `config.New()` converts `Loader` into the internal `Config` type and
exposes narrowly scoped SubConfig providers (`NewServerConfig`, `NewDatabaseConfig`,
etc.) so each component receives only the fields it needs.

## Consequences

### Positive Consequences

- Variable ownership is immediately clear from the prefix: `DB_HOST` belongs to the
  `Database` subsystem, `SERVER_PORT` to `Server`.
- Adding a new subsystem is a contained change: add a struct in `envspec.go`, a field
  in `Loader` with an `envPrefix`, and the corresponding SubConfig provider.
- Each subsystem struct is independently readable and testable.

### Negative Consequences

- Adding a new subsystem requires coordinated changes across `envspec.go`, `model.go`,
  and `config.go` (the Loader-to-Config conversion step).
- The `envPrefix` indirection means the raw env var name and the Go field name do not
  match; operators must refer to the canonical table in `env/README.md`.

## Alternatives Considered

### Single flat struct for all variables

All env vars parsed into one struct with no subsystem grouping. Rejected: collisions
on common names (e.g., `HOST`, `PORT`) require manual disambiguation, and the struct
becomes unnavigable at scale.

### One file per subsystem

Separate `envspec_*.go` files, one per subsystem, each with its own `ParseAs` call.
Rejected: multiple parse passes lose the benefit of a single, validated `Loader` and
require manual aggregation before validation.

## Notes

- Source: `internal/config/envspec.go` (Loader struct definition),
  `env/README.md` (variable naming conventions and subsystem tables).
- The SubConfig provider pattern and immutability rules are recorded in
  [ADR-0036](0036-immutable-fail-fast-config.md).
- The governance rule for when a field uses `envDefault` vs `required` is recorded in
  [ADR-0035](0035-config-default-vs-required-governance.md).
