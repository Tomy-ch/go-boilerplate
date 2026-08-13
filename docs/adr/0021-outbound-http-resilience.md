---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [http, resilience, infrastructure]
---

# ADR-0021: Provide an outbound-HTTP resilience foundation (retry / circuit breaker / retry budget / dual timeout)

## Status

accepted

## Context

Outbound HTTP calls — to external APIs, internal services, and webhook endpoints — are an
inherent reliability boundary. Without a shared resilience layer, each gateway or publisher
implementation must either reinvent retry logic or skip it entirely. The failure modes without
a shared substrate are:

- **Retry storms.** An upstream spike triggers many gateway callers to retry concurrently,
  amplifying load on the already-degraded downstream.
- **Inadvertent retry of non-idempotent requests.** A POST retried blindly creates duplicate
  side effects unless the downstream deduplicates; gateways that manage retries locally lack
  the information to enforce the distinction safely.
- **Goroutine accumulation from slow responses.** Without a per-attempt timeout that is
  shorter than the overall deadline, a single slow downstream can hold goroutines open for the
  full request lifetime.
- **No downstream-health awareness.** Without a circuit breaker, every request attempts the
  full retry budget even when the downstream has been consistently failing, adding latency
  without benefit.
- **Ad-hoc error mapping.** Callers that inspect raw HTTP status codes couple themselves to
  HTTP semantics and diverge from the rest of the codebase's `apperror` taxonomy.

The project already has an `apperror` error taxonomy (used at every other infra boundary) and
uses [ADR-0001](0001-avoid-lock-in.md) to prefer minimal, replaceable dependencies. A shared
substrate that sits between the transport and the semantic gateway implementations keeps
resilience logic in one place and keeps gateway code thin.

## Decision

All outbound HTTP calls are routed through `internal/infrastructure/httpclient.Client`, a
resilient substrate (the HTTP counterpart of `rdb/driver`). Infrastructure code — gateways
(`webapi/<service>`) and publishers — depends on this substrate; bare `net/http.Client` is
not used directly in gateways or publishers.

The substrate enforces four resilience mechanisms:

**Retry policy.** Idempotent methods (GET, PUT, DELETE) are always retry-safe. Non-idempotent
methods (POST, PATCH) are retried only when the caller sets `AllowRetry` and provides a
non-empty `IdempotencyKey`, which the substrate forwards as an `Idempotency-Key` header so
the downstream can deduplicate. Retryable outcomes are: 5xx responses, 429, and transport
failures (network errors). Non-retryable outcomes are: 4xx (except 429), 2xx, and context
cancellation. Backoff is exponential with full jitter, bounded by `MaxBackoff`, and the
substrate honours a `Retry-After` header when present.

**Retry budget.** A per-downstream token-bucket caps the fraction of requests that may be
retries (default: 10%, `RetryBudgetRatio`). When the budget is exhausted the substrate
returns the last response without further retry, preventing retry amplification under load.

**Circuit breaker.** A per-downstream state machine (closed / half-open / open) tracks the
failure ratio over the last `MinRequests` (default: 20) samples. When the failure rate exceeds
`FailureThreshold` (default: 0.5) the breaker opens and subsequent requests fail fast with
`ErrUnavailable` for `OpenDuration` (default: 5 s). After that it enters half-open and allows
`HalfOpenProbes` (default: 3) probe requests to test recovery.

**Dual timeout.** Each attempt is bounded by `PerAttemptTimeout` (default: 3 s); the full
call including all retries is bounded by `OverallTimeout` (default: 10 s). Before sleeping
between retries the substrate checks whether the backoff wait would overrun the overall
deadline and skips the retry if it would, so the caller's context deadline is always
respected.

Per-downstream profiles (`Profile`) are contributed to an fx value group (`httpclient_profiles`)
and aggregated by `Registry`; unregistered downstreams fall back to `DefaultProfile`.

Transport events and non-2xx responses are normalised to `apperror` sentinels
(`ErrUnavailable`, `ErrCanceled`, `ErrInvalidArgument`, etc.) before being returned to
callers, which branch on sentinels only. `net/http` types are not exposed in the substrate's
public API.

## Consequences

### Positive Consequences

- Retry, budget, circuit breaker, and timeout policies are defined once and applied uniformly
  across every outbound call, eliminating per-gateway divergence.
- Idempotency enforcement at the substrate layer prevents accidental duplicate side effects
  from retried POST/PATCH calls.
- The retry-budget prevents retry amplification: the substrate stops retrying under sustained
  load before it makes the downstream worse.
- The circuit breaker provides fast-fail under sustained downstream faults, reducing tail
  latency for callers.
- Gateway and publisher code focuses on semantic mapping (domain model ↔ request/response);
  resilience concerns are fully hidden behind the `Client` interface.
- Per-downstream profiles allow fine-grained tuning (e.g. the outbox publisher disables
  transport-layer retry because the relay loop provides at-least-once delivery at a higher
  level).

### Negative Consequences

- Gateway implementers must use the substrate's typed `Request` builder (`NewRequest` +
  `With*` options); they cannot construct `http.Request` directly.
- Adding a new downstream requires registering a `DownstreamProfile` in the fx group;
  forgetting to register falls back to `DefaultProfile` silently (though the fallback is
  intentionally safe).
- The per-downstream circuit breaker and budget are in-process state; they do not share state
  across multiple application instances.

## Alternatives Considered

### Bare `net/http.Client` per consumer

The simplest approach: each gateway constructs its own `http.Client`. Rejected because it
pushes all retry, timeout, and error-mapping logic to each consumer individually, producing
divergent implementations and the failure modes listed in the Context section.

### Third-party resiliency library (e.g. go-resilience, failsafe-go)

Libraries provide circuit breaker and retry primitives. Rejected per [ADR-0001](0001-avoid-lock-in.md):
the project prefers thin, replaceable abstractions over framework lock-in. The substrate is
thin enough (retry loop + token bucket + state machine) that the standard library suffices
without an external dependency.

### Retry at the usecase layer

Usecases could wrap gateway calls in a retry loop. Rejected because: (a) retry logic would
bleed into business logic, violating the onion-layer responsibility split
([ADR-0002](0002-onion-architecture.md)); (b) the usecase has no access to transport-level
signals (network errors, `Retry-After` headers) needed to make a correct retry decision; and
(c) the circuit breaker, budget, and timeout mechanisms belong at the infrastructure boundary,
not inside business logic.

### Service mesh / sidecar (e.g. Envoy, Linkerd)

Offloads retry and circuit breaking to the mesh. Rejected because it introduces a hard
infrastructure dependency that makes local development and testing complex, and it conflicts
with this project's goal of running with minimal external dependencies
(see [ADR-0001](0001-avoid-lock-in.md)).

## Notes

- Substrate implementation: `internal/infrastructure/httpclient/`.
- Transport (instrumentation + SSRF guard): `internal/observability/http_client_transport.go`
  — see [ADR-0022](0022-egress-ssrf-guard.md).
- DI registration: `internal/di/module/httpclient.go`.
- Example consumer that disables transport retry in favour of relay-level at-least-once:
  `internal/infrastructure/publisher/http_publisher.go`.
- source: `internal/infrastructure/httpclient/README.md`.
