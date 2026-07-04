---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [contract, openapi, security]
---

# ADR-0012: Validate requests and enforce auth from the spec at runtime; do not validate responses

## Status

accepted

## Context

[ADR-0009](0009-openapi-first.md) makes the OpenAPI spec the single source of truth for
the wire contract. To have the spec actually protect the server at runtime — not just
document it — request validation and security-scheme enforcement must run automatically for
every inbound request, derived from the same spec document. At the same time, validating
outbound responses against the spec at runtime is expensive and, if the spec and code are
kept in sync through code generation (see [ADR-0011](0011-oapi-codegen-strict-server.md)),
unnecessary.

An additional constraint: ops paths (`/health`, `/metrics`, `/ready`, `/healthz`,
`/version`) must not pass through the OpenAPI validation pipeline because they are not
described in the OpenAPI spec.

## Decision

Wire `oapimw.OapiRequestValidatorWithOptions` from the `oapi-codegen/echo-middleware`
package as an Echo middleware, passing the parsed spec and an `openapi3filter.AuthenticationFunc`.
Before the validator runs, an authn context slot is injected so the `AuthenticationFunc`
can write authentication results into the request context.

```go
func Middleware(
    spec *openapi3.T,
    skipper echomw.Skipper,
    authFunc openapi3filter.AuthenticationFunc,
) echo.MiddlewareFunc {
    oapiValidator := oapimw.OapiRequestValidatorWithOptions(spec, &oapimw.Options{
        SilenceServersWarning: true,
        Skipper:               skipper,
        Options: openapi3filter.Options{
            AuthenticationFunc: authFunc,
        },
    })
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            req := c.Request()
            req = req.WithContext(ctxhelper.WithAuthn(req.Context()))
            c.SetRequest(req)
            return oapiValidator(next)(c)
        }
    }
}
```

The middleware is configured with a skipper that bypasses validation for ops paths
(`internal/controller/httpstack/oapi/skipper`). The `security:` declarations in the spec
drive the `AuthenticationFunc`: the function only fires for operations that declare a
security requirement, so public endpoints (e.g. `GET /health`) are never challenged for
authentication.

**Responses are not validated at runtime.** The response contract is trusted by
construction: because handler code is generated from the same spec (ADR-0011), a
well-typed response cannot violate the spec.

## Consequences

### Positive Consequences

- Request bodies, query parameters, and path parameters are validated against the spec
  before the handler is invoked; invalid input never reaches business logic.
- Security requirements declared in the spec (`security:` blocks) are enforced
  automatically — a handler cannot be reached without passing the configured
  `AuthenticationFunc`.
- Ops paths are excluded without any per-handler opt-out; the skipper logic is
  centralised.
- No runtime overhead for response validation.

### Negative Consequences

- If a response value is produced outside the HTTP path (e.g. a seeded row with a value
  that violates the response schema), the violation is invisible at the server and only
  surfaces on the client side. See [`openapi/boundary-ownership.md`](../../openapi/boundary-ownership.md)
  for the direction invariant that guards against this.
- The middleware must be kept wired with the same bundled spec used for code generation;
  if the spec file loaded at startup drifts from the generated code, the middleware and
  handlers may disagree silently.

## Alternatives Considered

### Per-handler manual validation

Each handler calls its own binding and validation logic. Rejected: this is the status quo
that code generation is designed to replace — it is tedious, error-prone, and easy to
omit.

### Runtime response validation

Validate outbound response bodies against the spec. Rejected: significant latency cost on
every response, and the strict-server generated code already guarantees the response type
matches the spec at compile time.

## Notes

- Middleware implementation: [`internal/controller/httpstack/oapi/oapi.go`](../../internal/controller/httpstack/oapi/oapi.go).
- Ops-path skipper: [`internal/controller/httpstack/oapi/skipper/skipper.go`](../../internal/controller/httpstack/oapi/skipper/skipper.go).
- The `/metrics` auth exception (skipped from this pipeline; protected by a separate
  BasicAuth middleware) is recorded in [ADR-0014](0014-metrics-endpoint-auth-exception.md).
- Security and boundary notes: [`openapi/README.md`](../../openapi/README.md) (§ Security) and [`openapi/boundary-ownership.md`](../../openapi/boundary-ownership.md).
- Parent decision: [ADR-0009](0009-openapi-first.md) (OpenAPI-first).
