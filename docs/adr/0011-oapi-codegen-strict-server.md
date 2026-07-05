---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [contract, openapi, codegen]
---

# ADR-0011: Generate per tag/handler with oapi-codegen in strict-server mode

## Status

accepted

## Context

[ADR-0009](0009-openapi-first.md) requires that server code is generated from the OpenAPI
spec rather than written by hand. The question is *how* to scope that generation: a single
global generation produces one large interface that all handlers must implement together,
which creates tight coupling between otherwise-independent handler packages. Additionally,
the default oapi-codegen server mode passes raw `echo.Context` to each handler method,
leaving request unmarshalling and response serialization to the handler implementation —
this is boilerplate that every handler author must reproduce consistently.

## Decision

Use **oapi-codegen with `--generate=echo-server,strict-server`**, scoped per OpenAPI tag,
so each handler package owns only the interface for its own tag.

Each handler file carries two `go:generate` directives at the top:

```go
//go:generate oapi-codegen --include-tags=<tag> --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=<tag> --package=gen --generate=echo-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml
```

The `--include-tags=<tag>` flag filters the bundled spec to only the operations that belong
to this handler package. The generated `gen/` sub-package contains:

- `type.gen.go` — request/response types for the tag.
- `server.gen.go` — the `StrictServerInterface` for the tag, plus the `echo-server` glue
  that registers routes and calls the strict handler.

In **strict-server mode** the generated `StrictServerInterface` receives a typed
`RequestObject` (carrying already-unmarshalled params and body) and returns a typed
`ResponseObject`. The handler implementation never calls `c.Bind`, `c.JSON`, or similar
directly — the strict glue layer handles all marshalling. Example:

```go
type StrictServerInterface interface {
    GetUsers(ctx context.Context, request GetUsersRequestObject) (GetUsersResponseObject, error)
    PostUsers(ctx context.Context, request PostUsersRequestObject) (PostUsersResponseObject, error)
}
```

`NewStrictHandler(ssi StrictServerInterface, middlewares []StrictMiddlewareFunc)` adapts the
strict interface to the plain `ServerInterface` that Echo route registration expects.

## Consequences

### Positive Consequences

- Each handler package is independently generated, compiled, and tested; adding a new tag
  does not affect existing handler packages.
- Strict-server mode eliminates per-handler boilerplate: unmarshalling and serialisation
  are handled by generated code.
- Handler methods receive fully typed Go structs, making type mismatches compile-time
  errors rather than runtime panics.
- Regenerating a single package is fast (`go generate ./internal/controller/handler/<pkg>/`).

### Negative Consequences

- Every handler package must declare its own `//go:generate` directives — there is no
  single global generation target.
- The generated strict glue layer adds an indirection layer between Echo and the handler
  implementation; developers unfamiliar with strict-server mode may find the call flow
  non-obvious.
- Regeneration requires the bundled `openapi.gen.yaml` to exist first (see
  [ADR-0010](0010-redocly-modular-spec-pipeline.md)).

## Alternatives Considered

### Single global generation (all tags in one package)

Simpler to configure. Rejected because it couples all handler packages to a shared
interface — adding or changing one endpoint recompiles everything and makes responsibility
boundaries ambiguous. In active team development, simultaneous edits to a single shared
generated file also produce frequent merge conflicts.

### Plain echo-server mode (without strict-server)

Still scoped per tag, but each handler method receives a raw `echo.Context` and must
perform its own binding and serialisation. Rejected because it reproduces the same
boilerplate in every handler and leaves room for inconsistent error handling. Writing this
binding and serialisation boilerplate in every handler also significantly inflates overall
line count.

## Notes

- `//go:generate` directives are at the top of each `*_handler.go` file under
  [`internal/controller/handler/`](../../internal/controller/handler/).
- Generated files are in the `gen/` sub-package of each handler directory and must not be
  edited by hand.
- The handler layer contract (one `StrictServerInterface` method per `operationId`, no
  business logic in handlers) is enforced by the architecture rules in
  [`docs/rules.md`](../rules.md).
- Parent decision: [ADR-0009](0009-openapi-first.md) (OpenAPI-first).
- Spec bundling prerequisite: [ADR-0010](0010-redocly-modular-spec-pipeline.md).
