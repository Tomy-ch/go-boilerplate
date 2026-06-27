# Make Command List

English | [日本語](README.ja.md)

## Role

`.makefiles/` is the central registry for every `make` target used by the project. Each `.mk` file groups related targets by area (application / database / sql / go / openapi / docs / github / tools). The top-level `makefile` simply `include`s them, so adding a new target means dropping it into the right group file — no top-level edits required.

Make targets are mainly organized into the following units.

- `.makefiles/app` : Application startup / Job execution / Embedded env materialization
- `.makefiles/database` : DB initialization / Migration / Seed / DML / Schema
- `.makefiles/sql` : SQL Lint / Fix
- `.makefiles/markdown` : Markdown Lint / Fix
- `.makefiles/security` : Trivy dependency vulnerability scan
- `.makefiles/openapi` : OpenAPI bundle / API documentation generation
- `.makefiles/go` : Go code generation / Format / Lint / Test / Tool management
- `.makefiles/docs` : Portal / Tool information documentation generation
- `.makefiles/gen` : Batch execution of various generation processes
- `.makefiles/github` : GitHub initialization / Release / Labels / Rule configuration

## Conventions

- Target names use dash-separated lower case (`make new-migrate-<name>`, `make gen-api`).
- Targets are split into two flavors:
  - **Normal targets**: invoked by developers locally; run inside Docker containers for reproducibility.
  - **`-ci` targets**: low-level commands intended to run on bare metal (CI runners, or developers who already have the tool installed).
- Every target should be `.PHONY` and self-documenting via a trailing `##` comment so `make help` can pick it up.

## Notes

- Adding a new `.mk` file under a group is enough — `makefile` already `include`s all known groups.
- Prefer `make new-migrate-<name>` (and similar helpers) over manual file creation; the helpers enforce naming conventions and number sequences.
- For one-off operational commands (`make setup-repo`, etc.) keep them under `.makefiles/github/operation/` so they stay separate from developer-facing targets.

## `.makefiles/app` group

This is a group of targets related to application development environment startup and Job execution.

### Application startup related

| Command | Description | Main Use |
| --- | --- | --- |
| `make serve` | Starts Docker Compose services with the `development` profile in the background. | Start normal local development |
| `make serve-build` | Rebuilds Docker images (cache enabled) and then starts the development environment. | Reflect Dockerfile or dependency changes |
| `make serve-build-clean` | Cleanly rebuilds Docker images with `--no-cache --pull` and then starts the development environment. | Pick up base image updates (e.g., Go version upgrade) |
| `make tools` | Starts development support tools with the `tools` profile. | When using development tools |
| `make tool-runners-build` | Builds the on-demand tool runner images (go/node/python, cache enabled, no startup). | When updating tool runner Dockerfile or dependencies |
| `make tool-runners-build-clean` | Cleanly builds the tool runner images with `--no-cache --pull` (no startup). | Pick up base image updates for tool runners |

#### `make job NAME=<job_name> ARGS="<arguments>"`

Executes an application Job.
Calls `cmd/main.go job` within the `development` profile network.

- `NAME`: Job name to execute
- `ARGS`: Additional arguments passed to the Job (optional)

Example:

```sh
make job NAME=sample-job
make job NAME=batch-import ARGS="--target=local --dry-run"
```

### Embedded env materialization related

The server binary embeds `env/.env`. CI and the Docker build materialize the
per-environment file into `env/.env` before building, so these targets centralize
that step (and its undo for drift checks).

| Command | Description | Main Use |
| --- | --- | --- |
| `make materialize-env` | Copies `env/.env.$(APP_ENV)` over `env/.env` (defaults to `APP_ENV=ci`). | Materialize the embed target in CI / build before `go build` / `go run` |
| `make restore-env` | Restores `env/.env` to its git-tracked content via `git restore`. | Undo materialization before a generated-artifact drift / commit check |

## `.makefiles/database` group

