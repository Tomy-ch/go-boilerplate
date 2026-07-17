---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [http, framework]
---

# ADR-0017: Adopt Echo as the HTTP framework

## Status

accepted

## Context

The template needs an HTTP framework for routing and middleware that is lightweight,
predictable, and imposes low abstraction over the standard library, consistent with the
design goals of maintainability and structural safety over raw feature richness.

## Decision

Adopt **Echo** (`labstack/echo/v4`) for HTTP routing and middleware.

## Consequences

### Positive Consequences

- Simple, clear middleware structure that the priority-ordered middleware chain builds on.
- Low abstraction over the standard `net/http` model.
- Good performance for a general-purpose backend.

### Negative Consequences

- A framework dependency in the controller layer; handlers couple to `echo.Context`, kept
  contained at the controller boundary (never leaking inward).

## Alternatives Considered

### Gin

A very similar framework, but Echo's middleware structure is slightly simpler.

### Chi

An excellent router, but Echo provides a more complete set of framework features
out of the box.

## Notes

- The middleware chain design (priority-ordered, data-driven) and the HTTP-stack layering are recorded separately (see the HTTP-layer ADRs in [the ADR log](README.md)).
- Migrated from `docs/decisions.md` (§ "Why Echo").
