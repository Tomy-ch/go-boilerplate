# ctxhelper

English | [日本語](README.ja.md)

ctxhelper is a "boundary layer that controls the usage of context".

This package provides helper functions for manipulating `context.Context` and the context of the Echo framework.

## Implementation Method

The code in this package must not be implemented manually, and is created through code generation.

For details on the generation mechanism, refer to the following:

- `scripts/genctxkey/README.md`

## Usage

When adding a ctxkey, add a definition to `generate.go` as follows.

```go
//go:generate go run ../../../scripts/genctxkey --name UserID --type string --out .
```

When using external types:

```go
//go:generate go run ../../../scripts/genctxkey --name Authn --type "auth.Authn" --import boilerplate-go/internal/usecase/boundary/auth --out .
```

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
--import boilerplate-go/internal/usecase/boundary/auth
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

As an exception, minor fixes for dependency resolution (such as import adjustments) are allowed,  
but it is recommended to implement permanent fixes on the generator side.
