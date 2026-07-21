# Design References

日本語: [README.ja.md](../ja/design/README.ja.md)

This directory holds the **per-subsystem design references**. Each one consolidates a single subsystem's *role theory, state transitions / lifecycles, implementation locations, what an integrator must implement, and glossary* into one page, **derived from a close reading of the implementation**.

These complement — they do not replace — the package `README`s: a README is the API/overview for one package, while a design reference is the cross-package narrative for a whole subsystem. Where a design judgment has an ADR, it lives in [`docs/adr/`](../adr/README.md).

## How to read these

You do **not** need to read every document. Each subsystem is adopted independently — read
only the reference for the subsystem you are actually using.

Two invariants underpin all of them:

- **REST / Job / Worker are *driving adapters* into the Usecase layer. They are not package
  split axes for Usecase or Domain.** Usecase and Domain are split by business capability,
  business context, and transaction boundary — never by transport (REST vs Job vs Worker).
- **Layer boundaries are enforced by `golangci-lint` depguard, not only by documentation.**
  A cross-layer import (e.g. `domain` importing `infrastructure`) fails CI, so these design
  docs describe *why* the boundaries exist while the linter guarantees they hold.

## Documents

| Document | Subsystem | What it covers | Package README |
| --- | --- | --- | --- |
| [rest.md](rest.md) | REST (HTTP) scaffold | the synchronous request entry point: handler authoring, routing, error mapping | [controller](../../internal/controller/README.md) |
| [worker.md](worker.md) | Worker scaffold | the pull-ack queue entry point: engine, seam, Ack/Nack, circuit, drain | [worker](../../internal/controller/worker/README.md) |
| [job.md](job.md) | Job scaffold | the CLI / scheduled entry point and its state transitions | [job](../../internal/controller/job/README.md) |
| [outbox.md](outbox.md) | Transactional outbox | reliable event publication via the outbox pattern | [outbox](../../internal/usecase/boundary/outbox/README.md) |
| [idempotency.md](idempotency.md) | Idempotency | the `Idempotency-Key` subsystem and its GC job | [idempotency](../../internal/usecase/idempotency/README.md) |
| [observability.md](observability.md) | Observability | the cross-cutting traces / metrics / logs substrate | [observability](../../internal/observability/README.md) |
| [auth.md](auth.md) | Authentication | RS-side JWT / JWKS verification and the development OIDC provider (`mock-auth-server`) | [jwt](../../internal/infrastructure/auth/jwt/README.md) |

## Reading order

The documents are independent, but they read naturally as **entry points → reliability subsystems → cross-cutting**:

1. **Entry points** — [rest](rest.md) (sync), [worker](worker.md) (async), [job](job.md) (CLI / scheduled)
2. **Reliability subsystems** — [outbox](outbox.md), [idempotency](idempotency.md)
3. **Cross-cutting** — [observability](observability.md), [auth](auth.md)
