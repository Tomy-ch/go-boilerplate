---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [ci, testing]
---

# ADR-0082: CI boots the real fx graph against real Postgres (startup verification)

## Status

accepted

## Context

Unit tests and integration tests validate individual components in isolation or with mocks,
but they do not prove that the full dependency injection graph assembles correctly against
real infrastructure. A missing `fx.Provide`, a misconfigured database connection, or a
wiring error in `internal/di` can pass all unit tests yet fail at runtime with an opaque
panic or fatal log.

The project has three distinct entrypoints — HTTP server, background worker, and one-off
job — each with its own DI graph. All three must be verified on every pull request that
touches application code.

Historically, DI wiring bugs are discovered late: after deployment to staging, or during
manual testing. Catching them in CI on every PR requires a lightweight check that
exercises the real startup path without needing a running broker or full seed data.

## Decision

Introduce three dedicated CI workflow jobs that each boot the real fx graph against a
real Postgres container and verify that the graph assembled and the entrypoint reached
its dispatch logic:

**Server boot check** (`app-di-startup-check.yaml`, lines 58–83):
Build the server binary, start it in the background, then poll `GET /ready` up to 30
times with 2-second intervals. A successful readiness response proves that fx completed
dependency injection, the database connection was established, and the HTTP stack reached
a serving state.

**Worker boot check** (`worker-boot-check.yaml`, lines 60–82):
Pass a deliberately non-existent worker name (`__ci_boot_check_no_such_worker__`) to
`go run ./cmd/ worker`. Assert that the process exits non-zero AND that the output
contains the string `"unknown worker"`. A missing `"unknown worker"` message with a
non-zero exit means DI or database setup failed before the dispatch point — which would
be a hard failure, not just an unknown worker.

**Job boot check** (`job-boot-check.yaml`, lines 59–81):
Same pattern as the worker check but for the job entrypoint: pass
`__ci_boot_check_no_such_job__`, expect a non-zero exit containing `"unknown job"`.

All three jobs spin up a real Postgres 18 container via the `services:` block and run
`make materialize-env` to embed the CI environment configuration before executing.

## Consequences

### Positive Consequences

- DI wiring bugs are caught at PR time on the actual graph, not discovered post-deploy.
- The worker and job checks exercise the full startup path (DI construction, DB
  connection) without requiring a running broker or seed data.
- The server check uses the existing `/ready` health endpoint rather than introducing a
  new test-only path.
- Each entrypoint is covered independently; a wiring regression in one does not hide
  behind another's success.

### Negative Consequences

- Each check spins up a real Postgres container, adding to CI runtime and resource usage.
- The worker and job checks rely on a sentinel string in the error output. If the error
  message changes, the check must be updated.
- The server check's polling loop (30 retries × 2 s = 60 s max) means a genuine boot
  failure takes up to one minute to be detected.

## Alternatives Considered

### Mock the fx graph in tests

Possible, but a mocked DI graph does not exercise the real wiring in `internal/di`.
It would miss missing providers and circular dependencies.

### Rely on staging deployment for boot verification

Feedback loop is too long (hours to days), and failures affect the staging environment
rather than being contained to the PR.

### Single combined entrypoint check

A single job could sequentially boot all three entrypoints. Chosen against in favour of
three independent jobs so that a failure in one does not mask the others and CI reports
are clearly attributed.

## Notes

- Sources: `.github/workflows/app-di-startup-check.yaml` lines 58–83,
  `.github/workflows/worker-boot-check.yaml` lines 60–82,
  `.github/workflows/job-boot-check.yaml` lines 59–81.
- Related: [ADR-0035](0035-uber-fx-di.md) — the Uber fx DI container whose graph this
  check exercises.
- Related: [ADR-0045](0045-broker-agnostic-worker-scaffold.md) — the worker subsystem
  whose entrypoint is verified.
