---
name: repo-ops
description: Operational runbook for this repository's recurring, easy-to-trip-on gotchas around generated artifacts, the dockerized tool runners, and the DB. Use when a generated file drifts (`schema.gen.sql`, mocks, portal guides), when `git` suddenly refuses to touch a generated directory because it became root-owned, when a regeneration step fails because the DB container is down or carries stale tables, when building a per-environment image, or when the commit-msg hook errors. The repo runs most code generation inside Docker tool-runner containers as root, which is the root cause of several of these. Read-only knowledge skill — it tells you the exact command to run; it does not silently mutate state. Triggers: "schema.gen.sql が diff る / generate-db-check が落ちる", "git が docs/portal や mock を permission denied", "gen-query が DB に繋がらず落ちる", "per-env イメージのビルド", "commit-msg / commitlint が not found".
---

# Repo Ops Runbook

Concrete recovery + procedure steps for the operational gotchas that recur in this repo. Most stem from one fact: **`make gen-*` runs inside Docker tool-runner containers (`node_tool_runner` / `go_tool_runner` / `python_tool_runner`) as root**, and the DB-backed generators dump the **live** database. Knowing that explains every item below.

This skill is a lookup table, not a workflow. Find the symptom, run the fix. When a step is destructive (drops DB data, `chown`s a tree) say so to the user first per `CLAUDE.md`.

## 1. `schema.gen.sql` drift → `generate-db-check` / `gen-*-artifacts-check` fails

`database/gen/schema.gen.sql` is produced by `make dump-schema`, which `pg_dump`s the **running** DB. Any change to DB-init SQL (extensions in `docker/database/*.sql`) or to `database/migrations/**` changes the dump, so the committed `schema.gen.sql` drifts and the CI check (`dump-schema` → `git diff`) fails.

Fix: regenerate and commit it.

```bash
docker compose up -d database          # DB must be running
make dump-schema                       # rewrites database/gen/schema.gen.sql from the live DB
git add database/gen/schema.gen.sql
```

Rule of thumb: **any change under `docker/database/`, `database/migrations/`, or DB extensions requires regenerating `schema.gen.sql` and committing it in the same change.**

## 2. `git` refuses a generated directory — it became root-owned

`make gen-portal-docs` (and other docker-root generators) write `docs/portal/{guides,docs.json}` as **root** because the container runs as root over the `.:/app` bind mount. Afterwards host `git add` / `git restore` fail with `permission denied`. The same happens to freshly-created mock directories from `make gen-go-code` (e.g. `pkg/fs/mock`, `pkg/exec/mock`).

Fix: hand ownership back via a container (don't `sudo` unless you must):

```bash
# portal output
docker compose run --rm --user root node_tool_runner chown -R $(id -u):$(id -g) /app/docs/portal
# a root-owned mock dir
docker compose run --rm --user root go_tool_runner chown -R $(id -u):$(id -g) /app/pkg/fs/mock /app/pkg/exec/mock
```

Trap: `git restore docs/portal` to clean up root-owned files will also **revert any hand edits** you made under `docs/portal` — re-apply / re-sync those in a separate commit after the chown.

## 3. A `make gen-*` step needs the DB up and reset

`make gen-query` chains `dump-schema` (see §1), so it needs the **DB container running** — otherwise it dies with `connection refused`. It also reflects whatever tables currently exist: if you deleted source for a table but did not drop the table, the dump still contains it and the generated models keep the dead type.

Procedure when regenerating after a schema-affecting change:

```bash
docker compose up -d database
make db-local-migrate-down db-test-migrate-down   # drop to clean (destructive — confirm with user)
make db-init-local db-init-test                   # re-migrate from the current migration set (+ seed)
make gen-query                                     # now dumps the intended schema
```

For DB-layer integration tests (`internal/infrastructure/rdb/repository`, `query_service`), the **test DB must be migrated + seeded** first: after any reset, run migrate-up + seed for **both** local and test. To run the binary from the host against the docker DB, override the host: `DB_HOST=localhost go run ./cmd/ ...`.

## 4. Per-environment Docker images

The runtime image bakes a single `env/.env.<env>` chosen at build time via `--build-arg APP_ENV=<env>` (the deploy workflow injects it). There is no single image switched by a runtime ENV.

```bash
docker build --build-arg APP_ENV=stg --target runtime -t <img> -f docker/server/Dockerfile .
# verify only .env.stg was baked in
```

There is no separate migration image: `env/` and `database/migrations` are embedded in the
binary, so migrations run from the same `runtime` image via a command override
(`./server migrate-up`).

## 5. commit-msg hook / commitlint errors

The lefthook **commit-msg** hook runs `make commitlint COMMIT_MSG_FILE={1}`, which executes `commitlint` inside the `node_tool_runner` container — tools are resolved in the containerized runners, never on the host (the same reproducibility rule as every other lint / codegen target; see `docs/rules.md`). If the `node_tool_runner` image is missing or out of date, the hook fails with `commitlint: not found`; rebuild it once with `make tool-runners-build` (or `docker compose build node_tool_runner`).

```text
make commitlint COMMIT_MSG_FILE={1}
```

`commitlint.config.js` intentionally disables `type-case` (the repo prefixes are Cap-first `Feat`/`Fix`/… but CI uses all-caps, so a single case can't be enforced) and pins `type-enum` to the project prefixes. `Merge` / `Revert` commits are auto-ignored by commitlint defaults.

## 6. Generated mocks — never hand-write, regenerate via Docker

Mocks are generated, never authored by hand. Each source declares `//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE` (the `$GOFILE`-based destination is uniform across the repo, so the same line copies into any interface file), and `make gen-go-code` runs them in `go_tool_runner` with the **pinned** mockgen (`v0.6.0`). Regenerated mock directories may come back root-owned (see §2).

After changing an interface (signature, new method), regenerate rather than editing `*_mock.go`:

```bash
make gen-go-code        # runs go:generate (mockgen) in docker, pinned version
```

## 7. The `.makefiles` SETUP_DRY_RUN convention

The setup / teardown make targets gate dry-run on `$(if $(DRY_RUN),--dry-run,)` plus `[ -n "$(DRY_RUN)" ]`, which treat **any non-empty value as truthy** — `DRY_RUN=0 make <target>` is still a dry-run. To actually run, omit the variable entirely (`make <target>`). To preview, `DRY_RUN=1 make <target>`.

## Constraints

- ✅ Read-only knowledge: surface the exact command; run it only when the user asked you to perform the operation.
- ✅ Warn before destructive steps (DB drops in §3, `chown -R` in §2) per `CLAUDE.md`.
- ✅ Prefer the dockerized `--user root … chown` recovery over host `sudo`.
- ❌ Do not hand-edit generated artifacts (`*.gen.go`, `*.sql.go`, `*_mock.go`, `openapi.gen.yaml`, `schema.gen.sql`, `docs/portal/guides/**`) — regenerate instead.
- ❌ Do not commit a schema-affecting change without regenerating `schema.gen.sql` (§1).
