---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [errors, architecture]
---

# ADR-0037: Protocol-agnostic aggregated error classification (apperror)

## Status

accepted

## Context

Without a shared error taxonomy, each layer independently encodes its failures. Domain
and usecase code may return raw database errors, sentinel errors carrying HTTP status
codes, or unclassified `error` values. Controllers must then special-case every
possible error type, coupling the transport layer to the internals of every layer below
it.

In an onion architecture (see [ADR-0002](0002-onion-architecture.md)) the domain and
usecase layers must remain independent of protocols. An error type that carries an HTTP
status or a gRPC code violates that independence; swapping the transport layer would
require touching error-producing code deep in the domain.

Additionally, infrastructure errors (database errors, external API failures) use
library-specific types that inner layers should not need to understand.

## Decision

The `internal/apperror` package defines an **application-wide error taxonomy**
independent of any protocol. It provides sentinel errors such as `ErrNotFound`,
`ErrConflict`, `ErrValidation`, and `ErrInternal` that classify failures in
application-domain terms without reference to HTTP status codes or gRPC codes.

All layers — domain, usecase, infrastructure, and controller — may reference
`apperror`. Infrastructure repositories translate library-specific errors (e.g.,
PostgreSQL constraint violations) into the nearest `apperror` sentinel before
returning. Upper layers wrap their own errors with the appropriate sentinel using
`pkg/xerrors`.

Protocol mapping happens exclusively at the **edge**: the controller's error-handler
middleware reads the sentinel via `xerrors.Is` and converts it to an HTTP status and
response body. The domain and usecase layers never know which protocol the caller uses.

A second, orthogonal set of sentinels (`ErrRetryable`, `ErrPermanent`, `ErrFatal`) is
used by the worker engine to classify handler errors for retry / dead-letter / stop
decisions. These worker sentinels have no HTTP mapping and are intentionally excluded
from the HTTP classification helper `IsAppError`.

## Consequences

### Positive Consequences

- Domain and usecase code expresses failures in application-domain vocabulary; no
  transport knowledge leaks into inner layers.
- Switching or adding a transport (HTTP to gRPC, adding a CLI controller) requires only
  a new edge mapping; the error-producing code is unchanged.
- Infrastructure errors are translated once at the repository boundary; upper layers
  are not exposed to library-specific error types.
- `xerrors.Is`-based classification traverses wrapped error chains, so the original
  error context is preserved for logging and tracing.

### Negative Consequences

- Every repository and infrastructure adapter must explicitly translate external errors
  to `apperror` sentinels; untranslated errors fall through as `ErrInternal`.
- Adding a new sentinel category requires cross-cutting review (it must make sense
  across multiple use cases and have a clear HTTP/worker mapping).

## Alternatives Considered

### HTTP-status-carrying error types

Return errors that embed an HTTP status code from domain or usecase code. Rejected:
couples inner layers to HTTP and prevents reuse in non-HTTP controllers (CLI, workers).

### Unclassified errors everywhere

Let each layer return raw errors and classify them only in the controller. Rejected:
without a shared taxonomy, controller error-handling logic must pattern-match on every
library-specific error type from every dependency, which is unmaintainable.

## Notes

- Source: `internal/apperror/README.md` (Basic Policy and Mapping Table sections).
- Error-handling rules (wrapping, `xerrors.Join` vs `xerrors.Wrap`, redact rule):
  [`docs/rules.md`](../rules.md) — Error Handling Rules section.
- Protocol mapping via `xerrors.Is` follows the wrapping policy in `pkg/xerrors`.
