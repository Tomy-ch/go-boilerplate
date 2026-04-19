# OpenAPI Schemas

English | [日本語](README.ja.md)

`openapi/components/schemas/` stores **reusable OpenAPI schema definitions** — data structures for requests, responses, and security schemes.

## Directory Contents

|File|Type|Description|
|---|---|---|
|`ErrorResponse.yaml`|Response|Unified error response schema (code / message / details / request_id)|
|`PaginationMetadataResponse.yaml`|Response|Pagination metadata (total / limit / offset)|
|`BasicAuth.yaml`|Security|HTTP Basic authentication scheme|
|`BearerAuth.yaml`|Security|HTTP Bearer (JWT) authentication scheme|
|`UserBaseInputRequest.yaml`|Request|User input fields (sample)|
|`UserResponse.yaml`|Response|User response fields (sample)|

> `User*` files are **sample implementations**. Use them as reference for naming and structure conventions when building your own schemas.

## Design Policy

### One File = One Schema

Each file defines a single top-level schema. Do not nest multiple schemas in one file.

```yaml
# Good: UserResponse.yaml
type: object
required:
  - name
  - email
properties:
  name:
    type: string
  email:
    type: string
    format: email
```

### Naming Convention

|Element|Convention|Example|
|---|---|---|
|File name|PascalCase|`ErrorResponse.yaml`, `UserBaseInputRequest.yaml`|
|Request schemas|`*Request.yaml`|`UserBaseInputRequest.yaml`|
|Response schemas|`*Response.yaml`|`UserResponse.yaml`, `ErrorResponse.yaml`|
|Security schemes|Descriptive name|`BasicAuth.yaml`, `BearerAuth.yaml`|

### Request and Response in schemas/

Due to constraints with `redocly` bundling and `oapi-codegen` generation, request bodies and responses are defined as **schemas** (not under `requestBodies/` or `responses/`).

### $ref Usage

All `$ref` references use **relative YAML paths** (not `#/components/...` fragment format) for compatibility with `redocly bundle`.

```yaml
# From a path definition
schema:
  $ref: '../components/schemas/UserResponse.yaml'
```

### PATCH Support

For PATCH operations, create a dedicated wrapper schema using `allOf`:

```yaml
# UserPatchRequest.yaml
allOf:
  - $ref: './UserBaseInputRequest.yaml'
additionalProperties: false
```

This separates the input structure from the operation semantics.

## Core Schemas

### ErrorResponse

Unified error response used across all endpoints:

```yaml
type: object
required: [code, message, request_id]
properties:
  code:        # Machine-readable error code (e.g., BAD_REQUEST)
  message:     # Human-readable error message
  details:     # Optional array of detail strings
  request_id:  # Request tracking ID
```

Maps to `response.HTTPErrorResponse` in Go — see `internal/controller/error/response/`.

### PaginationMetadataResponse

Pagination metadata returned with list endpoints:

```yaml
type: object
required: [total, limit, offset]
properties:
  total:   # Total item count
  limit:   # Items per page
  offset:  # Current offset
```

## Rules

- Avoid defining schemas inline in path definitions — always extract to `schemas/`
- Do not make schemas serve both request and response purposes
- Include `description` and `example` on all properties
- Use `required` to explicitly list mandatory fields
- Keep `additionalProperties: false` on request schemas to reject unknown fields

## Checklist

- [ ] One file = one schema
- [ ] File name matches schema purpose (PascalCase)
- [ ] `$ref` uses relative path format
- [ ] PATCH uses a dedicated wrapper schema
- [ ] `description` and `example` on all properties
