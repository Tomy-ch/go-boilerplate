---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [contract, openapi, domain]
---

# ADR-0015: OpenAPI is the wire contract, not the domain rule; request is subset of domain, domain is subset of response

## Status

accepted

## Context

OpenAPI constraints such as `maxLength`, `minimum`, and `maximum` look like single facts
about a field, but the same boundary value can live in several layers owned by different
concerns: the wire contract (OpenAPI), the business rule (domain), and physical storage
(database). Treating an OpenAPI constraint as equivalent to the domain's business rule
leads to incorrect conclusions — for example, assuming that tightening an OpenAPI request
constraint also tightens the domain's validity check, or that a response constraint can be
made equal to the request constraint without breaking server-side correctness.

## Decision

The three layers own their constraints independently:

| Concern | Owner | Where it lives | What it means |
| --- | --- | --- | --- |
| Wire contract | OpenAPI | `openapi/components/schemas/*.yaml` | What the HTTP API accepts (request) or promises (response) |
| Business rule | domain | `internal/domain/<aggregate>/constant.go` | What the business considers valid |
| Storage capacity | DB | `database/migrations/*.sql` | Physical column limit |

These three numbers answer different questions and change for different reasons. They often
coincide by convenience, but they must not be assumed to be equivalent.

The **direction invariant** governs how the three layers relate on the input side:

```text
OpenAPI request constraint  ⊆  domain rule  ⊆  OpenAPI response capacity
        (tightest)                                    (loosest)
```

- The request constraint may be *stricter* than the domain rule. The request-validation
  middleware (see [ADR-0013](0013-spec-driven-request-validation.md)) rejects out-of-range
  input before the domain sees it, so a stricter wire limit is safe.
- The response constraint must be at least as permissive as the domain rule. If the domain
  (or any non-HTTP write path) can produce a value that the response schema forbids, the
  server emits a contract violation that nothing on the server side catches — there is no
  runtime response validation.

A violation at each layer also surfaces as a *different* error, which helps place
responsibility: a request-constraint violation is rejected by the validation middleware as a
`400` before the domain runs; a domain-rule violation is a business validation error raised
by the domain; a response-constraint violation is a server-side contract breach that nothing
catches at runtime (it must be prevented by construction, since responses are not validated).

When modifying a constraint value, decide from its own concern. Do not copy a domain
constant into OpenAPI or vice versa and assume they must stay equal.

## Consequences

### Positive Consequences

- Domain business rules remain stable independently of HTTP API versioning decisions.
- The request middleware can legitimately reject values that the domain would still accept
  (defensive wire boundary), without requiring a domain change.
- The direction invariant is checkable: a test that reads `openapi.gen.yaml` and domain
  constants can assert `request ≤ domain ≤ response` in CI.
- Ownership is cleanly separated by role — DB capacity → DBA / data owner, wire contract →
  developer, business rule → domain expert — so the system can be built without merging all
  three sets of requirements into one place (the owners coordinate, but do not have to
  co-locate their constraints).

### Negative Consequences

- Developers must track three separate places for what looks like "one number"; this
  overhead is unavoidable given the independent ownership.
- When the OpenAPI request constraint is deliberately tighter than the domain, reviewers
  may question why they differ — the divergence must be justified explicitly.

## Alternatives Considered

### Keep OpenAPI and domain constraints identical

Simpler to reason about. Rejected: conflates two independently-owned concerns, hides the
direction invariant, and produces incorrect results when the API wire contract is versioned
or adjusted for consumer compatibility independently of the domain.

### Let the domain own all constraints (derive OpenAPI from domain constants)

Would guarantee request ⊆ domain by construction. Rejected: violates the
[ADR-0009](0009-openapi-first.md) OpenAPI-first principle (the spec is authored first, not
derived from code) and couples the domain package to HTTP-layer tooling.

## Notes

- Full rationale and worked example (`firstName` maxLength 50 / 100 / 100 / 100 across
  request / domain / DB / response): [`openapi/boundary-ownership.md`](../../openapi/boundary-ownership.md).
- Domain constants live at `internal/domain/<aggregate>/constant.go`.
- The request validation that enforces the wire-contract side is described in
  [ADR-0013](0013-spec-driven-request-validation.md).
- Builds on [ADR-0014](0014-validation-value-authority.md): the domain is the authority for
  business validity, and *strictness* (value-set tightness) is a separate axis from
  *authority*. This ADR's direction invariant (`request ⊆ domain ⊆ response`) is the
  tightness view and does not override domain authority.
- Parent decision: [ADR-0009](0009-openapi-first.md) (OpenAPI-first).
