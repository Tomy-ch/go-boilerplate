# Shared Infra and the DB Slot Pool (parallel worktree development)

日本語: [db-worktree-pool.ja.md](../ja/maintenance/db-worktree-pool.ja.md)

The mechanism that lets several git worktrees (and the main checkout) use a **single shared infra**
in parallel without colliding. Compose services are split into two layers:

- **infra layer** — the services that can only run on fixed ports (`database` 5432 / `observability`
  3000, 4317, 4318, 3200 / `garage` 3900, 3902). They live in the fixed compose project
  `gobp-shared`, with **one instance for all checkouts**.
- **app layer** — `api_server` / `mock_auth_server`, which every checkout needs for itself. They are
  split into a per-checkout compose project and started in parallel on shifted host ports.

Worktree separation of the DB is "a different database inside the same instance"
(`wt<N>_local` / `wt<N>_test`), not "a different container on a different port". That removes the
need to allocate a host port per DB, so neither "another worktree holds 5432, so `make test` cannot
run" nor "the main checkout's serve collides with a worktree's DB" can happen. For o11y, sharing is
an advantage: the traces / metrics / logs of every checkout land in a single Grafana.

## How it works

- **infra layer** = the fixed compose project `gobp-shared` (`GOBP_DB_SHARED_PROJECT`). A worktree
  gets a different default compose project name per directory, so the shared side is pinned to an
  explicit fixed name. `make infra-up` starts it; `make serve` / `make job` / `make worker` also call
  it idempotently first. Idempotent here means non-destructive too: an already-running container is
  left alone rather than re-created (see Caveats).
- **app layer** = `APP_PROJECT`: `gobp-app-<directory name>` when no slot is held, `gobp-wt-N` when
  one is. `docker-compose.attach.yaml` is overlaid so the shared infra is reached through
  `host.docker.internal` on its host-published ports (`DB_HOST` / `OBS_OTLP_ENDPOINT` /
  `OBJECT_STORAGE_ENDPOINT` are overridden as runtime env; `loader.go` gives runtime env priority
  over `env/.env`).
- **slot N** = the database-name pair `wt<N>_local` / `wt<N>_test` inside the shared DB (MAX 12 by
  default = wt1–wt12). A checkout that takes no slot keeps the default `local` / `test`, so acquiring
  a slot stays an **opt-in for parallel work**.
- **implementation** = the host-run Go CLI `cmd/db-slot` (core in `internal/cli/dbslot`), which owns
  lease decisions, database creation, and compose startup in a testable form. The make targets call
  `go run ./cmd/ db-slot <sub>`.
- **lease** = the lock directory `${GOBP_DB_POOL_DIR:-~/.cache/gobp-db-pool}/slot-N.lock` on the
  host. A new acquisition relies on the atomicity of `os.Mkdir`, stale reclaim seizes ownership
  atomically via `rename`, and the whole acquire scan is serialized with `flock`, so two worktrees
  never double-lease the same slot. `meta` (owner / heartbeat / branch) is `0600`, and a pool dir
  pointing at a symlink is rejected as a pre-read attack countermeasure.
- **runtime environment guard** = db-slot refuses to run when `APP_ENV` is a deploy environment
  (dev/stg/prd), since it creates and destroys databases and is therefore dev/test only. The test
  configuration in `internal/config` likewise ignores the DB_NAME override in deploy environments.
