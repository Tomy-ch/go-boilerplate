---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [ci, lint]
---

# ADR-0075: Two-layer golangci config: minimal default vs full authoritative gate

## Status

accepted

## Context

golangci-lint picks up `.golangci.yaml` automatically when no `--config` flag is given.
IDE plugins (VS Code, GoLand) and editors that integrate golangci-lint directly use this
implicit default. The full suite of linters — many of which are slow, noisy, or irrelevant
for in-flight editing — would degrade the editor feedback loop if run on every save.

At the same time, CI and the team's `make lint` / `make fix` targets must enforce the
authoritative rule set, which is substantially stricter: it enables over 50 linters
(compared to roughly 20 in the minimal set), enforces additional `depguard` rules that
guard layer boundaries (e.g. restricting `reflect`, `os`, DI container, and zap logger
usage to their permitted scopes), and runs `forbidigo` patterns. The heavier analysis takes
minutes rather than seconds.

Running identical configs in both contexts is not practical: a multi-minute full lint pass
blocks responsive IDE feedback, while a 30-second minimal pass would let CI-only
violations slip through undetected.

## Decision

Maintain two golangci-lint configuration files with a clear division of authority:

- `.golangci.yaml` — the implicit default, picked up by IDE tooling automatically. Enables
  a curated minimal set of linters with a 30-second timeout. It is intentionally not
  the gate that fails CI.
- `.golangci-full.yaml` — the authoritative CI gate. Both `make lint` and `make fix` pass
  `--config .golangci-full.yaml` explicitly (see the `lint` and `fix` targets in
  `.makefiles/go/golangci-lint.mk`). This config enables the complete linter set including all `depguard` layer
  rules and carries no fixed timeout of its own: a full run grows with the repository, so
  a fixed budget in the config would go stale. The cutoff belongs to whichever entry point
  runs it — `GOLANGCI_LINT_TIMEOUT` in the makefile locally, `timeout-minutes` on the
  `go-lint` job in CI.

Any rule that must be enforced as a hard gate belongs in `.golangci-full.yaml` only.
The minimal config may contain a subset; drift between the two is intentional.

## Consequences

### Positive Consequences

- IDE tooling stays responsive: authors get fast, low-noise feedback without waiting for
  the full suite.
- CI enforcement is unambiguous: the gate is always the full config, regardless of what
  the editor runs.
- Layer-boundary rules (depguard) and dangerous-identifier rules (forbidigo) are enforced
  in CI even if editors never surface them.

### Negative Consequences

- A rule added to `.golangci-full.yaml` is not automatically visible in the editor unless
  it is also added to `.golangci.yaml`. Discoverability of new rules depends on the
  author remembering to check both files or running `make lint` locally.
- Two files to maintain. Settings common to both (formatter config, some linter settings)
  must be kept in sync manually.

## Alternatives Considered

### Single shared config

One `.golangci.yaml` used everywhere. Simple to maintain but forces a choice: either use
the full slow suite in IDEs, or weaken CI to the fast subset. Neither is acceptable.

### golangci-lint `--fast` flag

The `--fast` flag selects a subset of linters. Does not give the fine-grained control
needed to enforce the project-specific `depguard` rules selectively, and the flag's
semantics have changed across golangci-lint versions.

## Notes

- Source: `.golangci.yaml` (minimal default), `.golangci-full.yaml` (full gate),
  the `lint` and `fix` targets in `.makefiles/go/golangci-lint.mk`.
- The `depguard` rules in `.golangci-full.yaml` are the machine-enforced expression of
  the layer dependency rules documented in [`docs/rules.md`](../rules.md).
- Related: [ADR-0002](0002-onion-architecture.md) — the layer boundaries that `depguard`
  enforces.