This is a group of targets that handle all DB operations.
Provides migration, seed insertion, DML merge, schema generation, DB initialization, etc.

### DB initialization related

| Command | Description | Notes |
| --- | --- | --- |
| `make db-init` | Executes initialization for both LocalDB and TestDB. | Calls `db-init-local` and `db-init-test` sequentially. |
| `make db-init-local` | Initializes LocalDB. | Executes `db-local-migrate-down` → `db-local-migrate-up` → `db-local-seed`. |
| `make db-init-test` | Initializes TestDB. | Executes `db-test-migrate-down` → `db-test-migrate-up` → `db-test-seed`. |

### DB migration related

| Command | Description | Notes |
| --- | --- | --- |
| `make new-migrate-<name>` | Generates a new migration file. | Creates numbered `.up.sql` / `.down.sql` under `database/migrations`. |
| `make check-migration-up-version` | Checks duplicate versions in `up` migrations. | None |
| `make check-migration-down-version` | Checks duplicate versions in `down` migrations. | None |
| `make check-migration-up-gap` | Checks sequence gaps in `up` migrations. | None |
| `make check-migration-down-gap` | Checks sequence gaps in `down` migrations. | None |
| `make db-migrate-up DB=<database>` | Applies all migrations to the specified DB up to the latest. | Example: `make db-migrate-up DB=local` |
| `make db-migrate-up-<steps> DB=<database>` | Applies the given number of migrations relative to the current position. | Example: `make db-migrate-up-2 DB=local` |
| `make db-migrate-down DB=<database>` | Downgrades all migrations to the initial state. | None |
| `make db-migrate-down-<steps> DB=<database>` | Rolls back the given number of migrations. | None |
| `make db-local-migrate-up` | Applies all migrations to LocalDB. | Alias for `db-migrate-up` with `DB=local`. |
| `make db-local-migrate-up-<steps>` | Applies the given number of migrations on LocalDB. | None |
| `make db-local-migrate-down` | Downgrades LocalDB to initial state. | Alias for `db-migrate-down` with `DB=local`. |
| `make db-local-migrate-down-<steps>` | Rolls back the given number of migrations on LocalDB. | None |
| `make db-test-migrate-up` | Applies all migrations to TestDB. | Alias for `db-migrate-up` with `DB=test`. |
| `make db-test-migrate-up-<steps>` | Applies the given number of migrations on TestDB. | None |
| `make db-test-migrate-down` | Downgrades TestDB to initial state. | Alias for `db-migrate-down` with `DB=test`. |
| `make db-test-migrate-down-<steps>` | Rolls back the given number of migrations on TestDB. | None |
| `make db-migrate-ci-up DB=<database>` | Executes `cmd/main.go migrate-up` directly without Docker. | CI target |
| `make db-migrate-ci-up-<steps> DB=<database>` | Executes `migrate-up` for the given number of steps without Docker. | CI target |
| `make db-migrate-ci-down DB=<database>` | Executes `cmd/main.go migrate-down` directly without Docker. | CI target |
| `make db-migrate-ci-down-<steps> DB=<database>` | Executes `migrate-down` for the given number of steps without Docker. | CI target |

Example:

```sh
make new-migrate-create_users_table
make db-migrate-up DB=local
make db-migrate-up-10 DB=local
```

### DB seed related

| Command | Description | Notes |
| --- | --- | --- |
| `make db-seed DB=<database>` | Inserts seed data into the specified DB. | Executes `cmd/main.go db-seed` inside a Docker container. |
| `make db-seed-ci DB=<database>` | Executes seed insertion directly without Docker. | CI target |
| `make db-local-seed` | Inserts seed data into LocalDB. | Alias for `db-seed` with `DB=local`. |
| `make db-test-seed` | Inserts seed data into TestDB. | Alias for `db-seed` with `DB=test`. |

### DB generation / helper related

