---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [cli, build]
---

# ADR-0083: All roles are one multi-command binary

## Status

accepted

## Context

A Go API scaffold needs to ship multiple operational roles: HTTP server, database migrations
(up and down), seeding, schema dump, DML merge, job dispatch, background worker, and outbox
relay. Each role could be distributed as a separate binary, but that multiplies build
artifacts, complicates image distribution, and duplicates shared initialization code.

## Decision

Compile all roles into a single binary produced from `./cmd/`. The binary exposes each role
as a Cobra subcommand registered in `registerCommands`:

- `serve` — HTTP server (long-running)
- `migrate-up` / `migrate-down` — database migrations
- `db-seed` — data seeding
- `fix-collation` — collation repair utility
- `dump-schema` — schema dump
- `merge-dml` — DML merge
- `job` — application job dispatch
- `worker` — background worker
- `outbox-relay` — transactional outbox relay

The entry point (`cmd/main.go`) creates a root Cobra command, delegates subcommand
registration to `registerCommands`, and exits with a non-zero code on error.

## Consequences

### Positive Consequences

- One build artifact to version, push, and pull. The same image can run any role via command
  override (see [ADR-0084](0084-single-runtime-image.md)).
- Shared initialization paths (config, DI, logging) are compiled once and reused across all
  subcommands.
- Version metadata (`Version`, `Revision`, `BuildDate`) is embedded once via `-ldflags` and
  reported by `--version` for all roles.

### Negative Consequences

- The binary includes code for all roles regardless of which one is active at runtime,
  increasing binary size slightly.
- A defect in one role's init path could theoretically crash another role's startup if
  shared initialization is not isolated correctly.

## Alternatives Considered

### Separate binary per role

Produces the smallest per-role artifact but multiplies CI build steps, image layers, and
operational procedures. Sharing config / DI code requires a shared library, adding a
dependency boundary that Go module layout does not naturally enforce.

### Single binary with sub-process model

The root binary spawns child processes for each role. Adds complexity (process supervision,
signal forwarding) with no benefit for a scaffold that already isolates roles at the Cobra
subcommand level.

## Notes

- `cmd/main.go` and `cmd/commands.go` are the authoritative entry points.
- Command logic lives in `internal/cli/<command>/` per [ADR-0082](0082-cli-humble-object-split.md).
- Source: `cmd/main.go`, `cmd/commands.go`.
