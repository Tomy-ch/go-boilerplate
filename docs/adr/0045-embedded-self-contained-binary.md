---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [config, build]
---

# ADR-0045: go:embed bundles config (.env) and migrations for a self-contained binary

## Status

accepted

## Context

The application needs two sets of files available at runtime:

1. **Configuration defaults** (`env/.env`) — environment variables that seed the
   process before runtime overrides are applied.
2. **Database migrations** (`database/migrations/`) — SQL files run by the migration
   tool at startup or deployment.

Distributing these as external files alongside the binary requires operators to manage
file placement and introduces deployment surface. A binary that depends on external
files present at a specific path is harder to containerize consistently and easier to
misconfigure.

## Decision

The repo root `embed.go` uses `//go:embed` to bundle both asset groups into the binary
at build time:

```go
//go:embed env/.env database/migrations
var FS embed.FS
```

`FS` is an `embed.FS` exported from the `root` package. The config loader reads
`env/.env` from `FS` via godotenv and sets each variable **only if it is not already
set** in the process environment, so runtime-injected environment variables always take
precedence over the embedded defaults.

**Per-environment builds:** `env/.env` is the single embed target. The Docker `builder`
stage materializes the target environment's file before `go build` via:

```text
cp env/.env.${APP_ENV} env/.env
```

Non-Docker flows (`go run`, `go test`) embed the committed local `env/.env`. CI that
needs another environment re-bakes it the same way.

## Consequences

### Positive Consequences

- A single binary carries its configuration defaults and migrations; no external file
  placement is required at runtime.
- The embedded `env/.env` acts as the local default; any value can be overridden by
  setting the corresponding environment variable at runtime.
- Migration files are always in sync with the binary that was built (version drift
  between binary and migration files is impossible).

### Negative Consequences

- Changing configuration defaults requires a rebuild and redeployment, not just a file
  update.
- Only one `.env` file can be embedded at a time; the per-environment swap must happen
  before `go build` (a Docker-layer concern, not a Go concern).
- `FS` is package-level state; tests that need to read from it must be aware of the
  embedded content.

## Alternatives Considered

### External config files only

Require operators to place `.env` and migration directories alongside the binary.
Rejected: increases deployment surface and makes container images dependent on
out-of-band file distribution.

### Config server / remote key-value store

Fetch configuration from a remote source at startup. Rejected for defaults: adds a
network dependency before the application is ready and does not address the migration
bundling need.

## Notes

- Source: `embed.go` (repo root, `//go:embed env/.env database/migrations`),
  `internal/config/README.md` (Load step — embedded file read via `root.FS`),
  `env/README.md` (Notes section — per-environment build flow).
- The config loading sequence that consumes `FS` is described in
  [ADR-0044](0044-immutable-fail-fast-config.md).
