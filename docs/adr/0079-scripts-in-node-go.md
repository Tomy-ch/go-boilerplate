---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [toolchain, scripts]
---

# ADR-0079: Operational scripts live in scripts/ as TypeScript or Go; shell scripting is not used

## Status

accepted

## Context

The project needs a variety of utility programs beyond the application itself:
documentation-portal generation, Makefile help output, semantic versioning, runtime
version synchronisation, Mermaid diagram linting, GitHub Actions pinning, and
initial-project setup. These operations are too complex for one-liner Makefile recipes
but do not belong in the application code.

Shell scripting is fragile under edge cases (word splitting, filename glob expansion,
missing tools, portability differences across sh/bash/zsh), difficult to test, and hard
to maintain as complexity grows. The project already depends on Node.js and Go for the
application stack.

## Decision

Operational scripts live in `scripts/` and are written in TypeScript (run through `tsx`) or
Go, depending on the nature of the task.

**TypeScript** is used for documentation generation, Makefile help output, versioning, and
repository gates — work that involves text manipulation, file I/O, and Markdown or YAML
processing:

- `portal/gen-portal-docs.ts`, `portal/gen-docs-json.ts` — portal generation.
- `make-help.ts` — parse `.makefiles/*.mk` and render the help output.
- `semver.ts` — semantic-version bumping.
- `stamp-openapi-version.ts` — derive version from branch name and write it to `openapi.yaml`.
- `mermaid-lint.ts`, `skill-lint.ts`, `pr-comment-secret-lint.ts`, `pr-comment-fence-lint.ts`,
  `actions-cutoff-lint.ts` — repository gates over Markdown and workflow YAML.
- `setup/replace-*.ts` — one-time rewrites of the Go module path, app metadata, repository
  references, the LICENSE holder and the CODEOWNERS owner field.
- `setup/remove-sample-api.ts`, `setup/verify-sample-removal.ts` — sample-API removal and the
  check that it was exact.

Each script keeps its decision logic in a pure module under `scripts/lib/` (or
`scripts/portal/`) with a `vitest` suite next to it, leaving the entry file to do file I/O
and exit codes. Several of these scripts are gates whose failure mode is to inspect nothing
and still exit `0`, which a type checker and a test can pin and an untyped script cannot.

The one-time initial-setup scripts under `setup/` follow the same split, and are the reason the
split is not merely a convention. Five of them are never executed by CI at all, and every one of
them rewrites the repository in place — a Go module path across every file in the tree, the
LICENSE holder, the owner field of every CODEOWNERS rule. When a replacement rule over-matches or
misses a file type, the failure surfaces in a tree that has already been rewritten, in front of
someone holding no context to debug it with. So the replacement rules live in pure modules under
`setup/lib/` with a `vitest` suite next to each, and what those tests pin is the rule itself: what
it matches, what it must not match, and which file types it must not miss.

Two of them delete themselves. `remove-sample-api.ts` removes its own manifest and marker logic
along with the sample, and `verify-sample-removal.ts` removes itself, its decision module and that
module's test once the verification passes. A decision module extracted out of a self-deleting
script has to be deleted with it, or the tool survives in fragments in the user's repository.

**Go** is used where stronger type safety, error handling, or richer standard-library
support is needed, and for anything a gate invokes on the host without a Node toolchain:

- `sync-versions/` — parse `mise.toml` and propagate language-runtime versions to
  `go.mod` and Dockerfile `FROM` lines; atomic writes, pre-validation.
- `genctxkey/` — Echo context key code generator; driven by `//go:generate` directives.
- `pin-actions/` / `pin-images/` — resolve and apply commit-SHA and digest pins; network
  I/O, lockfile management, supply-chain quarantine.
- `go-cooldown/` / `tool-cooldown/` — supply-chain cooldown gates.
- `migration-lint/` / `cover-gate/` / `load-band/` — gates and resolvers called from
  `.makefiles/**`, where the alternative was leaving the decision in a shell recipe.

Shell scripting is not used for any of these tasks.

**Role separation from the rest of the codebase:**

- `scripts/` — utility programs for build, CI, and setup; not consumed as libraries.
- `pkg/` — reusable library packages consumed by `internal/` and potentially by
  external callers; must stay framework-agnostic and free of `internal/` imports.
- `internal/` — application business logic (domain, usecase, controller, infrastructure).

Scripts invoke application code only at arm's length (e.g. through generated artefacts
or by reading configuration files), never by importing `internal/` packages directly.

## Consequences

### Positive Consequences

- Both TypeScript and Go provide proper error handling, testability, and cross-platform
  behaviour absent from shell.
- A type checker plus a test suite covers the failure mode that matters for a gate — a
  script that stops inspecting anything while still exiting `0`. Shell cannot express
  either check, and untyped JavaScript only the second.
- Go scripts share the project's primary toolchain and run via `go run` without
  additional installation, so a gate in a `make` recipe needs no Node on the host.
- The role separation keeps `pkg/` and `internal/` clean of build-tooling concerns.

### Negative Consequences

- The `scripts/` directory houses two runtimes (Node and Go); contributors must identify
  which runtime applies before editing a script.
- No TypeScript script is dependency-free: every one of them needs `tsx` to run, so
  `scripts/` carries a `package.json` + `pnpm-lock.yaml` and the scripts cannot run on a
  bare host until `pnpm install --dir scripts` has been done. That cost is why a gate
  reachable from a `make` recipe is written in Go instead.
- Setup scripts are one-time use; they remain in the repository after initial setup is
  complete, which adds volume without ongoing value.
- The self-deleting setup tools take their tests with them, so the guarantee those tests carry
  ends at the moment of use. That is the correct trade — the alternative is shipping a test suite
  for a tool that no longer exists.

## Alternatives Considered

### Shell scripts

Ubiquitous and zero-dependency, but fragile under edge cases and hard to test. The
project's scripts involve enough conditional logic and file manipulation that shell
becomes a maintenance liability. Shell is also poorly suited for tasks such as numeric
computation, conditional branching over multiple states, or format-specific tooling such
as Mermaid diagram linting — all of which appear in this project's script inventory.

### Plain JavaScript (ESM `.mjs`)

Runs with no build step and no dependency. That was the argument for leaving `setup/` on `.mjs`,
and it did not survive the observation that those scripts are the least observed code in the
repository: a mistyped field or a renamed key surfaces as a silently empty result set rather than
an error, and for a setup script nobody is watching the result set. Once a test harness is
warranted anyway, the marginal cost of adding types over it is small, and `scripts/` now holds a
single JavaScript dialect rather than two.

### Python

The project has a Python tool runner (`python_tool_runner`) but Python is used
exclusively for SQL linting (sqlfluff). Introducing Python for general scripting would
add a third scripting runtime without clear benefit over Node for text-manipulation
tasks.

### Deno

Shares the ESM model with Node but is not present in the tool stack; introducing it
adds a dependency.

## Notes

- Scripts directory overview and per-script descriptions:
  [`scripts/README.md`](../../scripts/README.md).
- Toolchain container definitions (node_tool_runner, go_tool_runner):
  [ADR-0076](0076-containerized-pinned-toolchain.md).
- Make target entrypoint for invoking scripts:
  [ADR-0078](0078-make-single-entrypoint.md).
- **Upstream deviations**: while this repository is distributed as a boilerplate, the setup scripts carry an argument for the split that does not transfer, recorded in [`docs/get-started/boilerplate-only-conventions.md`](../get-started/boilerplate-only-conventions.md). <!-- boilerplate-only:line -->
