---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [di, config]
---

# ADR-0042: Swap implementations per environment via DI (env-gated wiring)

## Status

accepted

## Context

Some components require different concrete implementations depending on the deployment
environment. The authorizer is the canonical example: a full RBAC or external
policy-engine implementation is needed for production, but local development, CI, and
test environments need a simpler stub that does not depend on external policy
infrastructure.

If environment branching happens inside application code (domain, usecase, or
controller), protocol or environment knowledge bleeds into inner layers. If it happens
implicitly (feature flags, late-binding), the startup guarantees weaken and the wrong
implementation can silently run in production.

## Decision

Environment branching for concrete implementations happens exclusively at the
**composition root** — inside the DI module provider that constructs the component.
The provider reads the environment identifier from `config.ApplicationConfig`, selects
the appropriate implementation, and returns it as the declared interface type.

Production-incompatible stubs **fail closed**: when no production implementation is
wired for the current environment, the provider returns an error so `fx.App` fails to
start, preventing an unsafe stub from running silently. A `WARN`-level log is emitted
when a non-production stub is wired so the substitution is visible in logs.

The authorizer provider (`provideAuthorizer` in `internal/di/module/authz.go`) exemplifies
this pattern:

```go
switch appCfg.Env() {
case config.EnvLocal, config.EnvCI, config.EnvTest:
    logger.Warn("Allow-all authorizer wired: every request is permitted (non-production only)", ...)
    return allowall.New(), nil
default:
    logger.Error("No authorizer configured for the current environment", ...)
    return nil, xerrors.New("no authorizer configured for environment: " + appCfg.Env())
}
```

## Consequences

### Positive Consequences

- The composition root is the single, auditable place where environment-specific
  implementation selection occurs.
- Unsafe stubs cannot silently reach production: the provider fails closed with a
  startup error and a log entry.
- Inner layers (usecase, domain) depend only on the declared interface; they are
  unaware of which concrete implementation is injected.

### Negative Consequences

- Each env-gated provider needs test coverage for every branch (allow-all path and
  fail-closed path), or the selection logic is exercised only by the graph-validation
  test.
- A new provider must be written (or an existing one extended) when a new environment
  variant is introduced.

## Alternatives Considered

### Runtime feature flags

Defer the implementation choice to runtime via a flag or configuration value. Rejected:
harder to reason about at startup, and a misconfigured flag could silently run the
wrong implementation in production without a startup failure.

### Separate DI module files per environment

Compile separate module files for each environment. Rejected: increases build
complexity and diverges from the single-binary model.

## Notes

- Source: `internal/di/module/authz.go` (`provideAuthorizer`),
  `internal/di/module/README.md`.
- The module README documents this pattern as: "environment-gated: wires the allow-all
  stub only for local / CI / test and fails closed (returns an error) elsewhere".
- Config constants (`EnvLocal`, `EnvCI`, `EnvTest`) are defined in `internal/config`.
