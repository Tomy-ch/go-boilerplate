# db-slot

English | [日本語](README.ja.md)

Leases a per-worktree slot on a single shared infra stack so multiple `git worktree`s can run DB-backed work in parallel without host-port conflicts. Each slot owns its own databases (`wt<N>_local` / `wt<N>_test`) inside the shared DB container (fixed infra compose project `gobp-shared`, host 5432) plus its own app-layer compose project (`gobp-wt-N`) whose host ports are slot-relative. Every checkout attaches to the same shared infra either way, so leasing a slot is **opt-in** — it is what makes parallel work collision-free. See `docs/maintenance/db-worktree-pool.md` for the full model.

Runs on the **host** (not in a tool-runner container): it manages host-filesystem leases and drives host `docker compose`, and connects to the shared DB via pgx on `localhost:5432`.

## Commands

```text
db-slot acquire     # lease a free slot, create/setup wt<N> DBs, write .gobp-db-slot
db-slot release     # stop the slot's app containers, drop the lease (DBs left warm)
db-slot heartbeat   # refresh the held slot's lease heartbeat
db-slot status      # print slot occupancy
```

Prefer the make wrappers (`make slot-acquire` / `slot-free` / `slot-release` / `slot-status`) — `slot-acquire` also rebuilds the schema of the leased DBs, and `slot-release` tears the whole worktree down.

## Design

- **Lease** (`Registry`) — a lock directory per slot under `~/.cache/gobp-db-pool` (override `GOBP_DB_POOL_DIR`). `os.Mkdir` gives atomic fresh acquisition; stale reclaim atomically re-claims via `rename`, and a `flock` scan lock serialises the whole acquire loop so two worktrees can never double-lease the same slot. Symlinked pool dirs are refused (pre-attack guard); meta files are `0600`.
- **DB admin** (`DBAdmin` / `PgxAdmin`) — `CREATE DATABASE` + `pg_trgm` extension on each `wt<N>` DB, and a `pg_stat_activity` connection check used when deciding whether a stale slot is safe to reclaim. Timezone is not among them: the `database` container's `TZ` is the cluster default, which a database created later inherits.
- **In-use detection** — before reclaiming a stale slot, both its app project (`gobp-wt-N`) is checked for running containers and its databases for live connections. A connection pool empties while idle, so the connection check alone misses a worktree that is still serving; the container check alone misses host-run `go test`.
- **Compose** (`Compose` / `ExecCompose`) — brings up the shared DB in the fixed infra project (`--wait`) and, on release, tears down the slot's app project (`gobp-wt-N`) via `docker-compose.yaml` + `docker-compose.attach.yaml`.
- **Slot file** — `acquire` writes `.gobp-db-slot`, a gitignored `KEY=VALUE` file that `make` `-include`s to override the defaults in `.makefiles/docker/compose.mk`: `SLOT`, `DB_NAME_LOCAL` / `DB_NAME_TEST`, the slot-relative host ports `API_HOST_PORT` / `MOCK_AUTH_HOST_PORT` / `DLV_HOST_PORT` / `PPROF_HOST_PORT`, `COMPOSE_PROJECT_NAME` (the shared infra project) and `SERVE_PROJECT` (`gobp-wt-N`, the app-layer project).
- **Env guard** — refuses to run when `APP_ENV` is a deployed environment (`dev` / `stg` / `prd`); the pool creates and drops databases and must stay a dev/test-only tool (`config.IsLocalClassEnv`).

## Environment variables

|Variable|Default|Description|
|---|---|---|
|`GOBP_DB_POOL_DIR`|`~/.cache/gobp-db-pool`|Lease registry location|
|`GOBP_DB_SHARED_PROJECT`|`gobp-shared`|Fixed compose project of the shared infra|
|`GOBP_DB_POOL_MAX`|`8`|Number of slots (max parallel worktrees)|
|`GOBP_DB_POOL_TTL`|`1800`|Heartbeat staleness grace (seconds)|
|`GOBP_API_POOL_BASE` / `GOBP_MOCK_AUTH_POOL_BASE`|`8080` / `4000`|Base host ports of the API / mock auth server (slot N = base + N)|
|`GOBP_DLV_POOL_BASE` / `GOBP_PPROF_POOL_BASE`|`2345` / `6060`|Base host ports of the dlv debug / pprof endpoints (slot N = base + N)|

## Notes

- **Dev/test only.** Refuses to run in `dev` / `stg` / `prd`.
- A checkout without a slot still runs against the same shared infra — it just keeps the default `local` / `test` databases and the default host ports. Take a slot only when you need collision-free parallel work.
