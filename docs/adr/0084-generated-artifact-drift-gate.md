---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [ci, codegen]
---

# ADR-0084: Generated-artifact drift gate + release-branch-centralized auto-generation bot

## Status

accepted

## Context

The project generates several categories of artifacts from source:

- **Go code**: OpenAPI server stubs, mocks, and embedded validation spec
  (`make gen-go-code`).
- **Database artifacts**: sqlc type-safe query code, schema dump, merged DML
  (`make gen-query` and related targets).
- **Documentation**: OpenAPI HTML docs, ER diagram, test coverage HTML, godoc, portal
  docs bundle.

When a developer modifies a source (`.go`, `.sql`, OpenAPI definition) and forgets to
regenerate, the committed generated files lag behind the source. Tests may still pass
because they test the stale generated code, while runtime or API consumers see the
un-regenerated version. Detecting this drift requires re-running the generators in CI
and checking for uncommitted differences.

At the same time, running doc regeneration on every PR would require a full database
setup, portal build, and parallel generation pipeline — too heavy for routine PR
verification. Documentation generation also has a natural cadence tied to releases.

## Decision

Split the generated-artifact CI story into two complementary mechanisms:

### 1. PR-time drift gate (two workflows)

**`gen-go-artifacts-check.yaml`** (lines 36–111): On every PR that touches `.go`,
`go.mod`, `go.sum`, or makefile files, run `make gen-go-code`, then `git add -A` and
inspect the diff. If generated files changed:

- Determine whether the PR's source files (`.go`, excluding `*.gen.go`, `*.sql.go`,
  `*_mock.go`) were themselves modified. If yes, the developer forgot to commit the
  regenerated output. If no, the drift is in the generator itself (e.g. mockgen or
  oapi-codegen version difference on base).
- Post an upsert PR comment with the appropriate diagnosis and the list of drifted files.
- Exit non-zero unconditionally if any diff exists.

**`gen-db-artifacts-check.yaml`**: Same pattern for database artifacts. Runs the full
DB generation pipeline (migrate, dump schema, merge DML, regenerate sqlc, format),
then checks for drift. Distinguishes between SQL source modified in the PR (developer
forgot to commit) versus generator-side drift (sqlc version difference).

Both workflows post a persistent upsert comment on the PR so the diagnosis is visible
without reading workflow logs.

### 2. Release-branch auto-generation bot (`auto-generate-docs.yaml`)

Triggered on every push to `release/**` branches (not PRs). Runs the full generation
pipeline — OpenAPI bundle, docs, ER diagram, test coverage report, godoc, portal build
— and if the result differs from HEAD, opens (or updates) a PR targeting the same
release branch via `peter-evans/create-pull-request`. The PR is tagged `[skip ci]` on
its commit message and is safe to auto-merge once CI passes.

This centralises documentation generation to the release branch rather than requiring
every developer to run the full pipeline locally, while ensuring generated docs are
always in sync with the released source.

## Consequences

### Positive Consequences

- Generated code drift is caught at PR time with a clear, actionable diagnosis (forgot
  to regenerate vs. generator version mismatch).
- Documentation generation burden is removed from individual developers; the bot handles
  it automatically on release branches.
- The PR comment pinpoints exactly which files drifted, reducing investigation time.

### Negative Consequences

- The DB drift gate requires a real Postgres container and a full sqlc pipeline, making
  it the heaviest of the PR checks.
- The auto-generation bot requires `contents: write` and `pull-requests: write`
  permissions on the release branch, which is a broader scope than read-only CI checks.
- If the generator itself is broken (e.g. a new sqlc version produces invalid code), the
  drift gate will fail but the error message may not be obvious.

## Alternatives Considered

### Commit generated artifacts manually as part of every PR

Currently attempted but error-prone: developers forget, and there is no systematic
feedback until another developer or reviewer notices the stale file.

### Run full doc generation on every PR

Too slow. The portal build, godoc generation, and test coverage pipeline require several
minutes and a full database. The cost is not justified for every PR when the docs are
only consumed at release time.

### Store generated artifacts outside the repository (e.g. release assets)

Possible but breaks IDE tooling and `go build` flows that depend on generated `.go`
files being present in the tree.

## Notes

- Sources: `.github/workflows/gen-go-artifacts-check.yaml` lines 36–111,
  `.github/workflows/gen-db-artifacts-check.yaml`,
  `.github/workflows/auto-generate-docs.yaml` lines 90–181.
- Related: [ADR-0009](0009-openapi-first.md) — the OpenAPI-first contract that drives Go
  code generation.
- Related: [ADR-0022](0022-sqlc-type-safe-sql.md) — sqlc, the source of DB artifact
  generation.
