# fix-collation

English | [日本語](README.ja.md)

Fixes PostgreSQL collation version mismatch by running `REINDEX DATABASE` and `ALTER DATABASE ... REFRESH COLLATION VERSION`.

## Command

```text
fix-collation [flags]
```

## Flags

|Flag|Default|Description|
|---|---|---|
|`--database`|`local`|Target database name|

## Usage

```bash
./server fix-collation --database local
```

## Notes

- **Intended for local and test databases only.** Exercise caution if running in production; schedule during maintenance windows.
- Requires `psql` to be available on `PATH` and the executing user to have `REINDEX` and `ALTER DATABASE` privileges.
- Database connection information is read from the application configuration (`DSN`).
- SQL execution stops immediately on error (`ON_ERROR_STOP=1`).
