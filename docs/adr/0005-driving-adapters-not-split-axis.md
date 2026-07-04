---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [foundational, architecture]
---

# ADR-0005: REST / Worker / Job are driving adapters, not a service-split axis

## Status

accepted

## Context

Three entry-point types co-exist in this template:

- **REST** — synchronous HTTP requests handled by Echo handlers.
- **Worker** — asynchronous messages pulled from a queue and processed by worker handlers.
- **Job** — CLI-invoked or scheduled commands processed by job handlers.

A natural misreading of this structure is to treat entry-point type as a reason to split
Usecase or Domain packages: "REST usecases", "worker usecases", "job usecases". This
produces duplicated business logic across transport-specific subtrees and erodes the
single-responsibility of the Usecase layer.

The correct framing is that all three are **driving adapters** — they adapt an external
signal (HTTP request, queue message, CLI invocation) into a call on the Usecase layer. The
Usecase and Domain layers are split by **business capability, business context, and
transaction boundary** — not by the transport mechanism that triggered the operation.

REST is the original entry point; Worker ("message-in driving adapter") and Job
("command-in driving adapter") were modelled after the same pattern.

## Decision

REST, Worker, and Job entry points are **driving adapters** into the Usecase layer. They are
not a basis for splitting Usecase or Domain packages. Usecase and Domain packages are
organized by business capability and transaction boundary.

Each adapter type lives under `internal/controller/` and calls the same Usecase methods that
any other adapter would call for the same business operation.

## Consequences

### Positive Consequences

- Business logic is implemented once in the Usecase / Domain layers and invoked by any
  adapter; no duplication across transport types.
- Adding a new entry-point type (e.g., gRPC) requires only a new adapter under
  `internal/controller/` — the Usecase layer is unchanged.
- Layer boundaries enforced by depguard (see [ADR-0006](0006-structural-safety-via-tooling.md))
  apply uniformly across all adapter types.

### Negative Consequences

- Developers unfamiliar with the driving-adapter pattern may instinctively create
  per-transport usecase packages; the design doc and this ADR serve as the corrective
  reference.
- All adapters share the same Usecase method signatures; if one transport has significantly
  different input / output shapes, the DTO mapping in the adapter grows.

## Alternatives Considered

### Per-transport Usecase packages

Separate usecase subtrees for REST, Worker, and Job. Rejected: business logic is duplicated
across transport-specific packages, violating the single-responsibility of the Usecase layer.

### Single controller package with no adapter distinction

Collapsing all entry points into one package. Rejected: the distinct lifecycle and error
semantics of HTTP, queue, and CLI entry points justify separate adapter packages under
`internal/controller/`.

## Notes

- Source: `docs/design/README.md` § "How to read these" (first invariant: REST / Job / Worker
  are driving adapters into the Usecase layer, not package split axes).
- Source: `docs/design/rest.md` § "1. Role theory" — REST defined as the
  "request-in driving adapter, the synchronous HTTP entry point into the Usecase layer".
- Layer boundary enforcement: [ADR-0006](0006-structural-safety-via-tooling.md).
- Layer shape: [ADR-0002](0002-onion-architecture.md).
