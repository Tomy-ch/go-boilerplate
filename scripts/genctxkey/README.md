# genctxkey

English | [日本語](README.ja.md)

`genctxkey` is a tool that generates code to pass values to and from `context.Context` and `echo.Context` in a type-safe manner.

## Overview

The goal is to handle storing and retrieving values in context in a unified way while avoiding the following issues.

- collisions caused by string keys
- lack of type safety
- inconsistency in implementation

This tool automatically generates helper functions to solve these problems.

## Generated Code

- for context.Context
  - `SetXxx`
  - `GetXxx`
- for echo.Context
  - `SetXxxToEcho`
  - `GetXxxFromEcho`

## Usage

### 1. Define in generate.go

```go
package ctxhelper

//go:generate go run ../../../scripts/genctxkey --name authn --type "auth.Authn" --import boilerplate-go/internal/usecase/boundary/auth --out .
```

### 2. Generate code

```sh
make gen-go-code
```

## How to specify type

### Primitive types / same-package types

```sh
--type string
--type UserID
```

- import is not required
- handled directly as a Go type

### Types from external packages

```sh
--type "auth.Authn"
--import boilerplate-go/internal/usecase/boundary/auth
```

- specify Go type format in `--type`
- explicitly specify package with `--import`
- `--alias` is optional (auto-generated if not specified)

### Complex types (supported)

```sh
--type "*[]auth.Authn"
--type "map[string]auth.Authn"
```

- supports pointer / slice / map / generic
- types are treated as Go type expressions, not strings

## Invalid examples

```sh
--type github.com/foo/bar
```

- specifying only import path is invalid
- when using external types, combine `--type` and `--import`

## Output Specification

- file names are all lowercase
  - example: `authn_ctx.gen.go`
- test files are also automatically generated
  - example: `authn_ctx_test.go`

## Design Policy

### 1. Responsibilities of generator

- focus on code generation
- treat types as Go type expressions, do not analyze them
- process imports based on CLI input

### 2. template has minimal responsibility

- contains no logic
- display only

### 3. deterministic (reproducible)

- no heuristic processing (goimports not used)
- always generates identical results

## About editing

Generated `.gen.go` files are auto-generated code.

- manual editing is prohibited in principle
- make changes through the generator

### Exceptions

The following are allowed due to dependency resolution needs:

- import adjustments
- alias changes

However:

- may be overwritten during regeneration
- permanent fixes should be made on the generator side

## Relationship with CI

- `ctxhelper` generation is not executed in CI
- generation is performed locally, and results are committed

## Notes

This tool is based on the following principles:

- context is used within a "controlled boundary"
- direct manipulation is prohibited; always use wrappers
- consistency is ensured through generation

## Summary

|Item|Policy|
|------|------|
|type specification|Go type format|
|import|explicit via CLI|
|generation|reproducible|
|editing|prohibited in principle (exceptions exist)|

This tool is a foundational component that ensures consistency and safety in context usage.
