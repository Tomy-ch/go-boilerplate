---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [toolchain, build]
---

# ADR-0068: Make is the single tool entrypoint with .mk registration and self-documenting help

## Status

accepted

## Context

The project spans many domains: database migrations, code generation, linting, testing,
documentation, GitHub repository configuration, and more. Without a unified entrypoint,
contributors must memorise per-domain commands, and automation (CI, git hooks, skills)
must hard-code tool invocations that can drift from the canonical way to run a task.

Adding a new target should not require editing the top-level file. Targets should be
self-documenting so that `make help` is always accurate without a separately maintained
reference.

## Decision

The top-level `makefile` is the single entrypoint for all developer and automation
operations. Its structure embodies two conventions.

**`.mk` registration:** `.makefiles/` is the central registry. Each thematic `.mk`
file groups related targets (e.g. `.makefiles/go/lint.mk`, `.makefiles/database/migrate.mk`).
The top-level `makefile` includes every registered `.mk` file. Adding a new target
means placing it in the appropriate `.mk` file; no top-level edit is required.

**Self-documenting help contract:** `make help` is the default goal
(`.DEFAULT_GOAL := help`). It is implemented by running `node scripts/make_help.mjs`,
which walks every `.mk` file recursively and prints each `.PHONY` target whose line
carries a trailing `## description` comment, grouped under the `## Category` headings
defined in the `.mk` file. Targets missing the `## description` comment do not appear
in the help output; `make_help.mjs` emits a warning to stderr for each such target.

Target-naming convention: dash-separated lower case (e.g. `make new-migrate-name`,
`make gen-api`).

Two execution flavours:

- **Normal targets** — invoke tools inside Docker containers for reproducibility; this
  is the standard path for developers and automation.
- **`-ci` targets** — invoke tools on bare metal; intended for CI runners or inside
  containers themselves.

## Consequences

### Positive Consequences

- `make help` provides a self-updating, always-accurate reference for every registered
  target; documentation never drifts from the actual command surface.
- New `.mk` files are picked up automatically by the top-level include and by
  `make_help.mjs`; no registration step beyond placing the file in `.makefiles/`.
- CI, git hooks, and skills share one stable entrypoint.
- The normal / `-ci` flavour split keeps the same logical operation available in both
  containerised and bare-metal contexts.

### Negative Consequences

- Contributors must follow the `.PHONY` + `## comment` convention; omitting the comment
  causes the target to disappear from `make help` silently (only a stderr warning from
  `make_help.mjs` signals the problem).
- GNU Make must be available on contributor machines.
- The `.mk` file structure requires contributors to know which group file to edit when
  adding a target.

## Alternatives Considered

### Task (Taskfile)

YAML-based and more readable for simple tasks, but less universally available than
Make. The self-documenting convention in Make achieves the same discoverability goal.

### Just

Similar advantages to Task; introduces an additional tool dependency that is not
present in most CI images by default.

### Bare shell scripts

No structured discovery mechanism; contributors must read the filesystem. CI must
hard-code invocations that can diverge from the developer-facing commands.

## Notes

- Top-level makefile and include list:
  [`makefile`](../../../makefile).
- `.makefiles/` conventions (normal vs `-ci`, naming, group layout):
  [`.makefiles/README.md`](../../../.makefiles/README.md).
- Help generator source:
  [`scripts/make_help.mjs`](../../../scripts/make_help.mjs).
- Toolchain execution rules (container vs bare-metal):
  [`docs/rules.md`](../../rules.md) § "Toolchain Execution Rules".