| Command | Description | Notes |
| --- | --- | --- |
| `make gen-db-schema` | Generates DB schema documentation. | Used to update ER diagrams and schema outputs. |
| `make gen-db-schema-ci` | Executes SchemaSpy container directly to generate schema docs. | CI target |
| `make dump-schema` | Executes schema dump. | Used as preprocessing for SQLC generation and DML merge. |
| `make dump-schema-ci` | Executes `cmd/main.go dump-schema` directly without Docker. | CI target |
| `make fix-collation` | Fixes database collation. | None |
| `make fix-collation-ci` | Executes collation fix directly without Docker. | CI target |

### DML merge related

| Command | Description | Notes |
| --- | --- | --- |
| `make merge-dml` | Executes all DML merge processes. | Executes `merge-dml-repo` → `merge-dml-qs` → `merge-dml-sysq` → `merge-dml-cs`. |
| `make merge-dml-repo` | Merges DML for Repository. | None |
| `make merge-dml-qs` | Merges DML for Query Service. | None |
| `make merge-dml-cs` | Merges DML for Command Service. | None |
| `make merge-dml-sysq` | Merges DML for System Query. | None |
| `make merge-dml-core type="<type>" work-dir="<dir>"` | Executes DML merge for specified type. | Calls `make merge-dml-ci-core` via Docker container. |
| `make merge-dml-ci` | Executes all DML merge processes directly. | CI target |
| `make merge-dml-ci-repo` | Merges DML for Repository. | CI target |
| `make merge-dml-ci-qs` | Merges DML for Query Service. | CI target |
| `make merge-dml-ci-cs` | Merges DML for Command Service. | CI target |
| `make merge-dml-ci-sysq` | Merges DML for System Query. | CI target |
| `make merge-dml-ci-core type="<type>" work-dir="<dir>"` | Executes `cmd/main.go merge-dml` directly. | CI target |

Example:

```sh
make merge-dml-core type="repository" work-dir="/app"
```

## `.makefiles/sql` group

This group handles static analysis and auto-fix for SQL files.
Targets include Migration / DML / Seed SQL.

### SQL Lint related

| Command | Description | Notes |
| --- | --- | --- |
| `make sql-lint` | Executes SQL lint in batch. | Executes `sql-lint-migrations` → `sql-lint-dml` → `sql-lint-seed`. |
| `make sql-lint-migrations` | Lints migration SQL. | None |
| `make sql-lint-dml` | Lints DML SQL. | None |
| `make sql-lint-seed` | Lints seed SQL. | None |
| `make sql-lint-migrations-ci` | Executes `sqlfluff lint` on `database/migrations/`. | CI target |
| `make sql-lint-dml-ci` | Executes `sqlfluff lint` on `database/dml/`. | CI target |
| `make sql-lint-seed-ci` | Executes `sqlfluff lint` on `database/seed/`. | CI target |

### SQL Fix related

| Command | Description | Notes |
| --- | --- | --- |
| `make sql-fix` | Executes SQL auto-fix in batch. | Executes `sql-fix-migrations` → `sql-fix-dml` → `sql-fix-seed`. |
| `make sql-fix-migrations` | Auto-fixes migration SQL. | None |
| `make sql-fix-dml` | Auto-fixes DML SQL. | None |
| `make sql-fix-seed` | Auto-fixes seed SQL. | None |
| `make sql-fix-migrations-ci` | Executes `sqlfluff fix` on `database/migrations/`. | CI target |
| `make sql-fix-dml-ci` | Executes `sqlfluff fix` on `database/dml/`. | CI target |
| `make sql-fix-seed-ci` | Executes `sqlfluff fix` on `database/seed/`. | CI target |

## `.makefiles/markdown` group

This group handles linting and auto-fixing of Markdown files.

