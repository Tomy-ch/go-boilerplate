# OpenAPI Parameters

English | [日本語](README.ja.md)

`openapi/components/parameters/` stores **reusable OpenAPI parameter definitions** organized by concern.

## Directory Structure

```text
parameters/
├── pagination/         # Pagination parameters (shared across endpoints)
│   ├── PageParam.yaml
│   └── PerPageParam.yaml
├── search/             # Search parameters (shared across endpoints)
│   ├── KeywordParam.yaml
│   └── ActiveParam.yaml
└── user/               # User-specific parameters (sample)
    └── UserIdParam.yaml
```

> `user/` is a **sample implementation**. When building your own service, use it as a reference and replace or remove as needed.

## Available Parameters

### pagination

|File|Parameter|Type|Description|
|---|---|---|---|
|`PageParam.yaml`|`page` (query)|integer|Page number (1-based, default: 1)|
|`PerPageParam.yaml`|`per_page` (query)|integer|Items per page (default: 10, max: 100)|

### search

|File|Parameter|Type|Description|
|---|---|---|---|
|`KeywordParam.yaml`|`keyword` (query)|string|Full-text search keyword|
|`ActiveParam.yaml`|`active` (query)|boolean|Filter by active state (true/false/omit for all)|

### user (sample)

|File|Parameter|Type|Description|
|---|---|---|---|
|`UserIdParam.yaml`|`user_id` (path)|string (uuid)|User UUID|

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
