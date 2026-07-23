# db-pool

English | [日本語](README.ja.md)

Leases a per-worktree slot on a single shared Postgres so multiple `git worktree`s can run DB-backed work in parallel without host-port conflicts. Each slot owns its own databases (`wt<N>_local` / `wt<N>_test`) inside one shared DB container (fixed compose project `gobp-shared`, host 5432). See `docs/maintenance/db-worktree-pool.md` for the full model.

Runs on the **host** (not in a tool-runner container): it manages host-filesystem leases and drives host `docker compose`, and connects to the shared DB via pgx on `localhost:5432`.

## Commands

```text
db-pool acquire     # lease a free slot, create/setup wt<N> DBs, write .gobp-db-slot
db-pool release     # stop the slot's serve containers, drop the lease (DBs left warm)
db-pool heartbeat   # refresh the held slot's lease heartbeat
db-pool status      # print slot occupancy
```

Prefer the make wrappers (`make db-acquire` / `db-release` / `db-pool-status`) — `db-acquire` also rebuilds the schema of the leased DBs.

## Design

- **Lease** (`Registry`) — a lock directory per slot under `~/.cache/gobp-db-pool` (override `GOBP_DB_POOL_DIR`). `os.Mkdir` gives atomic fresh acquisition; stale reclaim atomically re-claims via `rename`, and a `flock` scan lock serialises the whole acquire loop so two worktrees can never double-lease the same slot. Symlinked pool dirs are refused (pre-attack guard); meta files are `0600`.
- **DB admin** (`DBAdmin` / `PgxAdmin`) — `CREATE DATABASE` + `pg_trgm` extension + `Asia/Tokyo` timezone on each `wt<N>` DB, and a `pg_stat_activity` connection check that prevents reclaiming a slot whose DB is still in active use.
- **Compose** (`Compose` / `ExecCompose`) — brings up the shared DB (`--wait`) and, on release, tears down the worktree's serve project (`gobp-wt-N`).
- **Env guard** — refuses to run when `APP_ENV` is a deployed environment (`dev` / `stg` / `prd`); the pool creates and drops databases and must stay a dev/test-only tool (`config.IsLocalClassEnv`).

## Environment variables

|Variable|Default|Description|
|---|---|---|
|`GOBP_DB_POOL_DIR`|`~/.cache/gobp-db-pool`|Lease registry location|
|`GOBP_DB_SHARED_PROJECT`|`gobp-shared`|Fixed compose project of the shared DB|
|`GOBP_DB_POOL_MAX`|`8`|Number of slots (max parallel worktrees)|
|`GOBP_DB_POOL_TTL`|`1800`|Heartbeat staleness grace (seconds)|
|`GOBP_API_POOL_BASE` / `GOBP_MOCK_AUTH_POOL_BASE`|`8080` / `4000`|Base host ports (slot N = base + N)|

## Notes

- **Dev/test only.** Refuses to run in `dev` / `stg` / `prd`.
- Adopting the pool moves the DB home from each checkout's default project to `gobp-shared`; the two are mutually exclusive on host 5432.
