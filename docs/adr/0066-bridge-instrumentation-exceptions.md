---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [dependencies, observability, exception]
---

# ADR-0066: Bridge / instrumentation libraries as bounded SRP exceptions

## Status

accepted

## Context

The library-selection policy ([ADR-0065](0065-library-selection-policy.md)) requires one
responsibility bound to a single upstream. Instrumentation and bridge libraries inherently
stand between **two independently-versioned upstreams** (a framework/library × OpenTelemetry),
so they cannot satisfy that criterion — yet re-implementing their glue by hand would couple
tightly to each target's internal lifecycle and increase maintenance debt.

## Decision

Admit bridge / instrumentation libraries as **explicit, individually-justified exceptions**
to the single-responsibility policy, on these common grounds:

- Hand-rolling the glue would couple tightly to the target's internal lifecycle
  (Echo / pgx / zap) and raise maintenance debt rather than reduce it.
- Each is **small and Apache-2.0 licensed**, so worst case it can be vendored / forked; the
  fork cost is bounded to the production line counts recorded per library.
- The otel-contrib ones (`otelecho`, `otelhttp`, `otelzap`) ship on **otel-contrib's monthly
  release train**, kept lockstep with OpenTelemetry; `otelpgx` is a third-party package
  (`github.com/exaring/otelpgx`, Apache-2.0) tracking pgx + OpenTelemetry independently. In
  every case the only residual drift surface is the framework-side interface, and those
  interfaces (`echo.MiddlewareFunc` / `net/http.RoundTripper` / pgx `QueryTracer` /
  `zapcore.Core`) are stable v1.

The currently-accepted exceptions are `otelecho` (root server span), `otelhttp` (outbound
HTTP client spans), `otelpgx` (SQL query spans), and `otelzap` (zap → OTel log bridge).

## Consequences

### Positive Consequences

- Observability instrumentation is obtained without hand-maintaining fragile glue.
- Each exception is bounded: license, upstream cadence, and worst-case fork cost are known.

### Negative Consequences

- These dependencies straddle two upstreams, so their drift surface is larger than a
  single-upstream library's — accepted knowingly and reviewed per library.

## Alternatives Considered

### Re-implement the glue by hand

Rejected: couples tightly to each target's internal lifecycle and increases maintenance debt
more than the bridge dependency does.

### Refuse the exception (drop instrumentation)

Rejected: would forfeit standard, low-cost observability instrumentation for no proportionate
gain, given the bounded worst-case fork cost.

## Notes

- Parent policy: [ADR-0065](0065-library-selection-policy.md). Related gating decision: [ADR-0059](0059-config-driven-observability-gating.md).
- Per-library versions and line counts are an inventory snapshot (as investigated 2026-06-25) and belong with the dependency reference (`docs/reference/dependencies.md`, Phase 5), not in this immutable record.
- Migrated from `docs/decisions.md` (§ "Exceptions: instrumentation / bridge libraries").
