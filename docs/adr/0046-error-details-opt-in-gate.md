---
status: accepted
date: 2026-07-17
deciders: [maintainers]
tags: [errors, architecture, api, security]
---

# ADR-0046: Opt-in gate for error-response details via schema split

## Status

accepted

Refines [ADR-0045](0045-error-metadata-code-message-details.md).

## Context

[ADR-0045](0045-error-metadata-code-message-details.md) introduced `apperror.Meta`, letting
an error-raising site attach `details` (public-safe identifiers such as invalid field names)
that surface in the error response. As delivered, that exposure was **fail-open**: any error
whose chain carried `Meta` details would render them on **every** endpoint, because a single
shared `ErrorResponse` schema (with an optional `details` field) backed all error responses.

This is a leakage risk. The concrete trigger was entity reconstruction from a stored row
(`rowToUser` → `user.New`): a data-integrity failure could reach the client as a `422` whose
`details` named internal fields, on an endpoint that never intended to expose them. Even after
that specific path was hardened, the structural default remained "details leak unless something
strips them" — the wrong direction for a security-relevant field.

We want the exposure to be **opt-in per endpoint**, with the decision expressed in the API
contract (so the contract, the generated client types, and the runtime behavior agree), and
enforced **fail-closed** at the transport edge.

## Decision

Split the error-response envelope into two schemas and gate details at the edge.

1. **Contract (SSOT).** `openapi/components/schemas/ErrorResponse.yaml` becomes the base
   envelope (`code` / `message` / `requestId`, **no** `details`). A new
   `ErrorResponseWithDetails.yaml` adds `details`. Only responses that intentionally expose
   details reference the `WithDetails` schema (currently the `422` of `PostUsers` /
   `PutUsersDetail` / `PatchUsersDetail`, via `errors/UnprocessableEntity422.yaml`). Every
   other error response keeps referencing the base schema. **Which operations reference
   `ErrorResponseWithDetails` is the opt-in switch**, and it lives entirely in the OpenAPI spec.

2. **Builder type.** The internal type-generation endpoint (`GenerateErrorSchema`) references
   `ErrorResponseWithDetails`, so the response builder's generated type is the superset
   (`gen.ErrorResponseWithDetails`). `HTTPErrorResponse` embeds that superset — the single
   builder DTO always *has* a `Details` field. Whether `details` reaches the wire is decided
   downstream, not by the builder (which stays request-agnostic, per ADR-0045).

3. **Fail-closed gate at the edge.** A `DetailPolicy` (built once at startup from the spec via
   `gorillamux.NewRouter` + a precomputed `operationId → exposes-details` map) is injected into
   the `errorhandler`. On the error path, if the response carries `details`, the handler
   resolves the request's operation and drops `details` from the **client wire** unless that
   operation opted in. The `resp` object (and therefore the logs) keep the full details.

This mirrors the existing `requestId` idiom: the request-agnostic `error/response` package
produces the skeleton (leaves `requestId` empty; attaches whatever `details` the error holds),
and the request-aware `errorhandler` finalizes it (fills `requestId`; strips `details` when the
endpoint is not opted in). The gate is host-agnostic — the policy router is built from a
servers-stripped copy of the spec so proxied / test hosts still resolve by path + method.

## Consequences

### Positive Consequences

- Detail exposure is **fail-closed**: a new endpoint returns no `details` until it declares
  `ErrorResponseWithDetails`, so forgetting the declaration cannot leak.
- The opt-in decision is a single machine-readable fact in the OpenAPI contract; generated
  client types and runtime behavior agree, and the split is auditable.
- Logs retain full `details` for debugging even when the wire omits them.
- No per-endpoint middleware and no hot-path cost: the router runs only on the error path.

### Negative Consequences

- "When do details appear" now spans two packages (`error/response` attaches, `errorhandler`
  gates) — the same two-place cost the `requestId` idiom already has.
- The `ErrorResponseWithDetails` struct is duplicated across the handler `gen` packages that
  declare a `422` (normal oapi-codegen per-package output).
- An endpoint that *should* expose details but forgets the schema declaration silently returns
  none. The schema is the only opt-in switch; this is documented in `docs/rules.md` and the
  `apperror` / `error/response` READMEs.

## Alternatives Considered

### Vendor extension (`x-expose-error-details`)

Mark opt-in with a vendor extension on the operation/response instead of a schema split.
Rejected: the schema split makes the contract itself (and the generated client types) tell the
truth about which responses carry `details`; a vendor extension is a side-channel that clients
do not see.

### Gate in the `error/response` builder

Pass an `exposeDetails` flag into `NewHTTPErrorFromAppError`. Rejected: the opt-in decision is
request-aware (it needs the matched operation), while the builder is deliberately
request-agnostic (pure `apperror → shape` mapping). Edge-time, request-scoped finalization is
the `errorhandler`'s existing role (`requestId`, logging, commit checks).

### Per-endpoint opt-in middleware

A middleware on each detail-exposing route that sets a flag. Rejected: it duplicates the truth
already in the OpenAPI contract and drifts from it; it also adds hot-path work on every request.

## Notes

- Source: `internal/apperror/README.md` (Error Metadata section),
  `internal/controller/error/response/README.md`,
  `internal/controller/httpstack/errorhandler/README.md`.
- The base `ErrorResponse` Go type is still generated in handler `gen` packages as response
  aliases (`BadRequest400 = ErrorResponse`, etc.); it is documentation-grade output not used by
  hand-written code. The builder package (`error/response/gen`) generates only the superset.
