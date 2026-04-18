# db-seed

English | [日本語](README.ja.md)

Inserts seed data into the database for development and testing purposes.

## Command

```text
db-seed
```

## Flags

|Flag|Default|Description|
|---|---|---|
|*(none)*|||

## Usage

```bash
./server db-seed
```

## Notes

- **Not intended for production use.** Use only in development and test environments.
- Seed files are loaded from the `database/seed` directory.
- If a target table does not exist, the corresponding seed file is skipped and a warning is logged.
- Always verify seed data contents before running against shared environments.
