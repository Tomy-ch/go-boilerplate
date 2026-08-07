---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [deploy, migration, exclusion, setup-review]
---

# ADR-0095: Migrations run as a pre-deploy one-shot; do NOT auto-migrate at application startup

## Status

accepted

## Context

Database schema migrations must run before the new application version starts serving
traffic. Two common approaches exist: (1) run migrations automatically when the application
starts (startup migration), or (2) run migrations as an explicit one-shot job before the
application is deployed (pre-deploy migration).

Startup migration is appealing because it requires no separate orchestration step. However,
it has well-known failure modes: the migration runs on every replica at startup, requiring
distributed locking to avoid parallel execution; a failed migration crashes the container,
potentially causing a cascade restart loop; and a slow migration delays health-check
readiness, triggering premature container replacement.

The `runtime` image ships the full `migrate-up` subcommand (see
[ADR-0092](0092-single-runtime-image.md)) precisely to support the pre-deploy pattern via
command override.

## Decision

We deliberately do NOT run migrations automatically at application startup (entrypoint).

Migrations are executed as a **pre-deploy one-shot job** in the CI/CD pipeline, invoked
before the `Deploy application` step. The expected invocation is:

```text
docker run <image> /app/server migrate-up
```

or the equivalent cloud-native one-shot job primitive (AWS ECS RunTask, EKS Kubernetes Job,
GCP Cloud Run Job, GKE Kubernetes Job, Azure Container App Job, AKS Kubernetes Job).

The deployment step runs only after the migration job completes successfully.

The `Run migration (one-time job)` step in `.github/workflows/deploy-app.yaml` must be
implemented for the target cloud environment, and the application start step must never
invoke `migrate-up`.

## Consequences

### Positive Consequences

- Migration runs exactly once per deployment, not once per replica per restart.
- A migration failure fails the deploy pipeline; the old application version continues
  serving traffic untouched.
- The application container starts faster (no migration delay) and its health check reflects
  true application readiness.

### Negative Consequences

- The CI/CD pipeline must include an explicit migration step and its credentials / job
  primitives, adding per-environment setup work.
- Migration and application deployment must be sequenced correctly; a pipeline
  misconfiguration that skips the migration step will deploy against the old schema.

## Alternatives Considered

### Startup migration (auto-migrate on application boot)

Simple to implement (call `migrate.Up()` in `main()`). Rejected due to: parallel execution
on multi-replica start, restart loops on failure, delayed readiness, and difficulty isolating
migration failures from application failures in observability.

### Separate migration binary or image

Would require maintaining a second artifact. The single-image command-override approach
achieves the same result without the overhead (see [ADR-0092](0092-single-runtime-image.md)).

## Notes

- `.github/workflows/deploy-app.yaml` `Run migration (one-time job)` step (lines 192-204)
  is the authoritative placeholder to implement for the target environment.
- The workflow comment "Must NOT be executed in container startup (entrypoint)" is the
  explicit guard against the rejected alternative.
- Source: `.github/workflows/deploy-app.yaml`.
