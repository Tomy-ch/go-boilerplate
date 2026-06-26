# CLI core

English | [日本語](README.ja.md)

`internal/cli` holds the **pure, testable core logic** for the application's CLI commands.

It does NOT depend on Cobra or infrastructure wiring. The Cobra command definitions and the
composition root that wires real dependencies (config / DB / DI / OS signals / golang-migrate)
live in `cmd/` (package `main`). This split keeps the core unit-testable and the wiring thin.

## Command List

|Command|Core package|Cobra shell|Description|
|---|---|---|---|
|`serve`|`server/`|`cmd/serve.go`|Start HTTP server and metrics server|
|`migrate-up`|`migrate/`|`cmd/migrate.go`|Upgrade DDL (`--steps` / `--database` options)|
|`migrate-down`|`migrate/`|`cmd/migrate.go`|Downgrade DDL (`--steps` / `--database` options)|
|`db-seed`|`seed/`|`cmd/seed.go`|Insert initial data into database|
|`job`|`job/`|`cmd/job.go`|Execute a registered job (`job <job-name> [args...]`)|
|`fix-collation`|`fixcollation/`|`cmd/fix_collation.go`|Fix PostgreSQL collation version mismatch|
|`dump-schema`|`dumpschema/`|`cmd/dump_schema.go`|Dump and format DB schema|
|`merge-dml`|`mergedml/`|`cmd/merge_dml.go`|Merge DML directory SQL files by type|
|`worker`|`worker/`|`cmd/worker.go`|Run a registered worker (`worker <worker-name> [args...]`)|
|`outbox-relay`|`outbox/`|`cmd/outbox_relay.go`|Run the outbox relay; `replay` subcommand returns dead rows to pending|

## Structure

```text
cmd/                     # package main: Cobra defs + composition root (excluded from coverage)
├── commands.go          # registerCommands (subcommand registration)
├── serve.go             # newServeCommand + serveRun wiring
├── migrate.go           # newMigrate{Up,Down}Command + buildMigrateInstance
└── ...                  # one file per command

internal/cli/            # pure testable core (covered by unit tests, 90%+)
├── server/              # RunServer / ResolveMetricsStop / NewMetricsServer
├── migrate/             # MigrateUpRun / MigrateDownRun / Migrator
├── seed/                # RunDBSeed
├── job/                 # RunJobWith
├── fixcollation/        # RunFix
├── dumpschema/          # RunDump / NewGenerator
├── mergedml/            # RunMerge / NewGenerator
├── worker/              # RunWorkerWith / NewHealthServer
└── outbox/              # RunRelay / RunReplayWith
```

`registerCommands` in `cmd/commands.go` registers all subcommands to the Cobra root command.

## Design Policy

- Each command is one core package under `internal/cli/` + one thin shell file under `cmd/`.
- The core must NOT import Cobra, `internal/di`, `internal/config`, OS signals, or infrastructure
  (except `infrastructure/rdb/driver` types). It operates on injected interfaces / function seams.
- The CLI layer does not contain feature business logic (that belongs in usecase / domain).
- Adding a new command: add `cmd/<command>.go` (Cobra def + real-dependency wiring), add the core
  logic under `internal/cli/<command>/`, and register it in `registerCommands`.

## Testing Policy

The core is a **humble object**: all decision logic lives here and is unit-tested; the wiring is
pushed out to `cmd/`. The package boundary equals the test boundary.

- **No silently-wrong logic in the shell.** Any decision (error handling, branching, formatting,
  deletion conditions, timeout dispatch) lives in `internal/cli/*` and is **unit-tested for branch
  coverage**. `internal/cli/*` is included in the coverage gate and is expected to meet 90%+.
- **OS / filesystem / external-process / DB / logger dependencies are injected** (interfaces or
  function seams). Production wires the real implementations in `cmd/`; unit tests pass fakes, so
  **tests never touch the real filesystem, run external binaries (`pg_dump` / `psql`), or open a DB**.
- **The thin `cmd/` shells are excluded** from the coverage gate (`gen|cmd|mock|apperror|scripts`).
  Their runtime correctness is covered by CI boot checks: `app-di-startup-check` (serve → `/ready`),
  `job-boot-check` (job dispatch), `migration-check` (up/down round-trip), `gen-*-artifacts-check`
  (codegen dogfooding) — all against a real Postgres service. DB access behaviour is covered by
  repository tests against a real Postgres (`internal/infrastructure/rdb/testkit`).

### When adding a command

- Keep the `cmd/` shell thin: parse flags → build dependencies → delegate to a core function.
- Bind flags to **local variables**, not package globals, so commands are parallel-test-safe.
- Inject OS / FS / exec / DB / logger through interfaces or params; provide the real implementations
  in `cmd/`.
- Unit-test the core for branch coverage; leave the thin `cmd/` shell to the CI gates.

## Notes

- Backup is recommended before running migration or seed operations.
- Server startup settings are managed via environment variables (see `internal/config`).
