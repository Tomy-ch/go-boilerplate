---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [toolchain, ci]
---

# ADR-0075: mise.toml is the single source of truth for mise-resolved versions; versions propagate downstream with a CI drift gate

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

Tools published to PyPI are a second case where a `[tools]` entry is not enough. mise
pins a version, but nothing below it: the transitive dependencies of a Python tool are
resolved at install time and are not pinned or hash-checked at all, so the same
`[tools]` entry installs a different dependency tree on different days. That is the
part of the supply chain where a compromise is most likely to arrive unnoticed, and it
is also the part no vulnerability scanner can see, because `mise.toml` is not a lockfile
that `osv-scanner` knows how to read.

## Decision

`mise.toml` is the single source of truth for every tool and runtime version that mise
resolves.

**PyPI tools are declared and locked outside mise.** The version is declared in
`python/<tool>.in`, and `python/<tool>.txt` — produced by `make py-lock`
(`uv pip compile --generate-hashes --universal`) — pins the whole transitive tree with
sha256 hashes. Installation goes through `uv pip install --require-hashes`, which
refuses any requirement lacking a version or a hash, so hash verification is not a
separate check that can be skipped. One `.in`/`.txt` pair per tool keeps each tool's
resolution independent. `mise.toml` still declares `uv` itself, and the Python runtime
the lockfiles are resolved against.

**Version propagation for language runtimes:** the `go`, `node`, and `python` values
declared under `[tools]` propagate to `go.mod` (the `go` directive) and to the
relevant `FROM` lines in `docker/server/Dockerfile` and `docker/tools/Dockerfile` via
`make sync-versions`, which runs `scripts/sync-versions/main.go`. The Go program
validates preconditions (version present, files exist, expected match counts) and
writes each file atomically, so a failure never leaves partial state.

**Docker image versions** that are outside mise's tool resolution model are declared in
the `[env]` section of `mise.toml` (for example `OTEL_LGTM_VERSION`), keeping them in
the same single file without polluting the `[tools]` table.

**Declaration / lockfile drift gate:** `scripts/mise-cooldown` reads `python/*.in`
alongside `mise.toml` and fails when a declared version is not the one its lockfile
pins. Without that check, raising a `.in` without regenerating its `.txt` would leave
the cooldown gate clearing a version that is never installed.

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
- Python tooling installs the same dependency tree everywhere, verified by hash, and
  `osv-scanner` reads `python/*.txt` as a lockfile it understands — coverage that a
  `[tools]` entry never gave.
- The CI drift gate is a deliberate guardrail: wrong versions cannot silently enter the
  development branch. Running `make sync-versions` and committing the result is expected,
  intentional friction — not having this gate would be the riskier choice.

### Negative Consequences

- The `[env]` section for Docker image versions is a slightly different pattern from
  the `[tools]` table; contributors must know which section applies to which kind of
  version.
- A PyPI tool now takes two files instead of one line, and upgrading it means editing
  `python/<tool>.in` and regenerating the lockfile rather than editing `mise.toml`.
  The drift gate above is what keeps that second step from being forgotten silently.

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

### Keep PyPI tools in `[tools]` and accept unpinned transitive dependencies

The one-line-per-tool form is simpler, and it is what this repository used until the
lockfiles were introduced. It was rejected because the simplicity is bought entirely
from the part of the dependency tree nobody reviews: the tool's own dependencies float,
so two builds a week apart install different code, and no scanner reads the declaration.

### `pyproject.toml` + `uv.lock`

uv's native project lockfile also records a hash per artifact and supports targeted
upgrades. It was rejected because it models one project with one resolution, while these
are unrelated CLI tools that happen to share an ecosystem: a dependency conflict between
them would have to be declared and worked around rather than simply not existing. It
would also put a Python project manifest at the root of a Go repository.

## Notes

- Related decision on container-based execution:
  [ADR-0074](0074-containerized-pinned-toolchain.md).
- `mise.toml` with SSOT comment and `[env]` section:
  [`mise.toml`](../../mise.toml).
- PyPI tool declarations and lockfiles: [`python/`](../../python/).
- Lockfile regeneration target: [`.makefiles/python/lock.mk`](../../.makefiles/python/lock.mk).
- CI drift-gate workflow:
  [`.github/workflows/sync-versions-check.yaml`](../../.github/workflows/sync-versions-check.yaml).
- Sync script: [`scripts/sync-versions/`](../../scripts/sync-versions/).
- Declaration / lockfile drift gate: [`scripts/mise-cooldown/`](../../scripts/mise-cooldown/).