| Command | Description | Notes |
| --- | --- | --- |
| `make md-lint` | Lints Markdown (markdownlint + mermaid syntax). | Invokes `make md-lint-ci` inside the `node_tool_runner` container. |
| `make md-fix` | Auto-fixes Markdown files. | Invokes `make md-fix-ci` inside the `node_tool_runner` container. |
| `make md-mermaid-lint` | Validates only the ` ```mermaid ` fences. | Invokes `make md-mermaid-lint-ci` inside the `node_tool_runner` container. |
| `make md-lint-ci` | Runs `markdownlint-cli2` then the mermaid syntax lint. | CI target. Excludes `vendor/`, `node_modules/`, `.git/`. |
| `make md-mermaid-lint-ci` | Validates ` ```mermaid ` fences with `scripts/mermaid-lint.mjs` (real `mermaid.parse`). | CI target. markdownlint never checks diagram grammar. |
| `make md-fix-ci` | Fixes `**/*.md` directly with `markdownlint-cli2 --fix`. | CI target. Excludes `vendor/`, `node_modules/`, `.git/`. |

## `.makefiles/security` group

This group runs local security scans (Trivy dependency scan, gitleaks secret scan), mainly to reproduce a CI security finding on the developer's machine. Image scanning is CI-only (`image-scan.yaml`).

| Command | Description | Notes |
| --- | --- | --- |
| `make trivy-fs` | Scans library dependencies with Trivy fs. | Invokes `make trivy-fs-ci` inside the `go_tool_runner` container. |
| `make trivy-fs-ci` | Runs `trivy fs` directly. | CI target. Skips `vendor/` to match CI. |
| `make secret-scan` | Scans the working tree for secrets with gitleaks. | Invokes `make secret-scan-ci` inside the `go_tool_runner` container. |
| `make secret-scan-ci` | Runs `gitleaks dir . --redact` directly. | CI target. Generated files are allowlisted in `.gitleaks.toml`. |

## `.makefiles/docker` group

This group lints Dockerfiles with hadolint via the `go_tool_runner` container.

| Command | Description | Notes |
| --- | --- | --- |
| `make docker-lint` | Lints `docker/*/Dockerfile` with hadolint. | Invokes `make docker-lint-ci` inside the `go_tool_runner` container. |
| `make docker-lint-ci` | Runs `hadolint docker/*/Dockerfile` directly. | CI target. Ignored rules are in `.hadolint.yaml`. |

## `.makefiles/openapi` group

| Command | Description | Notes |
| --- | --- | --- |
| `make gen-bundle-oapi` | Bundles split OpenAPI definitions into a single file. | Generates `openapi/openapi.gen.yaml` from `openapi/openapi.yaml`. |
| `make gen-api-docs` | Generates API documentation from OpenAPI definition. | None |
| `make gen-bundle-oapi-ci` | Generates `openapi/openapi.gen.yaml` via `redocly bundle`. | CI target |
| `make gen-api-docs-ci` | Generates `docs/openapi/index.html` via `redocly build-docs`. | CI target |

## `.makefiles/go` group

### Go generation related

| Command | Description | Notes |
| --- | --- | --- |
| `make gen-go-code` | Executes Go code generation. | Runs `go generate ./...` inside a Docker container. |
| `make gen-go-code-ci` | Executes `go generate ./...` directly without Docker. | CI target |
| `make gen-sqlc` | Executes SQLC code generation in batch. | Executes `remove-generated-sqlc` → `sqlc-generate`. |
| `make remove-generated-sqlc` | Deletes existing SQLC generated code. | None |
| `make sqlc-generate` | Executes SQLC code generation. | None |
| `make remove-generated-sqlc-ci` | Deletes `*.gen.sql.go` under `$(SQLC_OUT)`. | CI target |
| `make sqlc-generate-ci` | Executes `sqlc generate -f sqlc.yaml` directly. | CI target |

### Go format / lint / dependency update related

