---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [http, framework]
---

# ADR-0019: Adopt Echo as the HTTP framework

## Status

accepted

## Context

This repository needs an HTTP framework for routing and middleware that is lightweight,
predictable, and imposes low abstraction over the standard library, consistent with the
design goals of maintainability and structural safety over raw feature richness.

Echo v4 receives security and bug fixes only, so a repository that gates dependency
vulnerabilities in CI cannot keep an end-of-life HTTP framework. Echo v5 is the maintained
line, and the surrounding ecosystem (OpenAPI validation, OpenTelemetry instrumentation)
supplies v5 counterparts through separate modules.

## Decision

Adopt **Echo v5** (`labstack/echo/v5`) for HTTP routing and middleware, together with the
two ecosystem modules the HTTP stack depends on:

| Concern | Module |
| --- | --- |
| Routing / middleware | `github.com/labstack/echo/v5` |
| OpenAPI request validation | `github.com/oapi-codegen/echo-v5-middleware` |
| OpenTelemetry instrumentation | `github.com/labstack/echo-opentelemetry` |

`echo-opentelemetry` is pre-1.0 but is listed in Echo's own "official middleware
repositories" and is maintained by Echo maintainers; the OTel contrib instrumentation for
Echo is v4-only and its maintainers declined v5 support, pointing at this module instead.
The version number reflects a conservative first numbering of a new package rather than an
unstable API, and its dependency set matches the OTel version this repository already pins.

**Server lifecycle is owned by the application, not by the framework.** Echo v5 removed the
server fields and start/stop methods on `Echo` and concentrated them in `StartConfig`,
whose model is "block until the context is cancelled, then shut down within its own
graceful timeout". That does not compose with a DI container that separates start and stop
hooks, so the application constructs its own `http.Server` with `Echo` as the handler. The
listener is opened in the start hook (so a bind failure fails fast and aborts startup) and
`Shutdown` is driven by the stop hook's context. `http.Server` is also where the
per-request timeouts live now that `Echo.Server` is gone.

**Error detail on spans follows semantic conventions.** The v4 instrumentation put the
error text on the span as a non-standard `echo.error` attribute; the v5 instrumentation
records `error.type` instead, and the error text still reaches the span status description
for 5xx and the trace-correlated application logs. This repository does not add a hook to
restore the old attribute.

## Consequences

### Positive Consequences

- Simple, clear middleware structure that the priority-ordered middleware chain builds on.
- Low abstraction over the standard `net/http` model.
- Good performance for a general-purpose backend.
- The client address on spans is derived from the configured `IPExtractor` rather than the
  raw forwarded header, so a spoofed header no longer reaches telemetry.

### Negative Consequences

- A framework dependency in the controller layer; handlers couple to `*echo.Context`, kept
  contained at the controller boundary (never leaking inward).
- The OpenTelemetry instrumentation is a pre-1.0 module with a small maintainer base. It is
  reachable only through one middleware slot, and disabling tracing drops the whole
  integration back to a passthrough middleware, so a failure there degrades telemetry
  without taking down the server.

## Alternatives Considered

### Gin

A very similar framework, but Echo's middleware structure is slightly simpler.

### Chi

An excellent router, but Echo provides a more complete set of framework features
out of the box.

### Wrapping the framework with `otelhttp` instead of Echo-specific instrumentation

`otelhttp` is already a dependency and would sit outside Echo as a plain `http.Handler`,
but it cannot see the route template, so span names degrade to the raw path and the
metric cardinality follows. Kept as an escape hatch, not as the default.

## Notes

- The middleware chain design (priority-ordered, data-driven) and the HTTP-stack layering are recorded separately (see the HTTP-layer ADRs in [the ADR log](README.md)).
- Migrated from `docs/decisions.md` (§ "Why Echo").
