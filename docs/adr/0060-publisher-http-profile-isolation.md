---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [outbox, http]
---

# ADR-0060: Isolate the publisher's non-standard HTTP profile inside the relay

## Status

accepted

## Context

The outbox HTTP publisher requires three deviations from the shared default HTTP client
profile:

1. **`MaxAttempts = 1`** — transport-level retry must be disabled because the relay poll
   loop is the at-least-once retry mechanism (see [ADR-0055](0055-at-least-once-outbox-poll.md)).
   Enabling both would cause double-retry amplification (decision D10).

2. **`PropagateTrace = false`** — the W3C `traceparent` header is captured at emit time
   and stored verbatim in the `headers` column. The publisher replays that stored value so
   the receiver joins the originating trace span. Automatic trace-context injection at send
   time would overwrite the stored `traceparent` with the relay's own span, severing the
   trace continuity.

3. **`AllowPrivateNetwork = false`** — the receiver endpoint is an external service;
   delivering to private or loopback addresses is rejected to suppress SSRF risk (see
   ADR-0024 and ADR-0025 for the resilience and SSRF client policies).

If this profile were registered in the shared `InfrastructureModule`, it would be
contributed to the DI graph of every process that uses that module — including the main
API server — potentially overriding or conflicting with the standard HTTP profile used by
other downstream clients.

## Decision

`outboxPublisherModule` — which registers the non-standard `DownstreamProfile` via
`NewDownstreamProfile` — is **nested inside `OutboxRelayModule`** and is therefore
composed only in the relay-dedicated process (`cmd outbox-relay`). It is never included
in the main server's `InfrastructureModule`. The nesting means the non-standard profile
cannot leak into any other process.

```text
OutboxRelayModule
└── outboxPublisherModule          ← registers the non-standard DownstreamProfile
    └── provideHTTPClientProfiles  ← contributes to the value group
```

## Consequences

### Positive Consequences

- The non-standard profile is strictly scoped to the relay process; it cannot
  accidentally affect other downstream HTTP clients.
- Profile ownership is co-located with the relay entry point, making the constraint
  auditable from the DI graph alone without reading `http_publisher.go`.
- Observability (metrics, tracing) from the shared `httpclient` infrastructure remains
  available to the publisher without compromising the profile.

### Negative Consequences

- The relay process carries its own DI graph, which is a superset of the main server
  graph; composition code is larger.
- A developer adding a publisher-related module must be aware of the relay/server split
  to place their module in the correct `fx.Module`.

## Alternatives Considered

### Register the profile in InfrastructureModule with a feature flag

Adds runtime configuration surface and makes the profile conditional on an env var rather
than process identity. This is weaker: a misconfigured flag in the main server could
activate the wrong profile.

### Separate HTTP client instance not managed by the shared profile system

Breaks shared observability (RED metrics, tracing) and bypasses the policy enforcement
(SSRF, circuit-breaker, budget) provided by the profile system.

## Notes

- Non-standard profile rationale: `docs/design/outbox.md` (§ "Package placement and
  dependency direction", Glossary entry "OutboxRelayModule").
- Implementation:
  - `internal/di/module/outboxpublisher.go` (`outboxPublisherModule`, `NewDownstreamProfile`)
  - `internal/di/module/outboxrelay.go` (`OutboxRelayModule`)
  - `internal/infrastructure/publisher/http_publisher.go` (`NewDownstreamProfile`)
- Resilience and SSRF client policies: [ADR-0024](0024-outbound-http-resilience.md),
  [ADR-0025](0025-egress-ssrf-guard.md).
- Related ADRs: [ADR-0055](0055-at-least-once-outbox-poll.md),
  [ADR-0061](0061-relay-resident-gc-oneshot.md).