- **slot record** = acquire writes `.gobp-db-slot` (gitignored) at the worktree root. `make`
  `-include`s it, overriding the defaults from `.makefiles/docker/compose.mk` and propagating to
  every target:
  - `DB_NAME_LOCAL` / `DB_NAME_TEST` = `wt<N>_local` / `wt<N>_test` (default `local` / `test`; a
    host-run `go test` connects through the shared DB on localhost:5432 to its own worktree database
    under this name — read by the test configuration in `internal/config`)
  - `API_HOST_PORT` = `8080+N` / `MOCK_AUTH_HOST_PORT` = `4000+N` / `DLV_HOST_PORT` = `2345+N` /
    `PPROF_HOST_PORT` = `6060+N` (every app-layer host port is relative to the slot number)
  - `SERVE_PROJECT` = `gobp-wt-N` (the app layer's compose project = `APP_PROJECT`)
  - `COMPOSE_PROJECT_NAME` = `gobp-shared` (moves the default project to the infra layer so DB
    tooling — migrate / seed / psql / gen — runs on the shared infra's network; `compose.mk` sets the
    same default even when no slot is held)
- **persisted data that follows the shifted ports**: a host port is not only something to connect to —
  it can also be *stored* in the database. The JWT issuer is one such value: the mock auth server
  publishes on `4000+N`, so the `iss` of the tokens it issues shifts with the slot, and the
  `user_identities` row that the resolver matches on `(issuer, subject)` has to shift with it — with a
  pinned literal, every authenticated endpoint answers 401 in a worktree that holds a slot. The seed
  file therefore stores `${AUTH_ISSUER}` instead of the URL and `make db-seed` passes this slot's value
  in (see `database/seed/README.md`), so `db-reinit` / `db-seed` / `slot-acquire` all leave an identity
  that matches the environment. Data of this kind that you add later has to follow the slot the same
  way, rather than pinning the default port. Like the database name, the value reaches a host-run
  `go test` only through `make` (`make test` / `test-cached` export it) — run DB-backed tests through
  those targets, since a bare `go test` gets neither `DB_NAME_TEST` nor the slot's issuer.
- **extension bootstrap**: after `CREATE DATABASE` (guarded by an existence check) for
  `wt<N>_local` / `wt<N>_test`, acquire sets the `pg_trgm` extension on each database (the same thing
  the init script applies to `local` / `test`; a dynamically created worktree database needs it set
  explicitly). Timezone is *not* set per database: the `database` service's `TZ` is written into
  `postgresql.conf` at `initdb` time, so it is the cluster default and a database created later
  inherits it. The consequence is that a shared volume initialised before `TZ` was set keeps its old
  cluster default, and a slot leased in it shows that timezone in `psql` — the application is
  unaffected, because it sets the timezone per connection in the DSN. To pick the new default up,
  recreate the volume (`docker compose -p gobp-shared down -v` → `make db-init`) once every slot has
  been freed; the volume is shared by every worktree. See `env/README.md` (Changing the Timezone).
- **schema safety**: after acquiring, acquire rebuilds `wt<N>_local` / `wt<N>_test` to the current
  branch's schema via drop → migrate → seed. Inheriting a slot another branch used is therefore safe.
- **schema-generation isolation**: `dump-schema` (behind `make gen-query`) dumps neither the shared
  `local` nor your own worktree database. It drops a dedicated `gen_schema` database (`SCHEMA_GEN_DB`),
  migrates it up from *this* branch's migrations, and dumps that. Migrations from a concurrent
  worktree therefore cannot leak into the generated artifacts (`schema.gen.sql`, `models.gen.go`, …).
  This is a local-only guard: CI migrates a fresh postgres service and calls `dump-schema-ci`
  directly, so it never takes this path.
- **release**: `slot-free` stops the containers of this slot's app project (`gobp-wt-N`) before
  deleting the lease and `.gobp-db-slot`. The databases are kept warm for the next tenant.
- **safe stale reclaim**: a lease whose heartbeat has exceeded the TTL (1800 seconds by default,
  `GOBP_DB_POOL_TTL`) can be re-acquired by another worktree, so a crashed worktree does not hold a
  slot forever. Because the heartbeat is only sent on `make serve`, a slot whose app is left running
  also goes stale once the TTL passes. Before rebuilding the databases, therefore, two checks confirm
  the slot is genuinely unused; if it is in use, the slot is skipped rather than destroyed:
  1. the app project (`gobp-wt-N`) has no running container
  2. `pg_stat_activity` shows no connection to that slot's databases

  Connection pools drain when idle, so check 2 alone would miss a worktree that is serving.
  Conversely check 1 alone cannot catch a host-run `go test`, so both are used.

## Usage

A checkout that takes no slot (the main checkout, typically) reaches the shared infra with a plain
`make serve`.

```sh
make serve           # start the shared infra and the app as gobp-app-<dir> → curl localhost:8080
make serve-stop      # stop only this checkout's app (the shared infra stays up)
make infra-up        # start just the shared infra
make infra-down      # stop the shared infra (affects every checkout)
```

Acquire a slot when working in a worktree in parallel.

```sh
make slot-acquire    # lease a free slot and create/rebuild this worktree's databases
make test            # connects from the host through localhost:5432 to wt<N>_test
make serve           # start the app as gobp-wt-N → curl localhost:$API_HOST_PORT (DB is the shared wt<N>_local)
make slot-status     # show slot occupancy (database names / API ports)
make slot-free       # release only the slot (databases stay warm, the worktree stays)
```

Use `slot-release` when the work is done and the worktree itself is being retired. It stops the app
and removes locally built images, releases the slot, and removes the worktree, in that order.

```sh
make slot-release    # stop app + remove images → release the slot → remove the worktree
```

The order cannot be rearranged. Once `slot-free` deletes `.gobp-db-slot`, `SERVE_PROJECT` is lost and
`APP_PROJECT` falls back to `gobp-app-<dir>`, so releasing first would point the app shutdown at a
different project. `git worktree remove` deletes the cwd along with everything else, so it must come
last. Git refuses the removal when uncommitted or untracked files remain, which is why `--force` is
not passed. Run by mistake in the main checkout, it exits with an error without doing anything.

## Environment variables

| Variable | Default | Meaning |
| --- | --- | --- |
| `GOBP_DB_POOL_DIR` | `~/.cache/gobp-db-pool` | Where the lease registry lives (a symlink is rejected) |
| `GOBP_DB_SHARED_PROJECT` | `gobp-shared` | Fixed compose project name of the shared infra |
| `GOBP_API_POOL_BASE` | `8080` | Base for `API_HOST_PORT` (slot N = base + N) |
| `GOBP_MOCK_AUTH_POOL_BASE` | `4000` | Base for `MOCK_AUTH_HOST_PORT` |
| `GOBP_DLV_POOL_BASE` | `2345` | Base for `DLV_HOST_PORT` |
| `GOBP_PPROF_POOL_BASE` | `6060` | Base for `PPROF_HOST_PORT` |
| `GOBP_DB_POOL_MAX` | `12` | Number of slots (= the cap on concurrent parallel work) |
| `GOBP_DB_POOL_TTL` | `1800` | Heartbeat grace period for the stale decision (seconds) |

## Caveats

- **Blast radius of a shared instance**: every checkout shares one Postgres / o11y / object storage.
  Databases are isolated, so DDL cannot cross a database boundary — but getting the target database
  name of a reinit wrong can destroy another worktree's. Concretely, `db-init` / `db-local-reinit` /
  `db-test-reinit` hardcode `DB=local` / `DB=test`, so running them while holding a slot rebuilds the
  shared databases rather than your worktree's. Name the target explicitly when rebuilding your own,
  e.g. `make db-reinit DB=$DB_NAME_LOCAL`. `make infra-down` likewise stops every checkout.
- **Schema generation cannot run concurrently**: `gen_schema` is a single database name inside the
  shared instance, so two checkouts running `make gen-query` (`dump-schema`) at the same time rebuild
  the same database and corrupt each other's output. Generate schemas one checkout at a time.
- **Parallel tests contend over establishing connections, not over capacity**: when tests run at the
  same time, packages unrelated to your change fail with `failed to ping DB`, while `too many clients`
  never appears. The instance has connections to spare; what saturates is how many are being
  *established* at the same instant, and the ping budget expires while they queue. It does not take two
  worktrees — lefthook's `pre-commit` / `pre-push` are `parallel: true`, so `make lint` and `make test`
  overlap inside a single checkout. The test path is already tuned against this: see `DBCONN_MIN_CONNS`
  and `DB_PING_TIMEOUT` in `env/README.md` for what `ci` sets and why, mirrored by the test
  configuration in `internal/config` for the paths that do not load an env file. To diagnose a
  recurrence: `pgrep -fl "go test"` then
  `lsof -a -p <pid> -d cwd` identifies which checkout is running tests, sampling `pg_stat_activity`
  shows whether the peak is a momentary spike rather than a plateau, and a re-run with `go test -p 1`
  that comes back green means the failure was load rather than the change under test.
- **Re-creation of the infra layer**: compose decides whether a container is current by hashing its
  resolved service definition, and that hash includes bind-mount sources and build contexts as
  *absolute* paths. Every worktree resolves them under its own directory, so the hash differs
  between checkouts of the very same commit — re-creation is the norm, not a branch-divergence edge
  case. `database` and `garage` are affected (they bind-mount `docker/database/sql` and
  `docker/garage/garage.toml`); `observability` is not, because it mounts nothing. An `up` against
  `gobp-shared` from a worktree therefore passes `--no-recreate` (`INFRA_NO_RECREATE` in
  `.makefiles/docker/compose.mk`), which keeps a container another checkout is using rather than
  replacing it. The cost is that a *legitimate* definition change — a new image digest pin, an
  edited `garage.toml` — no longer takes effect on its own: run `make infra-down && make infra-up`
  at a point where every checkout can afford the interruption. The same applies to the `tools`
  profile, so `docs_viewer` keeps serving the `docs/` of whichever checkout first created it.
  A single checkout has no one to contend with, so the flag stays empty there and compose
  re-converges on a definition change as usual.
- **Object storage is shared**: the `garage` bucket is common to every checkout (unlike a database it
  has no schema, so it does not break across branches). Point a branch at a different
  `OBJECT_STORAGE_BUCKET` to isolate it. The access key is shared the same way — `garage_init` imports
  it under one fixed key name from the running checkout's `env/.env`, so a branch that edits
  `OBJECT_STORAGE_ACCESS_KEY_ID` / `_SECRET_ACCESS_KEY` changes what every other checkout authenticates
  with. Isolate by bucket, not by credential.
- **The queue is shared and cannot be isolated by configuration alone**: `elasticmq` serves one set of
  queues to every checkout, and unlike the object-storage bucket, pointing `OUTBOX_QUEUE_URL` at a
  different name is not enough — ElasticMQ creates only the queues declared in
  `docker/elasticmq/elasticmq.conf` and expands no environment variables, so a name nothing declared
  is simply absent. Two checkouts running `make outbox-relay` at once therefore publish into the same
  queue, and whichever consumer reads first takes the message. Nothing consumes it today
  (`provideWorkers()` is empty), so the overlap is invisible until a worker is wired. To isolate a
  branch, declare an extra queue in `elasticmq.conf` and point `OUTBOX_QUEUE_URL` at it — the conf is
  read at start-up, so the change needs `make infra-down && make infra-up`, which interrupts every
  checkout. Per-slot queues are not pre-declared because the pool size is configurable
  (`GOBP_DB_POOL_MAX`), and a static list in the conf would silently stop covering the pool as soon as
  that value changed.
- `sql_editor` / `docs_viewer` have been moved into the 7000 range so they do not collide with the
  API band 8080–8092.
- The wiring spans `docker/`, `internal/cli/dbslot`, and `.makefiles/`, so update this document
  whenever it changes.
