# OpenAPI Response Schemas

English | [日本語](README.ja.md)

`openapi/components/responses/` stores **endpoint response-body schemas**. They are plain OpenAPI **schemas** (not the `responses` component object type — see [`schemas/README.md`](../schemas/README.md)) and are referenced from a path under `responses.<status>.content.<media>.schema.$ref`.

## Role boundary vs `schemas/`

- `schemas/` — small, reusable building blocks (`UserResponse.yaml`, `PaginationMetadataResponse.yaml`, `CursorPaginationMetadataResponse.yaml`, `ErrorResponse.yaml`).
- `responses/` — the **per-endpoint** shape, usually composing those blocks via `allOf` (e.g. an item array + pagination metadata).

<!-- sample-api:replace-begin -->
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
<!-- sample-api:replace-with -->
<!-- = ```yaml -->
<!-- = # responses/<resources>/<Resources>Response.yaml -->
<!-- = allOf: -->
<!-- =   - type: object -->
<!-- =     required: [<resources>] -->
<!-- =     properties: -->
<!-- =       <resources>: -->
<!-- =         type: array -->
<!-- =         items: -->
<!-- =           $ref: '../../schemas/<Resource>Response.yaml' -->
<!-- =   - $ref: '../../schemas/PaginationMetadataResponse.yaml' -->
<!-- = ``` -->
<!-- sample-api:replace-end -->

## Directory Contents

One directory per resource or concern, named after it; each holds that unit's response bodies.

## Naming Convention

|Element|Convention|Example|
|---|---|---|
<!-- sample-api:replace-begin -->
|Directory|lowercase by resource / concern|`users/`, `health-check/`, `version/`|
|File name|PascalCase + `Response`|`UsersResponse.yaml`, `VersionResponse.yaml`|
<!-- sample-api:replace-with -->
<!-- = |Directory|lowercase by resource / concern|`<resources>/`, `health-check/`, `version/`| -->
<!-- = |File name|PascalCase + `Response`|`<Resources>Response.yaml`, `VersionResponse.yaml`| -->
<!-- sample-api:replace-end -->

## Rules

- Compose reusable blocks from `schemas/` via `allOf` instead of duplicating fields.
- A list response must include the matching pagination metadata block (`PaginationMetadataResponse` for offset, `CursorPaginationMetadataResponse` for cursor).
- The response constraint must cover everything the domain can emit (`domain ⊆ response`) — see [Input Boundary Value Ownership](../../boundary-ownership.md).
- Include `description` and `example` on all properties.