| Command | Description | Notes |
| --- | --- | --- |
| `make fmt` | Formats Go code. | Executes `go fmt ./...`. |
| `make lint` | Executes static analysis via GolangCI-Lint. | None |
| `make fix` | Executes auto-fix via GolangCI-Lint. | None |
| `make tidy-lib` | Cleans Go module dependencies and updates `vendor`. | Executes `go mod tidy` and `go mod vendor`. |

### Go test related

| Command | Description | Notes |
| --- | --- | --- |
| `make test` | Executes tests for CI. | Runs `go test` on packages excluding `gen` / `cmd` / `mock` / `apperror` / `scripts` (the `internal/cli` core is now included). |
| `make gen-test-repo` | Executes tests and generates HTML coverage report. | Output is `docs/coverage/index.html`. |
| `make test-cover-ci` | Executes tests with coverage. | CI target, outputs `coverage.out`. |
| `make cover-gate` | Fails if total coverage is below the threshold. | CI gate. `COVERAGE_THRESHOLD` (default 90). Requires `coverage.out` (run `test-cover-ci` first). |

### Go tool installation related

| Command | Description | Notes |
| --- | --- | --- |
| `make go-update` | Installs the Go runtime pinned in `mise.toml` via mise. See `docs/maintenance/go-upgrade.md`. | mise required |
| `make install-tools` | Installs host development Go tools via mise (versions from `mise.toml`). | Installs `gopls`, `gotests`, `impl`, `dlv`, `lefthook`, `golangci-lint`. |
| `make activate-tools` | Executes `lefthook install` to set up Git hooks. | None |

## `.makefiles/docs` group

| Command | Description | Notes |
| --- | --- | --- |
| `make gen-portal-docs` | Generates Portal documentation. | None |
| `make gen-docs-json` | Generates Portal documentation link JSON. | None |
| `make gen-portal-build` | Bundles the Portal frontend (`docs/portal/src/main.jsx`) into `bundle.js` / `bundle.css` via esbuild. | None |
| `make gen-portal-docs-ci` | Generates Portal documentation directly via Node.js script. | CI target |
| `make gen-docs-json-ci` | Generates Portal JSON directly via Node.js script. | CI target |
| `make gen-portal-build-ci` | Runs esbuild directly to bundle the Portal frontend. | CI target |
| `make gen-godoc` | Generates static godoc HTML into `docs/godoc/`. | None |
| `make gen-godoc-ci` | Runs godoc-static directly to generate static HTML. | CI target |

## `.makefiles/gen` group

| Command | Description | Notes |
| --- | --- | --- |
| `make gen` | Executes all code and documentation generation in batch. | Executes `gen-api` → `gen-query` → `gen-docs`. |
| `make gen-api` | Executes API-related generation in batch. | Executes `gen-bundle-oapi` →  `gen-api-docs` → `gen-go-code`. |
| `make gen-docs` | Executes documentation-related generation in batch. | Executes `gen-api-docs`, `gen-portal-docs`, `gen-docs-json`. |
| `make gen-all-docs` | Executes all documentation generation processes. | Executes `gen-docs`, `gen-db-schema`, `gen-test-repo`. |
| `make gen-query` | Executes SQLC code generation in batch. | Executes `dump-schema` → `merge-dml` → `gen-sqlc` → `fmt`. |
| `make gen-query-repo` | Executes SQLC code generation for Repository. | Executes `dump-schema` → `merge-dml-repo` → `gen-sqlc`. |
| `make gen-query-qs` | Executes SQLC code generation for Query Service. | Executes `dump-schema` → `merge-dml-qs` → `gen-sqlc`. |
| `make gen-query-sysq` | Executes SQLC code generation for System Query. | Executes `dump-schema` → `merge-dml-sysq` → `gen-sqlc`. |

## `.makefiles/github` group

### GitHub Actions lint / pin related

