# migrate-up / migrate-down

English | [日本語](README.ja.md)

Manages database schema migrations. `migrate-up` applies pending migrations; `migrate-down` rolls back applied migrations (all by default, or a given number of steps).

## Commands

```text
migrate-up
migrate-down
```

## Flags

|Flag|Default|Description|
|---|---|---|
|`--steps`|`0`|Number of migrations to apply relative to the current position. `0` applies all; a positive integer applies that many steps. Negative values are rejected.|
|`--database`|`""`|Target database name (e.g. `local`, `test`). When empty, the configured default is used.|

`--steps` is a **relative step count**, not an absolute target version. For example `migrate-up --steps 2` advances two migrations from the current position; `migrate-down --steps 2` rolls back two.

## Usage

```bash
# Apply all pending migrations
./server migrate-up

# Apply the next 2 migrations only
./server migrate-up --steps 2

# Roll back all migrations
./server migrate-down

# Roll back the last 2 migrations
./server migrate-down --steps 2

# Target a specific database
./server migrate-up --database test
```

## Notes

- Migration files are located in `database/migrations`.
- **Use `migrate-down` with caution in production** -- it can cause data loss. Always back up the database first.
- Never modify existing migration files; always create a new one.
- A full `migrate-down` (no `--steps`) automatically reconciles a `dirty` database at its current version before rolling back, so a previously failed migration does not block the rollback.
- `ErrNoChange` (already at the target position) is treated as success.
