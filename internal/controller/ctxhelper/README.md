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

Then execute the following.

```bash
make gen-go-code
```

For details on the generation process and options, refer to [scripts/genctxkey/README.md](../../../scripts/genctxkey/README.md).

## Notes

### How to specify type

The `--type` when generating ctxkey must be specified according to the following rules.

#### Primitive types / same-package types

```bash
--type string
--type UserID
```

- No import is required
- It is handled directly as a type

#### Types from external packages (recommended)

```bash
--type github.com/your/project/internal/domain/auth.Authn
```

- Specify in the format `<import-path>.<Type>`
- The generator automatically resolves import and alias

Example of generated code:

```go
import (
    auth "github.com/your/project/internal/domain/auth"
)

func GetAuthn(ctx context.Context) (auth.Authn, bool)
```

#### Caution

- Specifying only `<import-path>` (e.g., `github.com/foo/bar`) is not allowed
- Be sure to include the type name

### About editing

Files with `.gen.go` in this directory are automatically generated code.

- Manual editing is prohibited in principle
- Make changes through `scripts/genctxkey`

As an exception, minor fixes for dependency resolution (such as import adjustments) are allowed,
but it is recommended to implement permanent fixes on the generator side.
