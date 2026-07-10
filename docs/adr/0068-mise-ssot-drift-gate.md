---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [toolchain, ci]
---

# ADR-0068: mise.toml is the single source of truth; versions propagate downstream with a CI drift gate

## Status

accepted

## Context

Multiple files must agree on the same language-runtime versions:

- `mise.toml` — declares the versions for developer provisioning and Docker image builds.
- `go.mod` — the `go` directive must match the Go version in `mise.toml`.
- `docker/server/Dockerfile` and `docker/tools/Dockerfile` — `FROM golang:X`, `FROM node:X`,
  and `FROM python:X` tags must match `mise.toml`.

Keeping these files in sync by hand is error-prone: a contributor upgrading Go in
`mise.toml` may forget to update `go.mod` or the Dockerfile `FROM` lines. Without a
verification gate, the divergence silently lands on the main branch.

Some Docker image versions (for example `grafana/otel-lgtm`) cannot be managed through
the `[tools]` table because they are not installable via the mise registry; they still
need a single authoritative location.

## Decision

`mise.toml` is the single source of truth for all tool and runtime versions used by the
project.

**Version propagation for language runtimes:** the `go`, `node`, and `python` values
declared under `[tools]` propagate to `go.mod` (the `go` directive) and to the
relevant `FROM` lines in `docker/server/Dockerfile` and `docker/tools/Dockerfile` via
`make sync-versions`, which runs `scripts/sync-versions/main.go`. The Go program
validates preconditions (version present, files exist, expected match counts) and
writes each file atomically, so a failure never leaves partial state.

**Docker image versions** that are outside mise's tool resolution model are declared in
the `[env]` section of `mise.toml` (for example `OTEL_LGTM_VERSION`), keeping them in
the same single file without polluting the `[tools]` table.

**CI drift gate:** the `sync-versions-check` workflow triggers on pull requests that
touch `mise.toml`, `go.mod`, the Dockerfiles, or the sync script itself. It runs
`go run ./scripts/sync-versions` against the branch and then checks `git diff --quiet`.
Any resulting diff causes the workflow to fail and instructs the author to run
`make sync-versions` locally and commit the result.

## Consequences

### Positive Consequences

- One change in `mise.toml` is the only edit required to upgrade a language runtime
  everywhere; the sync script handles the rest.
- Drift is caught at PR review time before merge, not discovered later in a broken
  build.
- The Go-based sync script provides strong error handling and leaves no partial state
  on failure.
- Docker image versions have a single canonical location even though they are not
  `[tools]` entries.
- The CI drift gate is a deliberate guardrail: wrong versions cannot silently enter the
  development branch. Running `make sync-versions` and committing the result is expected,
  intentional friction — not having this gate would be the riskier choice.

### Negative Consequences

- The `[env]` section for Docker image versions is a slightly different pattern from
  the `[tools]` table; contributors must know which section applies to which kind of
  version.

## Alternatives Considered

### Renovate / Dependabot per-file PRs

Automated dependency PRs keep individual files up to date but treat each file
independently. Cross-file alignment (e.g. `go.mod` and `Dockerfile` both tracking the
same Go version declared in `mise.toml`) still requires coordination. The single-source
approach makes the relationship explicit and machine-verifiable.

### Shell-based sync script

A shell script is simpler to write but fragile under edge cases (word splitting,
missing tools, portability). The Go program can be run via `go run` without extra
dependencies and provides proper error handling and atomic writes.

### No sync; maintain each file independently

Acceptable for very small teams but does not scale; drift incidents are a recurring
cost in multi-contributor projects.

## Notes

- Related decision on container-based execution:
  [ADR-0067](0067-containerized-pinned-toolchain.md).
- `mise.toml` with SSOT comment and `[env]` section:
  [`mise.toml`](../../mise.toml).
- CI drift-gate workflow:
  [`.github/workflows/sync-versions-check.yaml`](../../.github/workflows/sync-versions-check.yaml).
- Sync script: [`scripts/sync-versions/`](../../scripts/sync-versions/).
