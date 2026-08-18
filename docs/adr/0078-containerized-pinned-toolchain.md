---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [toolchain, build]
---

# ADR-0078: Use a containerized toolchain pinned by mise for reproducibility

## Status

accepted

## Context

This project is intended for teams with multiple contributors working on different
machines and operating systems. Go, Node, Python, and a wide set of secondary programs
(linters, codegen tools, migration runner, debugger, etc.) each have exact version
requirements: an unversioned host tool can silently produce different output or behave
differently across environments.

Managing tool versions with host-level package managers diverges over time and makes
"works on my machine" failures hard to diagnose. CI and developer laptops must agree
on the same tool binaries. Reproducing a build or a lint failure should not depend on
what a developer happens to have installed.

## Decision

Tool versions are declared centrally — in `mise.toml` for everything mise resolves, and in
`python/*.in` + `python/*.txt` for PyPI tools ([ADR-0079](0079-mise-ssot-drift-gate.md)) —
and baked into Docker images. All
tool execution — lint, format, codegen, documentation generation, commit-message lint,
and so on — runs through `make` targets that invoke the tools inside the appropriate
Docker container (`go_tool_runner`, `node_tool_runner`, or `python_tool_runner`).

The three execution contexts are:

- **Normal `make` targets** — invoke tools inside Docker containers; this is the
  standard developer path and what automation must use.
- **`-ci` targets** — invoke tools on bare metal, intended for CI runners or inside
  containers themselves.
- **Host `mise`** — used only for provisioning versions (`make install-tools`, Quick
  Start onboarding); `mise exec` is not the tool execution path.

Automation (git hooks via lefthook, CI steps, and skills) MUST invoke tools through
`make` targets, never by running a tool binary directly on the host. Bypassing the
container depends on host-local tool state and breaks reproducibility.

## Consequences

### Positive Consequences

- Tool output is identical regardless of the contributor's host machine or operating
  system.
- New contributors do not need to manage tool versions manually beyond installing
  Docker and mise.
- The same `make` target works in CI and locally without modification.
- Upgrading a tool version is a single change in its declaration (and the Dockerfile that
  consumes it); for a PyPI tool, `make py-lock` then regenerates the lockfile.

### Negative Consequences

- Docker must be running for any standard tool invocation.
- Container startup adds latency compared with running a host binary directly.
- Bind-mounted outputs from the `generate` profile containers are owned by root; this
  is accepted behaviour, consistent with generated mock files.

## Alternatives Considered

### Direct host tool execution

Fast and familiar, but version drift between machines is inevitable and silent.
Debugging "lint passes locally, fails in CI" becomes a recurring cost.

### Nix flakes

Provides hermetic environments comparable to the container approach, but carries a
steep learning curve and is uncommon in Go teams. Docker Compose is already required
for the database and observability services, so adding containers for tool runners
imposes no new infrastructure dependency.

### devcontainer (VS Code Dev Containers)

Editor-specific; does not cover CI or non-VS Code workflows. The Docker Compose
approach is editor-agnostic.

## Notes

- Toolchain execution rules (including the "never bypass the container" constraint):
  [`docs/rules.md`](../rules.md) § "Toolchain Execution Rules".
- Tool versions declared in: [`mise.toml`](../../mise.toml), and [`python/`](../../python/)
  for PyPI tools.
- Container service definitions: [`docker-compose.yaml`](../../docker-compose.yaml)
  (`go_tool_runner`, `node_tool_runner`, `python_tool_runner` — profile `generate`).
- Docker image / Dockerfile details: [`docker/README.md`](../../docker/README.md).
