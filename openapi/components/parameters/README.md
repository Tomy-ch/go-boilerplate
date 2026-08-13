# OpenAPI Parameters

English | [日本語](README.ja.md)

`openapi/components/parameters/` stores **reusable OpenAPI parameter definitions** organized by concern.

## Directory Structure

One directory per parameter group. A group shared across endpoints is named after the concern
(`pagination/`, `search/`); one owned by a single resource is named after that resource.

## Pagination strategies

This project offers **two pagination strategies**, and each deliberately keeps its own idiomatic parameter names — they are **not** unified on purpose:

- **Offset** (`page` / `perPage`) — REST-style page navigation. Use for finite, page-addressable lists.
- **Cursor / keyset** (`first` / `after`) — Relay-style forward traversal. Use for large or infinite feeds where offset is too expensive or unstable.

Renaming the cursor params to `perPage` etc. would be non-idiomatic (cursor traversal has no "pages") and an API-breaking change, so the split is intentional.

## Usage

Reference parameters in endpoint definitions using `$ref`:

```yaml
parameters:
  - $ref: '../components/parameters/pagination/PageParam.yaml'
  - $ref: '../components/parameters/pagination/PerPageParam.yaml'
  - $ref: '../components/parameters/search/KeywordParam.yaml'
```

## Naming Convention

|Element|Convention|Example|
|---|---|---|
|Directory|lowercase by concern|`pagination/`, `search/`, `user/`|
|File name|PascalCase + `Param`|`PageParam.yaml`, `UserIdParam.yaml`|

## Rules

- `in: path` parameters must always set `required: true`
- Include `description` and `example` on all parameters for documentation quality
- Use `minimum`, `maximum`, `default` where appropriate for query parameters
- One parameter per file for maximum reusability
