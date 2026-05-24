# docker

English | [日本語](README.ja.md)

`docker/` is the directory for **Docker-related configuration files** required for development, building, and operations.

## Directory Structure

```text
docker/
├── server/             # Application server Dockerfile
├── tools/              # Code generation / tool runner Dockerfile
├── document/           # Documentation viewer Dockerfile + nginx config
└── database/
    ├── sql/            # DB initialization SQL
    ├── schemaspy/      # ER diagram generation config
    └── sqlfluff/       # SQL linter config
```

## Service Overview

The following table maps services defined in docker-compose.yaml to their corresponding Dockerfile / configuration.

### Development Environment (profile: `development`)

|Service|Dockerfile / Image|Port|Description|
|---|---|---|---|
|`api_server`|`docker/server/Dockerfile` (target: `tooling`)|8080, 2345, 6060|Development API server (hot reload via air)|
|`database`|`postgres:18.3-bookworm`|5432|PostgreSQL database|

### Auxiliary Services (profile: `tools`)

|Service|Dockerfile / Image|Port|Description|
|---|---|---|---|
|`docs_viewer`|`docker/document/Dockerfile`|8082|Documentation portal (nginx)|
|`sql_editor`|`sosedoff/pgweb`|8081|Web SQL editor|

### Tool Runners (profile: `generate`)

|Service|Dockerfile / Image|Description|
|---|---|---|
|`go_tool_runner`|`docker/tools/Dockerfile` (target: `go_tools`)|oapi-codegen, mockgen, sqlc, migrate|
|`node_tool_runner`|`docker/tools/Dockerfile` (target: `node_tools`)|redocly-cli|
|`python_tool_runner`|`docker/tools/Dockerfile` (target: `python_tools`)|sqlfluff|
|`er_diagram_generator`|`schemaspy/schemaspy`|ER diagram generation (SchemaSpy)|

## server

The Dockerfile for the application server. Provides the following targets via multi-stage build.

|Target|Purpose|Base Image|
|---|---|---|
|`builder`|Go binary build|`golang:1.26.3-alpine`|
|`runtime`|Production runtime container|`alpine:3.23`|
|`migration`|Migration execution container|Inherits `runtime`|
|`tooling`|Local development environment|`golang:1.26.3-alpine`|

### runtime

- Runs as non-root user (`app`)
- Embeds version / revision / build date via `ldflags`
- Builds in `vendor` mode (`GOPROXY=off`)

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
|`go_tools`|`golang:1.26.3-alpine`|oapi-codegen, mockgen, sqlc, migrate|
|`node_tools`|`node:24.14-alpine`|redocly-cli, js-yaml|
|`python_tools`|`python:3.14.2-slim`|sqlfluff|

## document

Container for the documentation portal.

- Base image: `nginx:1.29-otel`
- Volume mounts the entire `docs/` directory
- Portal app is served at `/portal/`
- Accessing `http://localhost:8082/` redirects to `/portal/`

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
