# migrations

English | [日本語](README.ja.md)

`database/migrations` stores **database migration files managed by golang-migrate**.

## File Generation

Generate new migration files with:

```bash
make new-migrate-<name>
```

Example:

```bash
make new-migrate-create_orders
```

This auto-generates a numbered up / down pair:

```text
000011_create_orders.up.sql
000011_create_orders.down.sql
```

## File Naming Convention

```text
{6-digit sequence}_{description}.up.sql    # Upgrade (apply)
{6-digit sequence}_{description}.down.sql  # Downgrade (rollback)
```

- Sequence starts at `000001`, zero-padded to 6 digits
- Description uses snake_case to briefly describe the change
- up and down must always be created as a pair

## Running Migrations

|Command|Description|
|---|---|
|`make migrate-up`|Apply all pending migrations|
|`make migrate-down`|Rollback the last migration|

Also available via CLI:

```bash
./server migrate-up
./server migrate-down
```

## Rules

- **Never modify existing migration files** — changing applied files causes hash mismatches
- **Always create new files** — schema changes must be done via new migrations
- **Create up and down as pairs** — rollback is impossible without down
- **down must be the exact inverse of up** — table creation → DROP, column addition → DROP COLUMN
- **Be idempotent where possible** — use `IF NOT EXISTS` / `IF EXISTS`
- **No gaps in sequence numbers** — CI detects gaps (`migration-check.yaml`)

## CI Check

The `migration-check.yaml` workflow automatically verifies:

- No duplicate sequence numbers
- No gaps in sequence numbers
- up / down pairs are complete

## Notes

- Only place DDL (table definitions, schema changes) in this directory
- DML (data manipulation) belongs in `database/dml/`
- Seed data belongs in `database/seed/`
- Initialization SQL (DB creation, extension setup) belongs in `docker/database/sql/`
