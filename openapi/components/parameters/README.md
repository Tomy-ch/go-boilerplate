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

## Filter conventions

These apply to every search endpoint, not only the ones that exist today.

### Multiple values for one condition use an `explode: false` array

A condition that can hold several values is **one** parameter carrying a comma-separated array, never the
same name repeated. Repeating a name is rejected by request validation, so a client cannot express "either
of these" at all unless the parameter is declared as an array.

```yaml
explode: false
schema:
  type: array
  uniqueItems: true
  maxItems: 32
  items: { type: integer, format: int32, minimum: 1, maximum: 32767 }
```

`uniqueItems` and `maxItems` are wire limits: they bound the URL length and the size of the resulting `IN`
list. Existing examples: `product/CategoryCodesParam.yaml`, `product/StatusCodesParam.yaml`, and
`purchase/PurchaseGroupByParam.yaml` (a string `enum` array).

### Filtering by master data takes `code`, not the row's UUID

The identifier a client sends for a master row is its `code` — a static alias fixed by the migration that
inserted the row. Which UUID that row carries is decided by the migration, so application code must not
hold it ([ADR-0029](../../../docs/adr/0029-master-data-via-migration.md)); the same reasoning applies at the
API boundary. A client can keep `code` as a constant, whereas a UUID has to be resolved by calling the
master endpoint first, and at 36 characters each it makes a multi-value filter expensive to express.

A code matching no master row yields **zero results**, not a 404 — it is a filter, not a lookup.

### `code` is an int16 range

`type: integer` / `format: int32` / `minimum: 1` / `maximum: 32767`. OpenAPI has no int16, so the range is
carried by the bounds and narrowed in Go through `pkg/safecast`. This matches the `SMALLINT` column, the
sqlc-generated `int16`, and the domain's own `minCode` / `maxCode`.

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
