---
status: accepted
date: 2026-08-14
deciders: [maintainers]
tags: [contract, http, security]
---

# ADR-0019: Optional authentication is allowed, and a failed authentication still denies the request

## Status

accepted

## Context

[ADR-0015](0015-spec-driven-request-validation.md) establishes that the OpenAPI middleware
enforces the security requirements declared in the spec, and [ADR-0018](0018-metrics-endpoint-auth-exception.md)
records the one endpoint that sits outside that pipeline. Both assume an operation is either
protected or public.

Some resources are neither. A cart belongs to a signed-in user when there is one and to an
anonymous session otherwise, and the same operation must serve both. OpenAPI expresses this as
several security requirements where one is empty:

```yaml
security:
  - BearerAuth: []
  - {}
```

**This form is not safe as it stands.** `kin-openapi`'s `ValidateSecurityRequirements` tries the
requirements in order, collects each failure, and moves on; an empty requirement names no scheme,
so it always succeeds. The failure of the preceding `BearerAuth` is discarded, and there is no
sentinel error that short-circuits the loop. An expired, tampered, or unknown-`kid` token
therefore arrives at the handler indistinguishable from no token at all.

The same path swallows something heavier. When identity resolution fails for an infrastructure
reason, the authentication function returns a 503 or 500, and the OR evaluation discards that too
— so a database outage would surface as an anonymous success rather than as an outage.

Both outcomes contradict decisions this repository has already taken: `docs/design/auth.md`
requires that every verification failure normalize to a denial and never to a default-allow, and
`docs/design/security.md` requires deny-by-default at every boundary, with no state in which
forgetting something opens it.

## Decision

**Allow an operation to declare authentication optional, and make a presented-but-rejected
credential deny the request anyway.**

The authentication function records its failure into the same request-context slot it already
uses to carry a successful `Authn` (`ctxhelper.SetAuthnFailure`). The oapi middleware re-reads
that slot after validation and before the handler, and returns the recorded error if there is
one:

```go
return oapiValidator(failClosed(next))(c)
```

The slot survives what the return value does not, because the validator discards only the error
it is handed.

Absence of a credential is not a failure and is never recorded, so an operation that admits
anonymous callers keeps admitting them. The resulting behavior, by declaration:

| `security` | Credential | Outcome |
| --- | --- | --- |
| `[BearerAuth]` | invalid | 401 (the validator returns before `failClosed` is reached) |
| `[]` | anything | unauthenticated, as before |
| `[{BearerAuth}, {}]` | absent | anonymous |
| `[{BearerAuth}, {}]` | invalid | **401** |
| `[{BearerAuth}, {}]` | valid, identity resolution unavailable | **503 / 500** |

An operation that uses the optional form is declaring an exception to "every endpoint is either
protected or public", so it is named and kept narrow the same way `/metrics` is: it is listed in
the `publicOperations` allow-list of
`internal/controller/httpstack/oapi/validator/security_declaration_test.go` with a reason, which
keeps the set greppable and reviewable.

`BearerAuth` must be written before the empty requirement. Evaluation is ordered, and the empty
requirement always succeeds, so anything after it is never reached.

## Consequences

### Positive Consequences

- One resource can serve signed-in and anonymous callers without splitting into two endpoints,
  and without the URL revealing which one the caller is.
- The fail-closed rule in `docs/design/auth.md` holds for every declaration form, so reading a
  `security:` block is enough to know what happens on failure.
- An infrastructure failure during identity resolution surfaces as an outage rather than as an
  empty-but-successful anonymous response — a bug that was latent and unobservable only because
  no operation used the optional form yet.
- No new middleware and no re-ordering of the chain: the change is confined to the existing
  authentication function and the oapi middleware.

### Negative Consequences

- The guarantee lives in `failClosed`, not in the spec. A reader of the OpenAPI document alone
  sees "authentication optional" and cannot tell that a rejected credential is refused; the
  behavior is discoverable only from this ADR and the middleware.
- The optional form is easy to write and its unsafe reading is the intuitive one, so the
  allow-list entry and its reason are what keep a future operation from adopting it casually.
- Anonymous access still has to be authorized. Making authentication optional does not make the
  resource public — an operation using this form must still decide what an anonymous subject may
  do.

## Alternatives Considered

### Split each such operation into a protected one and a public one

`/v1/carts/me` and `/v1/carts/guest`, each with a single security form. The conflict disappears
because the optional form is never written. Rejected: it turns five endpoints into eight or nine,
and it puts the caller's authentication state into the URL, so a client must know which one it
is before choosing a path. Optional authentication can be made safe, so there is no reason to pay
for it in API shape.

### Short-circuit inside the authentication function's return value

Return a distinguished error that stops requirement evaluation. Rejected after reading the
implementation: `ValidateSecurityRequirements` has no hook for it. Every error is collected and
evaluation continues to the next requirement.

### Require authentication everywhere and drop anonymous carts

Simplest, and the fail-closed question never arises. Rejected: it removes the ownership
transition on login, authorization of an anonymous subject, and the merge rules — three of the
four things the cart sample exists to demonstrate.

## Notes

- Fail-closed stage: [`internal/controller/httpstack/oapi/oapi.go`](../../internal/controller/httpstack/oapi/oapi.go).
- Failure recording: [`internal/controller/httpstack/oapi/auth/auth.go`](../../internal/controller/httpstack/oapi/auth/auth.go).
- The slot itself: [`internal/controller/ctxhelper/authn.go`](../../internal/controller/ctxhelper/authn.go).
- The behavior is fixed by a middleware test driving a synthetic spec that declares all three
  security forms; no operation in the shipped spec uses the optional form yet.
- Related decision: [ADR-0015](0015-spec-driven-request-validation.md) (spec-driven request validation).
- Related decision: [ADR-0018](0018-metrics-endpoint-auth-exception.md) (the other named authentication exception).
- Posture this upholds: `docs/design/auth.md` (fail-closed), `docs/design/security.md` (deny by default).
