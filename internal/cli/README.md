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

## Notes

- Backup is recommended before running migration or seed operations
- Server startup settings are managed via environment variables (see `internal/config`)
- This directory is infrastructure-level; AI agents should not modify it unless explicitly instructed
