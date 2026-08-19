---
status: accepted
date: 2026-07-05
deciders: [maintainers]
tags: [contract, domain, validation]
---

# ADR-0017: Designate the domain layer as the sole authority for business-validity rules

## Status

accepted

## Context

Validation constraints appear in multiple locations across a system: an OpenAPI schema
`maxLength`, a domain constant, a database column definition, and potentially a client-side
rule. When these numbers coincide — which they often do by convenience — it is tempting to
treat them as equivalent. When they diverge, it becomes unclear which layer should be
consulted as the source of truth.

Two conceptually distinct axes are in play:

**Strictness** describes how tight a constraint's value set is. A rule that allows values up
to 50 is stricter than one that allows up to 100. The tightest constraint admits the fewest
values.

**Authority** (or authenticity) describes which layer has the right to define what the
business considers valid. Authority is a role assignment — it answers "whose decision is
this?" not "which number is smaller?"

These axes are independent. The tightest constraint is not necessarily the most
authoritative one. For example, an OpenAPI request constraint of `maxLength: 50` may be
tighter than a domain constant of `maxLength: 100`, yet the domain constant is the
authoritative business rule and the request constraint is a wire-contract decision owned by
the API developer.

Without an explicit authority model, teams fall into several failure modes:

- Assuming the strictest layer is the business rule ("the API says 50, so the domain limit
  must be 50").
- Assuming all layers must agree ("the numbers differ — which one is right?").
- Changing a wire-contract value and believing the domain rule changed as a consequence.

The domain layer's purpose — holding business logic in a form that is framework-free and
independent of infrastructure — establishes it as the natural locus of business-validity
authority. Domain experts, not API developers or DBAs, are the role that defines what the
business considers valid.

## Decision

The **domain layer** is designated as the sole authority for **business-validity
decisions**. The domain constant (e.g., `internal/domain/<aggregate>/constant.go`) is the
single source of truth for what the business considers valid.

Strictness and authority are treated as **independent axes**:

| Layer | Role | Authority over |
| --- | --- | --- |
| OpenAPI (request) | API developer / wire contract | What the HTTP API accepts from callers |
| Domain | Domain expert / business rule | What the business considers valid — sole business-validity authority |
| Database | DBA / data owner | Physical storage capacity |
| OpenAPI (response) | API developer / wire contract | What the HTTP API promises to callers |

A layer may be **stricter** than the domain without thereby becoming the business-validity
authority. The strictest constraint governs what reaches the domain at runtime (the tightest
gate wins at the wire), but it does not redefine the domain's validity boundary.

Corollary: when layers disagree on a boundary value, the domain constant is correct by
definition for business validity. Divergence in other layers reflects each layer's own
independent concern, not an error in the domain.

## Consequences

### Positive Consequences

- Domain business rules remain stable and legible independently of HTTP API versioning,
  consumer-compatibility decisions, or DBA storage choices.
- The failure modes described in Context — assuming the strictest layer is authoritative, or
  that all layers must agree — are structurally ruled out.
- Each layer's owner (API developer, domain expert, DBA) can evolve their constraint for
  their own reasons without incorrectly implying a change to business validity.
- This authority model is the prerequisite for the direction invariant in ADR-0018
  (境界値の所有権 / boundary value ownership): the invariant
  `request ⊆ domain ⊆ response` presupposes that the domain is the authority and that
  other layers must respect its boundaries, not the other way around.

### Negative Consequences

- The same field can have up to four numbers (request, domain, DB, response) that are
  explicitly permitted to differ. This overhead is unavoidable; it is the cost of treating
  independent concerns as independent.
- Reviewers who encounter a divergence (e.g., request `maxLength: 50` vs domain `100`)
  must understand this model to avoid filing false-alarm review comments.

## Alternatives Considered

### Treat the strictest constraint as authoritative

Simple: look at whichever number is tightest and treat it as the truth. Rejected: it
conflates strictness (a value-set property) with authority (a role assignment). A defensive
wire limit tighter than the domain does not redefine business validity; mistaking it for
the authority causes domain rules to drift in response to wire-contract decisions.

### Treat the database column limit as authoritative (DBA owns validity)

The DB constraint is often the physical ceiling. Rejected: physical storage capacity
answers "what the column can hold," not "what the business considers valid." A domain might
legitimately store only values up to 100 while the column is 255 — the extra space is
buffer, not domain permission. Database ownership is about data integrity and storage
capacity, not business-validity semantics.

### Require all layers to agree (single number everywhere)

Eliminates ambiguity about which number is authoritative. Rejected: forces coupling across
independently-evolving concerns. A wire contract may need to tighten for consumer
compatibility without changing the domain rule; a DB column may widen for future capacity
without changing business validity. Requiring agreement conflates three separate
change-reasons into one.

## Notes

- Domain constants live at `internal/domain/<aggregate>/constant.go`.
- The downstream constraint relationship (direction invariant: `request ⊆ domain ⊆
  response`) is specified in ADR-0018 (境界値の所有権 / boundary value ownership). This
  ADR establishes the authority model that makes that invariant coherent; ADR-0018
  specifies the containment direction.
- The `openapi/boundary-ownership.md` guide illustrates the authority separation with the
  `firstName` `maxLength` worked example (request 50 / domain 100 / DB `VARCHAR(100)` /
  response 100).
