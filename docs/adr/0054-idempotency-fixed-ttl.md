---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [idempotency]
---

# ADR-0054: Fix idempotency key TTL at 24 hours with no per-route configuration

## Status

accepted

## Context

Idempotency records must expire to bound table growth and to ensure that a key reused
long after the original operation is treated as a fresh request rather than a replay.
The question is whether the TTL should be a single system-wide constant or a value that
can be configured per handler or per route.

A per-route TTL would increase flexibility — a low-risk idempotent endpoint might use a
shorter window, while a financial operation might want a longer one — but it would add
configuration surface, require each handler to supply and document its TTL, and introduce
edge cases when a TTL change is deployed mid-flight. It would also complicate the GC job,
which currently iterates over expired rows without knowledge of per-route policies.

For the template's intended use cases (API writes on the order of minutes to hours of
client retry windows), 24 hours is a practical upper bound that comfortably covers
transient network failures and manual retries.

## Decision

The TTL is fixed at **24 hours** (`ttl = 24 * time.Hour`), coded as a constant in the
`Run[T]` orchestrator. There is no per-route or per-handler configuration flag. Every
claim sets `expires_at = now + 24h`. After the TTL has elapsed, a retry with the same key
is treated as a fresh operation — no cached state is present.

## Consequences

### Positive Consequences

- Handlers have zero TTL configuration to maintain; adoption is a two-step opt-in with no
  policy decisions.
- The GC job can sweep all expired rows with a single `expires_at < now` predicate,
  independent of route metadata.
- Behavior is uniform and predictable across all endpoints.

### Negative Consequences

- 24 hours may be too long for some endpoints (storing a response payload for a full day)
  or too short for others (multi-day retry scenarios).
- Changing the TTL requires a code change and redeployment rather than a configuration
  update.

## Alternatives Considered

### Per-route TTL configuration

Allow each handler to pass a TTL when calling `Run[T]`. Rejected because it significantly
increases integration surface area, makes the GC query more complex, and offers little
practical benefit for the template's intended use cases.

### Configurable system-wide TTL via environment variable

Read the TTL from an environment variable at startup. Rejected because it adds operational
complexity (a misconfiguration silently changes replay semantics) with minimal gain over
the 24-hour constant, which already covers the dominant retry scenarios.

## Notes

- Source: [`docs/design/idempotency.md`](../design/idempotency.md) §4 (operational notes, "TTL =
  24h") and §5 (glossary entry for "ttl").
- The `expires_at` column is indexed in the `idempotency_keys` migration so the GC's
  range scan remains cheap.
- Related ADR-0056 (idempotency GC as a separate job) relies on this fixed TTL to keep
  the sweep query simple.