| Command | Description | Notes |
| --- | --- | --- |
| `make actions-lint` | Lints workflow / composite-action definitions with actionlint. | Invokes `make actions-lint-ci` inside the `go_tool_runner` container. |
| `make actions-lint-ci` | Runs `actionlint` directly. | CI target. |
| `make pin-actions-resolve` | Resolves each `uses:` tag to its commit SHA and updates the `.github/actions-pin.toml` lockfile. | Quarantines refs younger than `PIN_ACTIONS_MIN_AGE_DAYS` (default 14; 0 disables). |
| `make pin-actions-apply` | Pins `uses:` to `@<sha> # <tag>` from the lockfile. | None |
| `make pin-actions-check` | Verifies `uses:` are pinned per the lockfile (no write). | CI / pre-commit gate. |

### GitHub configuration related

| Command | Description | Notes |
| --- | --- | --- |
| `make gh-login` | Logs in to GitHub using `gh` command. | Uses browser-based authentication. |
| `make delete-all-labels` | Deletes all existing labels in the GitHub repository. | None |
| `make create-default-labels` | Creates default labels based on `.github/settings/labels.json`. | None |
| `make apply-branch-protection` | Applies branch rules based on `.github/settings/branch-protection.json`. | None |

### GitHub repository initialization related

#### `make setup-repo`

Executes repository initialization in batch.
Performs the following in order.

- `gh` login
- Create and push initial tag `v0.0.0`
- Create branches `develop` / `staging` / `production`
- Set GitHub default branch
- Apply branch rule set
- Initialize labels

This is an initial setup command when launching a new repository as a boilerplate.

#### Setup helper commands

| Command | Description | Notes |
| --- | --- | --- |
| `make setup-replace-module OLD_MODULE=<old> NEW_MODULE=<new>` | Replaces Go module name in batch. | Updates `go.mod` and import paths using `node_tool_runner`. |
| `make setup-replace-app-metadata APP_NAME=<name> OPENAPI_TITLE=<title> COPILOT_TITLE=<title>` | Replaces application name and OpenAPI title in batch. | Reflected in README and OpenAPI definitions. |
| `make setup-replace-repository-reference REPOSITORY=<org/repo>` | Replaces repository references (GitHub URLs, etc.) in batch. | Updates links in README and documentation. |
| `make setup-replace-license-copyright COPYRIGHT_HOLDER=<name> [COPYRIGHT_YEAR=<year>]` | Updates LICENSE copyright notation. | Year is optional. |
| `make setup-remove-sample-api` | Removes the sample API (`user`/`product`/`order`) in batch. | Deletes via `node_tool_runner`, then runs `gen-api` → `gen-query` → `fix` → `lint`. **Requires the DB container (`database`) running** (`gen-query` dumps the live schema). After removal, rebuild with `make db-init-local db-init-test && make gen-query` so dropped tables don't linger in generated models. Use `DRY_RUN=1` to preview without changes — any non-empty value (including `0`) enables preview, so omit the variable to actually run. |

### Release branch related

| Command | Description | Notes |
| --- | --- | --- |
| `make hotfix-patch` | Creates a hotfix branch from `production` and sets it as the default branch. | Advances patch by one based on the latest tag. |
| `make branch-patch` | Creates a patch release branch from `production` and sets it as the default branch. | Advances patch version based on the latest tag. |
| `make branch-minor` | Creates a minor release branch from `production` and sets it as the default branch. | Advances minor version based on the latest tag. |
| `make branch-major` | Creates a major release branch from `production` and sets it as the default branch. | Advances major version based on the latest tag. |

### Release tag related

| Command | Description | Notes |
| --- | --- | --- |
| `make tag-patch` | Creates a tag with incremented patch version and creates a GitHub Release. | Uses `.github/release/<version>.md` for release notes. |
| `make tag-minor` | Creates a tag with incremented minor version and creates a GitHub Release. | Based on the latest tag. |
| `make tag-major` | Creates a tag with incremented major version and creates a GitHub Release. | Based on the latest tag. |
