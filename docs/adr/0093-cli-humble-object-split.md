---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [cli, architecture]
---

# ADR-0093: CLI humble-object split (thin cmd/ shell + testable internal/cli core)

## Status

accepted

## Context

CLI commands need business logic that is testable without launching real databases, external
processes, or the file system, while still wiring real dependencies in production. A single
Cobra handler function that both defines the command and contains the logic conflates two
concerns — testability and wiring — making the decision logic impossible to unit-test in
isolation.

The coverage gate ([ADR-0084](0084-coverage-hard-gate.md), 90%+ branch coverage) must apply to command logic, but `cmd/` shell files
that import Cobra, `internal/di`, and OS signals cannot be unit-tested cheaply; their runtime
correctness is instead verified by CI boot checks (`app-di-startup-check`, `job-boot-check`,
`worker-boot-check`, `migration-check`, and `gen-*-artifacts-check`) against a real Postgres
service.

## Decision

Split every CLI command into two parts:

1. **A thin `cmd/<command>.go` shell** — defines the Cobra command, parses flags into local
   variables, wires real OS / DB / logger dependencies, and delegates to a single core
   function. This file is excluded from the coverage gate.
2. **A testable `internal/cli/<command>/` core package** — contains all decision logic
   (error handling, branching, formatting, deletion conditions, timeout dispatch). All
   OS / filesystem / external-process / DB / logger dependencies are injected via interfaces
   or function seams; production code wires the real implementations; unit tests pass fakes.
   This package is included in the coverage gate and must meet 90%+ branch coverage.

The core must not import Cobra, `internal/di`, `internal/config`, OS signals, or
infrastructure (other than `infrastructure/rdb/driver` types for type-passing purposes).

## Consequences

### Positive Consequences

- Decision logic is fully unit-testable without real infrastructure.
- The `cmd/` shell stays to a handful of lines (config load → dependency build → delegate),
  making it easy to read and keeping wiring concerns separate from logic concerns.
- The coverage gate is enforceable: `internal/cli/*` meets 90%+ branch coverage; `cmd/` is
  exempt and guarded by CI boot checks instead.
- Adding a new command is mechanical: add `cmd/<command>.go`, add core logic under
  `internal/cli/<command>/`, register in `registerCommands`.

### Negative Consequences

- Two files must be created for every new command instead of one.
- The interface seam between shell and core requires designing injected interfaces upfront,
  which adds a small design overhead for simple commands.

## Alternatives Considered

### Cobra handler contains all logic

Simple for small commands, but decision logic becomes untestable in isolation. As commands
grow, this tends toward integration-test-only coverage or test-skipped code paths.

### Separate CLI binary per command

Avoids Cobra registration but requires distributing multiple binaries, complicates
deployment, and prevents sharing initialization code (config loading, DI). Inconsistent
with the single-binary decision (see [ADR-0094](0094-single-multi-command-binary.md)).

## Notes

- Pattern illustrated in `cmd/outbox_relay.go` (thin shell) and `internal/cli/outbox/`
  (core package).
- `registerCommands` in `cmd/commands.go` is the single registration point for all
  subcommands.
- Source: `internal/cli/README.md` §"Design Policy" and §"Testing Policy".
