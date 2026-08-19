---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [job, async]
---

# ADR-0070: Jobs are explicitly registered (no auto-discovery)

## Status

accepted

## Context

The job subsystem's runner dispatches a CLI invocation to a named `Job` implementation.
The runner must know which jobs are available at startup. Two general patterns exist:

1. **Auto-discovery** — the process scans a directory, uses reflection, or relies on
   `init`-function side effects to find all registered jobs at boot.
2. **Explicit registration** — each job constructor is deliberately listed in a central
   wiring file; the DI container aggregates them at compile time.

The repository uses Uber Fx for dependency injection (see [ADR-0040](0040-uber-fx-di.md)),
which provides the `fx.Annotate` / `group` tag mechanism for collecting multiple
implementations of the same interface into a slice.

## Decision

Jobs are **explicitly registered** via Fx's value-group mechanism — there is no
auto-discovery. Each job constructor is added to `provideJobs(...)` inside `JobModule()`
(`internal/di/module/job.go`). Fx collects all tagged constructors into the slice that
`NewRunner` consumes.

Duplicate job names are rejected at build time (`ErrDuplicateJob`), so a mis-registration
is caught before the process starts.

## Consequences

### Positive Consequences

- The full list of available jobs is visible in a single file (`internal/di/module/job.go`)
  — easy to audit and review.
- No reflection or `init`-based side effects; the dependency graph is statically verifiable
  by Fx.
- Duplicate names produce a deterministic build-time error, not a silent runtime collision.

### Negative Consequences

- Adding a new job requires editing `internal/di/module/job.go` in addition to the job
  package itself — two touch points instead of one.
- There is no automatic enforcement that a newly authored `Job` is actually wired; a job
  implementation can be written but omitted from the registration list.

## Alternatives Considered

### Auto-discovery via init functions or reflection

Rejected: `init`-based registries introduce implicit global state that makes the dependency
graph invisible to Fx and harder to test in isolation. Reflection-based discovery couples
the runner to naming conventions or package paths, which are fragile refactoring targets.

### Code generation for the registration list

Not adopted: the explicit list is short and human-readable; the overhead of a generation
step is not justified at this scale.

## Notes

- The same explicit-registration pattern is used for workers (`provideWorkers` in
  `internal/di/module/worker.go`).
- Source: `docs/design/job.md` (§ 4 "What an integrator implements", intro sentence and
  registration table).
- Related: [ADR-0040](0040-uber-fx-di.md) (Uber Fx DI).
