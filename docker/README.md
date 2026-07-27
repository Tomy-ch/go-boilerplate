# docker

English | [日本語](README.ja.md)

`docker/` is the directory for **Docker-related configuration files** required for development, building, and operations.

## Directory Structure

```text
docker/
├── server/             # Application server Dockerfile
├── tools/              # Code generation / tool runner Dockerfile
├── document/           # Documentation viewer Dockerfile + nginx config
├── garage/             # Object storage Dockerfile + config + provisioning script
├── mock-auth-server/   # Mock OIDC auth server Dockerfile + application
└── database/
    ├── sql/            # DB initialization SQL
    ├── schemaspy/      # ER diagram generation config
    └── sqlfluff/       # SQL linter config
```

## Compose Layering (infra / app)

Compose services are split into two layers, so the main checkout and any number of worktrees can run side by side.

|Layer|compose project|Services|
|---|---|---|
|**infra**|`gobp-shared` (fixed name) — **one instance for every checkout**|`database` / `observability` / `garage` (+ `garage_init`). The auxiliary services and tool runners default to the same project via `COMPOSE_PROJECT_NAME`|
|**app**|`APP_PROJECT` — one per checkout (`gobp-app-<directory name>`, or `gobp-wt-N` while a DB slot is held)|`api_server` / `mock_auth_server`|

The infra layer holds every service that can only run on a fixed host port, which is why it exists exactly once on the host.

|File|Role|
|---|---|
|`docker-compose.yaml`|Definitions of all services|
|`docker-compose.attach.yaml`|app layer override, **always** overlaid (`docker compose -f docker-compose.yaml -f docker-compose.attach.yaml`)|

`docker-compose.attach.yaml` publishes the `api_server` host ports as `${API_HOST_PORT:-8080}` / `${DLV_HOST_PORT:-2345}` / `${PPROF_HOST_PORT:-6060}`, narrows `depends_on` to `mock_auth_server` alone (the infra layer is already up), and points the app at the shared infra by overriding `DB_HOST=host.docker.internal` / `DB_NAME=${DB_NAME_LOCAL:-local}` / `OBS_OTLP_ENDPOINT=http://host.docker.internal:4318` / `OBJECT_STORAGE_ENDPOINT=http://host.docker.internal:3900` / `AUTH_ISSUER=http://localhost:${MOCK_AUTH_HOST_PORT:-4000}`. `mock_auth_server`'s `OIDC_ISSUER` follows the same published port.

|Purpose|Reference|
|---|---|
|Layer variables (`INFRA_PROJECT` / `APP_PROJECT` / `COMPOSE_INFRA` / `COMPOSE_APP`)|[`.makefiles/docker/compose.mk`](../.makefiles/docker/compose.mk)|
|Local development topology as a whole|[`docs/maintenance/local-environment.md`](../docs/maintenance/local-environment.md)|
|Slot assignment of host ports / DB names|[`docs/maintenance/db-worktree-pool.md`](../docs/maintenance/db-worktree-pool.md)|

## Service Overview

The following table maps services defined in docker-compose.yaml to their corresponding Dockerfile / configuration.

### Development Environment (profile: `development`)

|Service|Layer|Dockerfile / Image|Port|Description|
|---|---|---|---|---|
|`api_server`|app|`docker/server/Dockerfile` (target: `tooling`)|`${API_HOST_PORT:-8080}`, `${DLV_HOST_PORT:-2345}`, `${PPROF_HOST_PORT:-6060}`|Development API server (hot reload via air)|
|`mock_auth_server`|app|`docker/mock-auth-server/Dockerfile`|`${MOCK_AUTH_HOST_PORT:-4000}`|Mock OIDC auth server (JWT test provider)|
|`database`|infra|`postgres:18.4-trixie`|5432|PostgreSQL database|
|`observability`|infra|`grafana/otel-lgtm`|3000, 4317, 4318, 3200|Local observability stack (OTLP endpoint / Grafana) for o11y verification|
|`garage`|infra|`dxflrs/garage`|3900, 3902|S3-compatible object storage (S3 API / Web API)|
|`garage_init`|infra|`docker/garage/Dockerfile`|-|One-shot provisioning of the garage layout / bucket / access key / website access (idempotent)|

The app layer host ports are the ones a DB slot shifts (`8080+N` / `4000+N` / `2345+N` / `6060+N`); the container-internal ports never move.

### Auxiliary Services (profile: `tools`, infra layer)

|Service|Dockerfile / Image|Port|Description|
|---|---|---|---|
|`docs_viewer`|`docker/document/Dockerfile`|7001|Documentation portal (nginx)|
|`sql_editor`|`sosedoff/pgweb`|7000|Web SQL editor|

### Tool Runners (profile: `generate`, infra layer)

|Service|Dockerfile / Image|Description|
|---|---|---|
|`go_tool_runner`|`docker/tools/Dockerfile` (target: `go_tools`)|oapi-codegen, mockgen, sqlc, migrate, trivy, actionlint, hadolint, gitleaks, godoc, godoc-static|
|`node_tool_runner`|`docker/tools/Dockerfile` (target: `node_tools`)|redocly-cli|
|`python_tool_runner`|`docker/tools/Dockerfile` (target: `python_tools`)|sqlfluff|
|`er_diagram_generator`|`schemaspy/schemaspy`|ER diagram generation (SchemaSpy)|

## server

The Dockerfile for the application server. Provides the following targets via multi-stage build.

