# dump-schema

English | [日本語](README.ja.md)

Dumps the database schema using `pg_dump` and sanitizes the output for sqlc consumption by removing tool-specific meta-commands.

## Command

```
dump-schema [flags]
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--work-dir` | `/app` | Working directory (project root) |

## Usage

```bash
./server dump-schema

./server dump-schema --work-dir /app
```

## Notes

- Output is written to `database/gen/schema.gen.sql`.
- Requires `pg_dump` on `PATH` and a valid database connection configured via the application DSN.
- Lines starting with `\` (psql meta-commands) are automatically stripped from the output.
- Default `pg_dump` flags: `--schema-only --no-owner --no-privileges --format=plain`.
