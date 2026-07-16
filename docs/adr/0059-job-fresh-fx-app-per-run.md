---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [job, async]
---

# ADR-0058: Each job launch constructs a fresh fx.App (one-shot lifecycle)

## Status

accepted

## Context

The job subsystem is a CLI-driven entry point into the Usecase layer — on par with the HTTP
handler and the worker, but invoked once and expected to exit. It uses the same Uber Fx
dependency-injection container as the resident `serve` and `worker` commands (see
[ADR-0032](0032-uber-fx-di.md)).

Two lifecycle models were possible:

1. **Shared / resident app** — a single long-lived `fx.App` instance started at process
   boot, with jobs dispatched into it across invocations.
2. **Per-invocation app** — a fresh `fx.App` constructed, started, and stopped for each
   job invocation.

The job command must exit cleanly after running exactly one job, returning a well-defined
exit code, with no residual state leaking between invocations.

## Decision

Each `job` subcommand invocation constructs a **fresh `fx.App`** (no resident process).
`di.RunJob` / `NewJobCore` compose the container, `app.Start` fires lifecycle hooks, the
job executes on a detached goroutine whose run context comes from `SupervisedRunner`
(`context.WithCancel(context.Background())`), so it is not cancelled when the `OnStart`
start-context returns, and `app.Stop` tears everything down before the process exits. The job result is returned to the CLI as a non-zero exit code on failure.

## Consequences

### Positive Consequences

- Each invocation gets a clean dependency graph with no shared mutable state from a prior
  run.
- Startup and teardown ordering are fully managed by Fx lifecycle hooks (OnStart / OnStop),
  the same mechanism used by `serve` and `worker`.
- The one-shot pattern maps directly to the "run exactly one job, then exit" requirement
  without special-casing inside a long-lived process.

### Negative Consequences

- Fx startup overhead (container construction, OnStart hooks) occurs on every invocation,
  even for trivially short jobs.
- Connection pools and other expensive resources (e.g. database) are initialised and torn
  down per invocation; this is acceptable for infrequent CLI jobs but would be wasteful for
  high-frequency automated dispatch.

## Alternatives Considered

### A resident process that dispatches jobs on demand

Rejected: the job command is a one-shot CLI invocation, not a daemon. A resident process
would require a separate dispatch protocol, complicate lifecycle management, and add
supervisory machinery that belongs to the worker subsystem instead.

### Reusing the serve / worker fx.App

Rejected: different commands wire different dependencies; a shared app would either
over-provide (unused infrastructure) or require conditional wiring that obscures the
dependency graph.

## Notes

- The Shutdowner (`fx.Shutdowner`) is used by the job hook goroutine to request `app.Stop`
  after the job finishes, completing the one-shot lifecycle.
- Source: `docs/design/job.md` (§ 1 Role theory, design principle paragraph and
  responsibility table row "DI / cli / cmd").
- Related: [ADR-0032](0032-uber-fx-di.md) (Uber Fx DI).