|Target|Purpose|Base Image|
|---|---|---|
|`builder`|Go binary build|`golang:1.26.5-alpine`|
|`runtime`|Production runtime container|`alpine:3.23`|
|`tooling`|Local development environment|`golang:1.26.5-alpine`|

### runtime

- Runs as non-root user (`app`)
- Embeds version / revision / build date via `ldflags`
- Builds in `vendor` mode (`GOPROXY=off`)
- Migrations run via a command override on the same image (`./server migrate-up`); there is no dedicated migration image

### tooling

Includes the following tools for local development.

|Tool|Purpose|
|---|---|
|`air`|Hot reload|
|`dlv`|Debugger|
|`golines`|Line-length-limited formatting|
|`gofumpt`|Enhanced gofmt|
|`golangci-lint`|Go linter|

## tools

Tool containers for code generation and bundling. Split into three stages.

|Stage|Base|Included Tools|
|---|---|---|
|`go_tools`|`golang:1.26.5-alpine`|oapi-codegen, mockgen, sqlc, migrate, trivy, actionlint, hadolint, gitleaks, godoc, godoc-static|
|`node_tools`|`node:24.18.0-alpine`|redocly-cli, js-yaml|
|`python_tools`|`python:3.14.6-slim`|sqlfluff|

## document

Container for the documentation portal.

- Base image: `nginx:1.29-otel`
- Volume mounts the entire `docs/` directory
- Portal app is served at `/portal/`
- Accessing `http://localhost:7001/` redirects to `/portal/`

## garage

S3-compatible object storage for local development (tests use in-process gofakes3 instead).

- The server itself runs the official `dxflrs/garage` image. `garage_init` is the one that needs a build: the official image is `scratch`-based and has no shell, so the provisioning script cannot run in it — the Dockerfile copies the `garage` binary onto `alpine:3.24` for that
- `garage.toml`: single-node configuration (`replication_factor = 1`, S3 API `3900` / Web API `3902`), mounted read-only
- `init.sh`: idempotent provisioning run by the `garage_init` service (layout assignment → bucket creation → fixed access key import → bucket permission → website permission), mounted the same way so that editing it needs no image rebuild. It reads `OBJECT_STORAGE_*` straight from `env/.env` (passed via `env_file`) and fails immediately if one is unset, so the bucket and key can never drift from what the Go side connects with

### Public delivery (anonymous read)

The Web API (`3902`, Garage's `[s3_web]`) serves bucket objects **without credentials**, so a browser can load
product images straight from the object storage, the way a CDN fronts S3 in production. Writing still goes through
`POST /v1/products/images` (BearerAuth + admin); only reading is open.

- Delivery origin: `http://gobp-local.web.garage.localhost:3902` — an object is `<origin>/<object key>`, e.g.
  `http://gobp-local.web.garage.localhost:3902/products/{uuid}.png`. This is the value the frontend puts in its
  media-origin setting; the API never returns a full URL, only the object key (`imagePath`)
- **Virtual-host addressing only.** Garage's web endpoint resolves the bucket from the `Host` header
  (`<bucket>.<root_domain>` or `<bucket>`), so path style (`localhost:3902/<bucket>/<key>`) does not work.
  macOS and the major browsers resolve `*.localhost` to `127.0.0.1` on their own, so no `/etc/hosts` entry is
  needed. From inside a glibc Linux container, which does not, either send the header explicitly
  (`curl -H 'Host: gobp-local' http://<host>:3902/products/...`) or add `127.0.0.1 gobp-local.web.garage.localhost`
  to `/etc/hosts`
- Listing stays closed: the web endpoint never lists, and an anonymous `ListObjects` against the S3 API is
  unsigned and rejected. Object keys carry a UUIDv7, whose ~74 random bits put enumeration out of reach —
  but note that the remaining bits are a millisecond timestamp, so a key is opaque, not secret. Treat
  anything in this bucket as world-readable to whoever holds the key
- **Website permission is per bucket, not per object** — every object in `gobp-local` becomes anonymously
  readable. The bucket holds nothing but product images; storing non-public objects would require a second bucket

## mock-auth-server

Mock OIDC auth server (JWT test provider) container. Endpoints, flows, and fixtures: [`docker/mock-auth-server/README.md`](mock-auth-server/README.md).

- Base image: `node:24.18.0-alpine`, running the `.ts` sources directly via Node's native type stripping (no `tsc` build step)
- Runs as non-root user (`node`); the internal port is always `4000`
- `OIDC_ISSUER` is the URL as seen from the host OS / browser, so `docker-compose.attach.yaml` makes it follow the published host port

## database

### sql

Stores DB initialization SQL files.

- Executed via `docker-entrypoint-initdb.d` on PostgreSQL container startup
- Executed in order of filename numeric prefix (e.g., `001-...`, `002-...`)
- DDL (table definitions) should not be placed here; use `database/migrations/` instead

### schemaspy

Connection configuration for the ER diagram generator (SchemaSpy).

|File|Purpose|
|---|---|
|`schemaspy.properties`|Local environment (host: `database`)|
|`schemaspy-ci.properties`|CI environment (host: `localhost`)|

### sqlfluff

Configuration files for the SQL linter (sqlfluff). Different rules are applied per target.

|File|Target|Notes|
|---|---|---|
|`.dml.sqlfluff`|`database/dml/` (sqlc queries)|`@param` placeholder support, some rules excluded|
|`.migrations.sqlfluff`|`database/migrations/`|Max line length 150|
|`.seed.sqlfluff`|Seed data|Max line length 500|

Common rules

- Dialect: `postgres`
- Keywords: uppercase
- Identifiers: lowercase
- Functions / literals / types: uppercase
