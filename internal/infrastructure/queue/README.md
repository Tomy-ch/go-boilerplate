# infrastructure/queue

Adapters that implement the worker seam (`internal/usecase/boundary/worker`) against a
concrete message broker.

## Layout convention

- **Broker-agnostic contract** lives at `internal/usecase/boundary/worker` (the seam), above
  this layer — not here. Infrastructure implements those ports; it does not own the abstraction.
- **Broker-specific adapter** lives at `queue/<broker>/` (e.g. `queue/sqs`). The package name is
  the broker, so the concrete technology stays visible at the import site.
- **Code shared across brokers** lives directly under `queue/`. Shared code is extracted only
  when two or more adapters duplicate a concrete implementation detail, so a helper is hoisted
  from observed duplication rather than designed up front.
