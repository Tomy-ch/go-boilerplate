# OpenAPI Response Schemas

English | [日本語](README.ja.md)

`openapi/components/responses/` stores **endpoint response-body schemas**. They are plain OpenAPI **schemas** (not the `responses` component object type — see [`schemas/README.md`](../schemas/README.md)) and are referenced from a path under `responses.<status>.content.<media>.schema.$ref`.

## Role boundary vs `schemas/`

- `schemas/` — small, reusable building blocks (`UserResponse.yaml`, `PaginationMetadataResponse.yaml`, `CursorPaginationMetadataResponse.yaml`, `ErrorResponse.yaml`).
- `responses/` — the **per-endpoint** shape, usually composing those blocks via `allOf` (e.g. an item array + pagination metadata).

```yaml
# responses/users/UsersResponse.yaml
allOf:
  - type: object
    required: [users]
    properties:
      users:
        type: array
        items:
          $ref: '../../schemas/UserResponse.yaml'
  - $ref: '../../schemas/PaginationMetadataResponse.yaml'
```

## Directory Contents

One directory per resource or concern, named after it; each holds that unit's response bodies.

## Naming Convention

|Element|Convention|Example|
|---|---|---|
|Directory|lowercase by resource / concern|`users/`, `health-check/`, `version/`|
|File name|PascalCase + `Response`|`UsersResponse.yaml`, `VersionResponse.yaml`|

## Rules

- Compose reusable blocks from `schemas/` via `allOf` instead of duplicating fields.
- A list response must include the matching pagination metadata block (`PaginationMetadataResponse` for offset, `CursorPaginationMetadataResponse` for cursor).
- The response constraint must cover everything the domain can emit (`domain ⊆ response`) — see [Input Boundary Value Ownership](../../boundary-ownership.md).
- Include `description` and `example` on all properties.
