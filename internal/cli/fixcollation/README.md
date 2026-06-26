# fix-collation

English | [日本語](README.ja.md)

Fixes PostgreSQL collation version mismatch by running `REINDEX DATABASE` and `ALTER DATABASE ... REFRESH COLLATION VERSION`.

## Role

This command exists to recover from collation drift: when the operating system's collation library is upgraded under a running database, the sort order of existing text indexes can silently no longer match the order the database assumes, which risks wrong query results until the indexes are rebuilt. It rebuilds the affected indexes and re-stamps the recorded collation version so the database stops warning and the indexes are trustworthy again. It lives as its own pure, unit-tested core so the rebuild-and-refresh decision logic is verifiable in isolation, separate from the thin command wiring.

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
