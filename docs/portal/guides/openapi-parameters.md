# OpenAPI Parameters

English | [日本語](README.ja.md)

`openapi/components/parameters/` stores **reusable OpenAPI parameter definitions** organized by concern.

## Directory Structure

```text
parameters/
├── pagination/         # Pagination parameters (shared across endpoints)
│   ├── PageParam.yaml        # offset strategy
│   ├── PerPageParam.yaml     # offset strategy
│   ├── CursorFirstParam.yaml # cursor strategy
│   └── CursorAfterParam.yaml # cursor strategy
├── search/             # Search parameters (shared across endpoints)
│   ├── KeywordParam.yaml
│   └── ActiveParam.yaml
└── user/               # User-specific parameters (sample)
    └── UserIdParam.yaml
```

> `user/` is a **sample implementation**. When building your own service, use it as a reference and replace or remove as needed.

## Available Parameters

### pagination

This project offers **two pagination strategies**, and each deliberately keeps its own idiomatic parameter names — they are **not** unified on purpose:

- **Offset** (`page` / `perPage`) — REST-style page navigation. Use for finite, page-addressable lists.
- **Cursor / keyset** (`first` / `after`) — Relay-style forward traversal. Use for large or infinite feeds where offset is too expensive or unstable.

Renaming the cursor params to `perPage` etc. would be non-idiomatic (cursor traversal has no "pages") and an API-breaking change, so the split is intentional.

|File|Parameter|Type|Strategy|Description|
|---|---|---|---|---|
|`PageParam.yaml`|`page` (query)|integer|offset|Page number (1-based, default: 1)|
|`PerPageParam.yaml`|`perPage` (query)|integer|offset|Items per page (default: 10, max: 100)|
|`CursorFirstParam.yaml`|`first` (query)|integer|cursor|Max items to return (default: 50, max: 200)|
|`CursorAfterParam.yaml`|`after` (query)|string|cursor|Opaque cursor for the next page; omit for the first page|

### search

|File|Parameter|Type|Description|
|---|---|---|---|
|`KeywordParam.yaml`|`keyword` (query)|string|Full-text search keyword|
|`ActiveParam.yaml`|`active` (query)|boolean|Filter by active state (true/false/omit for all)|

### user (sample)

|File|Parameter|Type|Description|
|---|---|---|---|
|`UserIdParam.yaml`|`userId` (path)|string (uuid)|User UUID|

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
