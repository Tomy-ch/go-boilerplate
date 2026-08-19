# OpenAPI Request Schemas

English | [日本語](README.ja.md)

`openapi/components/requests/` stores **endpoint request-body schemas**. They are plain OpenAPI **schemas** (not the `requestBodies` component object type — see [`schemas/README.md`](../schemas/README.md)) and are referenced from a path under `requestBody.content.<media>.schema.$ref`.

## Role boundary vs `schemas/`

- `schemas/` — small, reusable building blocks (e.g. `UserBaseInputRequest.yaml`).
- `requests/` — the **per-endpoint** shape, usually composing a base block via `allOf` and adding operation-specific constraints (e.g. a `required` list).

```yaml
# requests/users/UsersPostRequest.yaml
allOf:
  - $ref: '../../schemas/UserBaseInputRequest.yaml'
  - required:          # fields mandatory for "create user"
      - firstName
      - lastName
      - email
```

## Directory Contents

One directory per resource, named after it; each holds that resource's request bodies.

## Naming Convention

|Element|Convention|Example|
|---|---|---|
|Directory|lowercase by resource|`users/`|
|File name|PascalCase + `Request`|`UsersPostRequest.yaml`|

## Rules

- Compose a reusable base from `schemas/` via `allOf` instead of duplicating fields.
- Declare `additionalProperties: false` on the schema object that declares the properties, so unknown fields are rejected. `additionalProperties` only sees properties declared **in the same schema object**, so when composing with `allOf` the declaration belongs on the base that holds them — a wrapper adding only `required` cannot see them and would reject every field instead.
- A nested element schema (an array's `items`) needs its own declaration; the enclosing request's does not reach it. `TestRequestBodyRejectsUnknownFields` in [`internal/architest`](../../../internal/architest/README.md) checks both halves for every schema reachable from a request body.
- Include `description` and `example` on all properties.
- Boundary values (`maxLength` etc.) are a **wire contract**, not the domain rule — see [Input Boundary Value Ownership](../../boundary-ownership.md).
