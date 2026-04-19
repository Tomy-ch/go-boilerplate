# OpenAPI Paths

English | [日本語](README.ja.md)

`openapi/paths/` stores **API endpoint route definitions** organized by resource and version.

## Directory Structure

```text
paths/
├── health/             # Health check endpoint
│   └── health.yaml
├── healthz/            # Kubernetes health check endpoint
│   └── healthz.yaml
├── ready/              # Readiness check endpoint
│   └── ready.yaml
├── version/            # Version info endpoint
│   └── version.yaml
├── metrics/            # Prometheus metrics endpoint (Basic auth)
│   └── metrics.yaml
├── internal/           # Internal types (error response schema for oapi-codegen)
│   └── types/
│       └── error_response.yaml
└── v1/                 # Versioned API (sample)
    ├── users.yaml
    └── users/
        ├── user_id.yaml
        └── search/
            └── search.yaml
```

## Endpoint Categories

### Operational Endpoints

Infrastructure endpoints for monitoring and orchestration. These are not part of the business API and are excluded from OpenAPI validation and authentication via the `Skipper`.

|Path|File|Description|
|---|---|---|
|`/health`|`health/health.yaml`|Health check|
|`/healthz`|`healthz/healthz.yaml`|Kubernetes liveness probe|
|`/ready`|`ready/ready.yaml`|Readiness probe|
|`/version`|`version/version.yaml`|Build version / revision / date|
|`/metrics`|`metrics/metrics.yaml`|Prometheus metrics (Basic auth protected)|

### Versioned API (`v1/`)

Business API endpoints following a **URL versioning strategy**.

```text
/v1/users          → users.yaml
/v1/users/{user_id} → users/user_id.yaml
/v1/users/search   → users/search/search.yaml
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
|File name|snake_case|`user_id.yaml`, `search.yaml`|
|Directory structure|`/<version>/<resource>/`|`v1/users/`|
|operationId|`{HTTPMethod}{Resource}`|`getUsers`, `postUsers`|
|tags|Path-based grouping|`v1/users`, `health`|

## Path-to-Handler Mapping

Path definitions correspond to handler implementations:

```text
paths/v1/users.yaml           → handler/v1/users/
paths/v1/users/user_id.yaml   → handler/v1/users/detail/
paths/v1/users/search/        → handler/v1/users/search/
```

## Rules

- Use `$ref` to reference schemas, parameters, and responses — do not define inline
- Split path files by responsibility (list vs. detail vs. search)
- Do not use `#/` fragment pointers in this directory
- Align naming with `components/` for easy cross-referencing
- Each path file should define a single route
