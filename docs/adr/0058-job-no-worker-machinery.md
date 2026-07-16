---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [job, async, exclusion, setup-review]
---

# ADR-0058: Jobs deliberately have no broker, circuit breaker, drain, or health machinery

## Status

accepted

## Context

The job subsystem and the worker subsystem are both async entry points into the Usecase
layer, and they share the same Fx DI container pattern. The worker engine
([ADR-0040](0040-broker-agnostic-worker-scaffold.md)) provides a rich operational layer:
a pull-ack broker seam, a 3-state circuit breaker on the intake side, a drain phase on
shutdown (waiting for in-flight messages), and an HTTP health listener (`/healthz`,
`/readyz`).

Because jobs are implemented in the same codebase and share the Fx lifecycle
infrastructure, there is a temptation to reuse or replicate these mechanisms — for example,
adding a circuit breaker to retry failed jobs, a health endpoint to observe long-running
jobs, or a drain phase to allow a job to finish gracefully before SIGTERM.

## Decision

We deliberately do NOT provide broker integration, a circuit breaker, drain-phase logic, or
a health listener in the job subsystem.

A job is a **one-shot CLI invocation**: it runs exactly one unit of work, returns a result
to the CLI, and exits. This is fundamentally different from the resident worker, which polls
indefinitely and must manage concurrent in-flight messages over an unbounded lifetime:

- **No broker**: a job is invoked by an operator or a scheduler directly via the CLI, not
  by a message queue. Retry and scheduling are the caller's responsibility (e.g. cron, a
  CI step, or an operator manual re-run).
- **No circuit breaker**: a one-shot invocation either succeeds or fails; a circuit breaker
  has no intake stream to throttle and no repeated attempts to protect against.
- **No drain phase**: there is only one unit of work; the graceful-stop window (default 45 s, `SHUTDOWN_TIMEOUT`) is
  provided to allow Fx lifecycle hooks to clean up, but there are no in-flight messages to
  drain.
- **No health listener**: a job runs for a bounded duration under direct operator control;
  there is no long-lived process for a health check to observe.

Job failures are returned to the CLI as a non-zero exit code and logged with structured
fields (`JobResultKey`, `JobErrorKey`).

## Consequences

### Positive Consequences

- The job subsystem stays simple and auditable — the lifecycle reduces to start, run, stop.
- No operational machinery means no configuration surface to tune (no `FailureThreshold`,
  `DrainTimeout`, `ProgressStaleAfter`, etc.).
- The contrast with the worker is explicit and intentional, preventing accidental feature
  drift toward a second worker implementation.

### Negative Consequences

- Long-running jobs that could benefit from a health endpoint require external monitoring
  (e.g. process supervision, structured log tailing) rather than a standard `/readyz`.
- Retry logic for failed jobs must be implemented by the caller rather than by the job
  subsystem itself.

## Alternatives Considered

### Adding a circuit breaker for automatic retry on failure

Rejected: a job is invoked once. Automatic retry inside the job lifecycle would require a
loop, turning a one-shot command into a resident process — that is the worker's domain.

### Exposing a health endpoint for long-running jobs

Rejected: the job process is ephemeral and directly observable via its exit code and
structured logs. A health listener would add an HTTP server whose lifetime outlives its
usefulness for a process expected to exit imminently.

## Notes

- Contrast with [ADR-0040](0040-broker-agnostic-worker-scaffold.md) (worker: broker seam,
  circuit breaker, drain, health listener).
- Source: `docs/design/job.md` (§ 1 Role theory, design principle paragraph).
