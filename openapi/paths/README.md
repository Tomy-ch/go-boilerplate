# OpenAPI Paths

English | [日本語](README.ja.md)

`openapi/paths/` stores **API endpoint route definitions** organized by resource and version.

## Directory Structure

The directory layout **mirrors the URL path** (industry-standard / Redocly split style): the file location corresponds 1:1 to the URL. A path with no children is a flat `<segment>.yaml`; a path that itself is an endpoint **and** has children is a `<segment>.yaml` file sitting beside a `<segment>/` directory for those children.

```text
paths/
├── health.yaml             # /health
├── healthz.yaml            # /healthz
├── ready.yaml              # /ready
├── version.yaml            # /version
├── metrics.yaml            # /metrics (Basic auth)
├── internal/               # Internal types (error response schema for oapi-codegen)
│   └── types/
│       └── error_response.yaml
└── v1/                     # Versioned API (sample)
    ├── users.yaml          # /v1/users       ← endpoint + has children
    └── users/
        ├── userId.yaml     # /v1/users/{userId}
        ├── search.yaml     # /v1/users/search
        ├── feed.yaml       # /v1/users/feed
        └── me.yaml         # /v1/users/me
```

## Endpoint Categories

### Operational Endpoints

Infrastructure endpoints for monitoring and orchestration. These are not part of the business API and are excluded from OpenAPI validation and authentication via the `Skipper`.

|Path|File|Description|
|---|---|---|
|`/health`|`health.yaml`|Health check|
|`/healthz`|`healthz.yaml`|Kubernetes liveness probe|
|`/ready`|`ready.yaml`|Readiness probe|
|`/version`|`version.yaml`|Build version / revision / date|
|`/metrics`|`metrics.yaml`|Prometheus metrics (Basic auth protected)|

### Versioned API (`v1/`)

Business API endpoints following a **URL versioning strategy**.

```text
/v1/users             → users.yaml
/v1/users/{userId}    → users/userId.yaml
/v1/users/me          → users/me.yaml
/v1/users/search      → users/search.yaml
```

> `v1/` contents are **sample implementations**. Replace with your own resources when building a service.

#### Versioning Strategy

This project recommends **URL path versioning** (`/v1/`, `/v2/`, etc.):

- Clear and explicit in URLs and documentation
- Allows parallel operation of multiple API versions
- Breaking changes are introduced in a new version prefix
- Non-breaking additions can be made within the current version

When introducing breaking changes, create a new `v2/` directory alongside `v1/`.

### Internal Types

`internal/types/error_response.yaml` defines the error response type used by `oapi-codegen` for type generation. This is not a public API endpoint.

## Naming and Structure Rules

|Element|Convention|Example|
|---|---|---|
|File name|Match the URL segment (lowercase); path-parameter files match the param's `camelCase` name|`search.yaml`, `userId.yaml`|
|Directory layout|Mirror the URL path (one file per path item)|`/v1/users/search` → `v1/users/search.yaml`|
|Leaf vs. parent|Leaf = flat `<segment>.yaml`; endpoint-with-children = `<segment>.yaml` beside `<segment>/`|`users.yaml` + `users/`|
|Path parameter|File name matches the param's `camelCase` name (no braces)|`{userId}` → `userId.yaml`|
|operationId|`{HTTPMethod}{Resource}` (PascalCase, verb-first)|`GetUsers`, `PostUsers`|
|tags|Path-based grouping|`v1/users`, `health`|

## Path-to-Handler Mapping

Path definitions correspond to handler implementations:

```text
paths/v1/users.yaml             → handler/v1/users/
paths/v1/users/userId.yaml      → handler/v1/users/detail/
paths/v1/users/me.yaml          → handler/v1/users/detail/
paths/v1/users/search.yaml      → handler/v1/users/search/
```

## Rules

- Use `$ref` to reference schemas, parameters, and responses — do not define inline
- Split path files by responsibility (list vs. detail vs. search)
- Do not use `#/` fragment pointers in this directory
- Align naming with `components/` for easy cross-referencing
- Each path file should define a single route
