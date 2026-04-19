# migrate-up / migrate-down

English | [日本語](README.ja.md)

Manages database schema migrations. `migrate-up` applies pending migrations; `migrate-down` rolls back the last migration.

## Commands

```text
migrate-up
migrate-down
```

## Flags

|Flag|Default|Description|
|---|---|---|
|*(none)*|||

## Usage

```bash
# Apply all pending migrations
./server migrate-up

# Roll back the last migration
./server migrate-down
```

## Notes

- Migration files are located in `database/migrations`.
- **Use `migrate-down` with caution in production** -- it can cause data loss. Always back up the database first.
- Never modify existing migration files; always create a new one.
