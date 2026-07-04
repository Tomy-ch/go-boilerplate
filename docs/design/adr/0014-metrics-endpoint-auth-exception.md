---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [contract, http, observability, security]
---

# ADR-0014: /metrics is an auth exception — outside OpenAPI validation, protected by a separate BasicAuth middleware

## Status

accepted

## Context

[ADR-0012](0012-spec-driven-request-validation.md) establishes that the OpenAPI middleware
validates every inbound request and enforces security requirements declared in the spec.
The Prometheus metrics endpoint (`GET /metrics`) exposes operational data and must be
protected against unauthenticated access. However, `/metrics` is an ops path that is not
part of the OpenAPI API contract: it is not versioned as a public resource, it is not
consumed by API clients, and its response format (Prometheus text exposition) is not
described by the OpenAPI spec. Routing it through the OpenAPI validation pipeline would
require adding it to the spec and generating a handler interface for it, which is
architecturally wrong.

## Decision

Register `/metrics` **outside the OpenAPI validation pipeline** and protect it with a
dedicated Echo `BasicAuth` middleware.

The oapi middleware skipper classifies `/metrics` as an ops path and bypasses spec
validation entirely:

```go
// internal/controller/httpstack/oapi/skipper/skipper.go
func New() echomw.Skipper {
    return func(c echo.Context) bool {
        return ops.IsOpsPath(c.Request().URL.Path)
    }
}
```

The route is registered with `echomw.BasicAuth(validator)` applied inline:

```go
// internal/controller/handler/metrics/metrics_handler.go
func BindHandler(e *echo.Echo, bav echomw.BasicAuthValidator) {
    e.GET("/metrics",
        echo.WrapHandler(promhttp.Handler()),
        echomw.BasicAuth(bav),
    )
}
```

The BasicAuth validator uses constant-time comparison to resist timing attacks
(`internal/controller/httpstack/basicauth/basic.go`). Credentials are read from
`MetricsConfig`.

Any `security:` annotation for `/metrics` in the OpenAPI spec is **documentation only** —
it does not drive runtime enforcement. The actual authentication is the BasicAuth
middleware registered on the route.

## Consequences

### Positive Consequences

- `/metrics` is protected without being forced into the OpenAPI contract, keeping the spec
  limited to API resources that consumers depend on.
- Credentials for the metrics endpoint are independently configurable via `MetricsConfig`,
  separate from JWT/BearerAuth used for API endpoints.
- The constant-time comparison in the validator prevents credential leakage via timing
  analysis.

### Negative Consequences

- The `/metrics` security mechanism lives outside the spec, so it is not discoverable from
  the OpenAPI document alone — developers must know to check the handler registration and
  `MetricsConfig`.
- If the OpenAPI spec includes a `security:` annotation for `/metrics`, it is silently
  inoperative at runtime; readers may incorrectly assume the oapi middleware enforces it.

## Alternatives Considered

### Include /metrics in the OpenAPI spec and generate a handler interface for it

Would make the security enforcement consistent with other endpoints. Rejected: the metrics
endpoint is an operational concern, not an API resource — adding it to the spec pollutes
the consumer contract and requires generating types for a Prometheus response format that
oapi-codegen cannot model.

### Skip auth on /metrics and rely on network-level access control

Simpler operationally. Rejected: the template should provide a usable auth mechanism out
of the box; leaving the endpoint open shifts the security responsibility entirely to
infrastructure configuration.

## Notes

- Skipper implementation: [`internal/controller/httpstack/oapi/skipper/skipper.go`](../../../internal/controller/httpstack/oapi/skipper/skipper.go).
- Route registration and BasicAuth wiring: [`internal/controller/handler/metrics/metrics_handler.go`](../../../internal/controller/handler/metrics/metrics_handler.go).
- Validator: [`internal/controller/httpstack/basicauth/basic.go`](../../../internal/controller/httpstack/basicauth/basic.go).
- Ops-path classification: `internal/controller/httpstack/ops/paths.go`.
- Security note in [`openapi/README.md`](../../../openapi/README.md) (§ Security): the `/metrics` `security:` declaration is documentation-only.
- Related decision: [ADR-0012](0012-spec-driven-request-validation.md) (spec-driven request validation — this ADR is its companion exception record).
- Parent decision: [ADR-0009](0009-openapi-first.md) (OpenAPI-first).
