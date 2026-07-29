# ctxhelper

English | [日本語](README.ja.md)

ctxhelper is a "boundary layer that controls the usage of context".

This package provides helper functions for carrying request-scoped values on `context.Context`.

## Implementation Method

Simple value keys are created through code generation. For details on the generation mechanism, refer to the following:

- `scripts/genctxkey/README.md`

The `Authn` helpers (`authn.go`) are hand-written: since the OpenAPI `AuthenticationFunc` cannot propagate context forward, `Authn` is carried through a mutable slot installed by the middleware before authentication.

## Provided helpers

Hand-written (`authn.go`) — the `Authn` slot:

- `WithAuthn(ctx) context.Context` — install an empty `Authn` slot (call before authentication)
- `SetAuthn(ctx, authn) bool` — write into the slot; returns `false` when no slot is present
- `GetAuthn(ctx) (auth.Authn, bool)` — read from the slot; `ok=false` when unset

Generated (`genctxkey`, defined in `generate.go`) — boolean request-scoped flags. Each name exposes a `context.Context` pair plus an `*echo.Context` pair:

- `ErrorHandled` — `SetErrorHandled` / `GetErrorHandled`, `SetErrorHandledToEcho` / `GetErrorHandledFromEcho`
- `Recovered` — `SetRecovered` / `GetRecovered`, `SetRecoveredToEcho` / `GetRecoveredFromEcho`

## Usage

When adding a ctxkey, add a definition to `generate.go` as follows.

```go
//go:generate go run ../../../scripts/genctxkey --name UserID --type string --out .
```

When using external types:

```go
//go:generate go run ../../../scripts/genctxkey --name Actor --type "auth.Authn" --import go-boilerplate/internal/usecase/boundary/auth --out .
```

The example above only illustrates the external-type syntax. The real `Authn` slot in this package is hand-written (`authn.go`) and is **not** produced by this command.

Then execute the following.

```bash
make gen-go-code
```

## How to specify type

### Primitive types / same-package types

```bash
--type string
--type UserID
```

- import is not required
- handled directly as a type

### Types from external packages

```bash
--type "auth.Authn"
--import go-boilerplate/internal/usecase/boundary/auth
```

- `--type` is specified in Go type format
- package is explicitly specified with `--import`
- `--alias` is optional

### Complex types

```bash
--type "*[]auth.Authn"
--type "map[string]auth.Authn"
```

- supports pointer / slice / map / generic

## Notes

- specifying only import path (e.g., `github.com/foo/bar`) is not allowed
- external types must always specify both `--type` and `--import`

## About editing

Files with `.gen.go` in this directory are automatically generated code.

- manual editing is prohibited in principle
- make changes through `scripts/genctxkey`

Hand-written helpers (such as `authn.go`) are edited directly.
