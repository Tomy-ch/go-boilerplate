---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [toolchain, dev-env]
---

# ADR-0071: Local dev environment is provided via Docker Compose with profile-separated services

## Status

accepted

## Context

Local development requires several independent services: a running API server with hot
reload and debugger access, a PostgreSQL database, an observability stack, documentation
viewing, a SQL editor, and access to containerised code generation tools. These services
have different lifecycles and are not all needed at the same time — a contributor
working on a feature does not need the documentation viewer or the ER diagram generator
running.

Tool execution must also be reproducible across machines (see
[ADR-0067](0067-containerized-pinned-toolchain.md)), which means tools run inside
containers rather than directly on the host. A single `docker-compose.yaml` that covers
both application services and tool runners gives one consistent mechanism for all
container management.

## Decision

The local development environment is defined in `docker-compose.yaml` using Docker
Compose **profiles** to separate concerns. Contributors start only the services they
need.

**Profile: `development`** — standard feature development.

| Service | Image / Dockerfile | Ports | Description |
| --- | --- | --- | --- |
| `api_server` | `docker/server/Dockerfile` target `tooling` | 8080, 2345, 6060 | Hot reload (air), debugger (dlv), pprof metrics |
| `database` | `postgres:18.3-bookworm` | 5432 | PostgreSQL; health-checked before `api_server` starts |
| `observability` | `grafana/otel-lgtm` | 3000, 4317, 4318, 3200 | Grafana, OTLP gRPC/HTTP, Tempo API |

**Profile: `tools`** — auxiliary developer tools; shares the `database` service.

| Service | Image / Dockerfile | Ports | Description |
| --- | --- | --- | --- |
| `docs_viewer` | `docker/document/Dockerfile` target `document_viewer` | 8082 | nginx serving `docs/`; portal at `/portal/` |
| `sql_editor` | `sosedoff/pgweb` | 8081 | Web SQL editor |

(A lightweight **`database`** profile also exists — `database` + `sql_editor` only — for working
against the DB without the `api_server` / observability stack; several services carry more than
one profile tag.)

**Profile: `generate`** — on-demand tool runners for codegen and documentation; started
per `make` target invocation, not kept running.

| Service | Dockerfile target | Description |
| --- | --- | --- |
| `go_tool_runner` | `go_tools` | oapi-codegen, mockgen, sqlc, migrate, trivy, hadolint, and others |
| `node_tool_runner` | `node_tools` | redocly-cli, markdownlint-cli2, commitlint (+ script deps such as js-yaml) |
| `python_tool_runner` | `python_tools` | sqlfluff |
| `er_diagram_generator` | `schemaspy/schemaspy` image | ER diagram generation |

Tool runners run as root because mise is installed under `/root`; bind-mounted outputs
are consequently root-owned, which is accepted behaviour consistent with generated mock
files elsewhere in the project.

The `tooling` target of `docker/server/Dockerfile` includes developer tools (air, dlv,
golines, gofumpt, golangci-lint) on top of the Go runtime, providing the full
development environment in a single image for the `api_server` service.

## Consequences

### Positive Consequences

- Profile separation means contributors start only the services required for their
  current task, reducing resource usage.
- Hot reload (air) and remote debugging (dlv) are available without any host tool
  configuration.
- Tool runner containers guarantee reproducible codegen output identical to CI.
- A single `docker-compose.yaml` is the canonical reference for all local service
  configuration.

### Negative Consequences

- Docker must be running for all standard operations, including tool invocations.
- The observability stack (`grafana/otel-lgtm`) is bundled into the `development`
  profile, adding memory and CPU overhead even when observability is not the focus.
- Tool runner bind-mounted outputs are owned by root, requiring `sudo` or group
  configuration to delete them on some systems.
- The `tooling` Dockerfile target bundles development tools into the server image,
  making it larger than a pure runtime image.

## Alternatives Considered

### Process-based local development (tools on host)

Running the API server directly on the host (e.g. `air` installed via `mise`) is
faster to start but breaks the reproducibility guarantee: tool and runtime versions
depend on what the contributor has installed. It also requires managing the database and
observability stack separately.

### Separate docker-compose files per concern

More explicit separation, but sharing the `database` service across profiles (e.g.
`tools` and `development`) becomes awkward without a common file. A single file with
profiles achieves the same separation more simply.

### devcontainer (VS Code Dev Containers)

Editor-specific; does not cover CI or non-VS Code workflows. Docker Compose is
editor-agnostic and usable from any terminal.

## Notes

- Docker Compose service and profile definitions:
  [`docker-compose.yaml`](../../docker-compose.yaml).
- Dockerfile targets and service details:
  [`docker/README.md`](../../docker/README.md).
- Container-based reproducibility rationale:
  [ADR-0067](0067-containerized-pinned-toolchain.md).
- `make serve` (development profile), `make tools` (tools profile):
  [`.makefiles/README.md`](../../.makefiles/README.md).
