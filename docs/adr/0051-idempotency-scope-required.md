---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [idempotency, security]
---

# ADR-0051: Every Store call requires an explicit scope to prevent cross-user key collisions

## Status

accepted

## Context

An idempotency key is a client-chosen string. If key uniqueness is enforced globally
(across all users), two different users using the same key string — a common value such as
`"retry-1"` or a UUID that happens to collide — would share the same idempotency record.
This would allow one user's request to replay another user's response, which is an
Insecure Direct Object Reference (IDOR) vulnerability.

In addition, without scope isolation a malicious client could deliberately use another
user's key to probe whether that user performed a particular operation.

The authentication context is always available at the point the idempotency middleware runs
(the subsystem requires authentication as a prerequisite), making the authenticated
principal a natural scoping unit.

## Decision

Every method on the `Store` interface takes a mandatory `scope` parameter. There is no
key-only lookup. The underlying `idempotency_keys` table enforces `UNIQUE(scope,
idempotency_key)`, so uniqueness is always scoped to an authenticated principal. The scope
value is the authenticated subject (`authn.Subject()`) resolved by the middleware before
the key is processed.

## Consequences

### Positive Consequences

- IDOR is prevented at the persistence layer — even if application code were to pass the
  wrong scope, the DB constraint would catch conflicting claims.
- Scope isolation is a compile-time contract: a `Store` call without a scope argument does
  not compile.
- Different users may reuse the same key string without interference.

### Negative Consequences

- Every caller must supply a scope; anonymous or partially authenticated flows cannot use
  the idempotency subsystem without providing a principal identifier.
- Scope is derived from the authentication context, coupling the idempotency subsystem to
  the authentication mechanism.

## Alternatives Considered

### Global key namespace (no scope)

Enforce uniqueness on the key string alone. Rejected because it introduces IDOR: one
user's key could collide with and replay another user's stored response.

### Caller-specified arbitrary scope string

Allow callers to pass any string as a scope rather than requiring the authenticated subject.
Rejected because it shifts the security responsibility to each call site, making it easy
to accidentally use a constant or empty scope and re-introduce the collision problem.

## Notes

- Source: [`docs/design/idempotency.md`](../design/idempotency.md) §1 (design principles, "Scope
  is mandatory") and §5 (glossary entry for "scope").
- The `UNIQUE(scope, idempotency_key)` constraint is defined in the
  `database/migrations/` idempotency migration.
- Related: [ADR-0002](0002-onion-architecture.md) (the middleware resolves scope; the
  `Store` seam enforces it — responsibility is correctly split across layers).
