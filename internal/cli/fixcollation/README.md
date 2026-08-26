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
|`--database`|`local`|Target database name. One of `local`, `test`, `template1`, `wt<N>_local`, `wt<N>_test`|

## Usage

```bash
./server fix-collation --database local
./server fix-collation --database template1   # unblocks CREATE DATABASE ... TEMPLATE template1
./server fix-collation --database wt3_test    # a database leased by a worktree slot
```

## Notes

- **Intended for development and test databases only.** Any name outside the list above is rejected with an error before any SQL runs. The allowlist doubles as the injection guard, since the database name is interpolated into the SQL text.
- `template1` is accepted because the mismatch propagates to every database cloned from it: `CREATE DATABASE ... TEMPLATE template1` fails outright while the template carries a stale collation version.
- `wt<N>_local` / `wt<N>_test` are the databases a worktree leases from the slot pool (see `docs/maintenance/db-worktree-pool.md`).
- The command **connects to the database it is about to fix**, overriding the database in the configured DSN. `REINDEX DATABASE` can only target the currently open database, so reusing the configured connection would fail for every other name.
- Requires `psql` to be available on `PATH` and the executing user to have `REINDEX` and `ALTER DATABASE` privileges.
- Connection information other than the database name (host, port, user, `sslmode`) is read from the application configuration (`DSN`).
- SQL execution stops immediately on error (`ON_ERROR_STOP=1`).
