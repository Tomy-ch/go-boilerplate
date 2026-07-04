---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [contract, api, openapi]
---

# ADR-0009: Define the API contract OpenAPI-first

## Status

accepted

## Context

The template needs API contracts to be clear, type-safe, and agreed on **before**
implementation, so that request/response shapes are unambiguous and can stay consistent with
frontend consumers and generated documentation. Deriving the contract from handwritten
handler code instead leaves the contract implicit and prone to drift.

## Decision

Define API specifications **OpenAPI-first**: the OpenAPI document is authored first, and
server code is generated from it with `oapi-codegen`. The spec is the single source of truth
for the wire contract; handlers implement the generated interface rather than defining the
shape.

## Consequences

### Positive Consequences

- Clear, explicit API contracts fixed before implementation.
- Type-safe request/response structures generated from the spec.
- Consistency with frontend consumers working from the same document.
- API documentation is generated, not maintained by hand.

### Negative Consequences

- Adding or changing an endpoint requires editing the spec first and regenerating — a
  heavier step than editing a handler directly.
- Contributors must learn the OpenAPI authoring + generation workflow.

## Alternatives Considered

### Code-first API (generate OpenAPI from code)

Rejected: generating the spec from handler code tends to leave the API contract unclear and
lets the implementation, not an agreed contract, drive the shape.

### GraphQL-first

Rejected: GraphQL is powerful but introduces high complexity for a general-purpose backend
template where REST resources are the expected shape.

## Notes

- Enforced by the OpenAPI-first rules in [`docs/rules.md`](../rules.md#openapi-first) (spec defined before implementation; generated code not edited by hand), which are the day-to-day *consequences* of this decision.
- Migrated from `docs/decisions.md` (§ "Why OpenAPI-first").
- The spec build pipeline (Redocly modular split → bundle → generate) and per-tag generation are recorded separately (see the API-contract ADRs in [the ADR log](README.md)).
