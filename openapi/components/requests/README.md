# OpenAPI Request Schemas

English | [日本語](README.ja.md)

`openapi/components/requests/` stores **endpoint request-body schemas**. They are plain OpenAPI **schemas** (not the `requestBodies` component object type — see [`schemas/README.md`](../schemas/README.md)) and are referenced from a path under `requestBody.content.<media>.schema.$ref`.

## Role boundary vs `schemas/`

- `schemas/` — small, reusable building blocks (e.g. `UserBaseInputRequest.yaml`).
- `requests/` — the **per-endpoint** shape, usually composing a base block via `allOf` and adding operation-specific fields.

```yaml
# requests/users/UsersPostRequest.yaml
allOf:
  - $ref: '../../schemas/UserBaseInputRequest.yaml'
  - properties:
      password:        # field specific to "create user"
        type: string
```

## Directory Contents

```text
requests/
└── users/
    ├── UsersPostRequest.yaml      # Create user (base + password)
    ├── UserPutRequest.yaml        # Full update (all fields required)
    ├── UserPatchRequest.yaml      # Partial update (base, fields optional)
    └── UserPasswordPutRequest.yaml # Password change (current + new)
```

> `users/` is a **sample implementation**. Mirror its structure for your own resources.

## Naming Convention

|Element|Convention|Example|
|---|---|---|
|Directory|lowercase by resource|`users/`|
|File name|PascalCase + `Request`|`UsersPostRequest.yaml`|

## Rules

- Compose a reusable base from `schemas/` via `allOf` instead of duplicating fields.
- Keep `additionalProperties: false` (directly or on the wrapper) to reject unknown fields.
- Include `description` and `example` on all properties.
- Boundary values (`maxLength` etc.) are a **wire contract**, not the domain rule — see [Input Boundary Value Ownership](../../boundary-ownership.md).
