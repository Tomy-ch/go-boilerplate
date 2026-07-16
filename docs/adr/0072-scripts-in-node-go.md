---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [toolchain, scripts]
---

# ADR-0072: Operational scripts live in scripts/ as Node (.mjs) or Go; shell scripting is not used

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

Operational scripts live in `scripts/` and are written in Node (ESM `.mjs` modules) or
Go, depending on the nature of the task.

**Node (`.mjs`)** is used for documentation generation, Makefile help output,
versioning, and initial-setup tasks — work that involves text manipulation, file I/O,
and Markdown or YAML processing:

- `gen-portal-docs.mjs`, `gen-docs-json.mjs`, `build-portal.mjs` — portal generation.
- `make_help.mjs` — parse `.makefiles/*.mk` and render the help output.
- `semver.mjs` — semantic-version bumping.
- `stamp-openapi-version.mjs` — derive version from branch name and write it to
  `openapi.yaml`; dependency-free ESM.
- `mermaid-lint.mjs` — validate Mermaid fences in Markdown.
- `setup/*.mjs` — one-time initial-project setup scripts (module rename, metadata
  replacement, sample-API removal).

**Go** is used where stronger type safety, error handling, or richer standard-library
support is needed:

- `sync-versions/` — parse `mise.toml` and propagate language-runtime versions to
  `go.mod` and Dockerfile `FROM` lines; atomic writes, pre-validation.
- `genctxkey/` — Echo context key code generator; driven by `//go:generate` directives.
- `pin-actions/` — resolve and apply commit-SHA pins for GitHub Actions `uses:`
  references; network I/O, lockfile management, supply-chain quarantine.

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

- Both Node and Go provide proper error handling, testability, and cross-platform
  behaviour absent from shell.
- Dependency-free ESM scripts (e.g. `stamp-openapi-version.mjs`) run on any Node
  install without an `npm install` step.
- Go scripts share the project's primary toolchain and run via `go run` without
  additional installation.
- The role separation keeps `pkg/` and `internal/` clean of build-tooling concerns.

### Negative Consequences

- The `scripts/` directory houses two runtimes (Node and Go); contributors must identify
  which runtime applies before editing a script.
- Setup scripts are one-time use; they remain in the repository after initial setup is
  complete, which adds volume without ongoing value.
- Documentation scripts require the `node_modules` available in `docker/tools/`; they
  cannot be run on a bare host without `npm install`.

## Alternatives Considered

### Shell scripts

Ubiquitous and zero-dependency, but fragile under edge cases and hard to test. The
project's scripts involve enough conditional logic and file manipulation that shell
becomes a maintenance liability. Shell is also poorly suited for tasks such as numeric
computation, conditional branching over multiple states, or format-specific tooling such
as Mermaid diagram linting — all of which appear in this project's script inventory.

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
  [ADR-0069](0069-containerized-pinned-toolchain.md).
- Make target entrypoint for invoking scripts:
  [ADR-0071](0071-make-single-entrypoint.md).
