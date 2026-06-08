# CLI

English | [日本語](README.ja.md)

`internal/cli` provides a **Cobra-based command-line interface** for the application.

It defines commands required for application operations such as server startup, database migration, seeding, and job execution.

## Command List

|Command|Package|Description|
|---|---|---|
|`serve`|`server/`|Start HTTP server and metrics server|
|`migrate-up`|`migrate/`|Upgrade DDL (`--version` / `--database` options)|
|`migrate-down`|`migrate/`|Downgrade DDL (`--version` / `--database` options)|
|`db-seed`|`seed/`|Insert initial data into database|
|`job`|`job/`|Execute a registered job (`job <job-name> [args...]`)|
|`fix-collation`|`fixcollation/`|Fix PostgreSQL collation version mismatch|
|`dump-schema`|`dumpschema/`|Dump and format DB schema|
|`merge-dml`|`mergedml/`|Merge DML directory SQL files by type|

## Structure

```text
internal/cli/
├── cli.go              # RegisterCommands (subcommand registration)
├── server/             # serve command + metrics server
├── migrate/            # migrate-up / migrate-down
├── seed/               # db-seed
├── job/                # job <name>
├── fixcollation/       # fix-collation
├── dumpschema/         # dump-schema
└── mergedml/           # merge-dml
```

`RegisterCommands` in `cli.go` registers all subcommands to the Cobra root command.

## Design Policy

- Each command is isolated as one package = one command
- The CLI layer does not contain business logic (calls Usecase via DI)
- Adding a new command only requires adding it to `RegisterCommands` in `cli.go`

## Testing Policy

The CLI layer is a **driving adapter (humble object)**. Its assurance strategy is split deliberately, and **this split is documented here rather than implied by coverage numbers** — coverage is a measurement, not a place to encode "handle with care".

- **No silently-wrong logic in the shell.** Any decision (error handling, branching, formatting, deletion conditions, timeout dispatch) MUST be extracted into a pure function/method and **unit-tested for branch coverage**. The thin `RunE` / wiring around it is not unit-tested.
- **OS / filesystem / external-process / DB dependencies are injected via interfaces.** Production code wires the real implementations; unit tests pass fakes, so **tests never touch the real filesystem, run external binaries (`pg_dump` / `psql`), or open a DB**.
- **What covers the rest (not unit tests):**
  - DB access behaviour → repository tests against a real Postgres (`internal/infrastructure/rdb/testkit`).
  - "Does the entrypoint actually run?" → CI boot checks: `app-di-startup-check` (serve → `/ready`), `job-boot-check` (job dispatch), `migration-check` (up/down round-trip), `gen-*-artifacts-check` (codegen dogfooding) — all against a real Postgres service.
- **Why `cli` is excluded from unit coverage** (`makefile` `TGT_PKGS`): the shell is intentionally thin and its runtime behaviour is verified by the CI gates above. Do **not** read the low / absent unit coverage as "untested" — read this section.

### When adding a command

- Keep `RunE` thin: parse flags → build dependencies → delegate to a testable function.
- Bind flags to **local variables**, not package globals, so commands are parallel-test-safe.
- Inject OS / FS / exec / DB through interfaces and provide the real implementations as defaults.
- Unit-test the extracted logic for branch coverage; leave the thin shell to the CI gates.

## Notes

- Backup is recommended before running migration or seed operations
- Server startup settings are managed via environment variables (see `internal/config`)
- This directory is infrastructure-level; AI agents should not modify it unless explicitly instructed
