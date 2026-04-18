# Tools Container

English | [日本語](README.ja.md)

This Dockerfile provides **code generation and bundling tool containers** for the project. It uses multi-stage builds to offer Go, Node.js, and Python tool environments.

## Build Targets

|Target|Base Image|Included Tools|
|---|---|---|
|`go_tools`|`golang:1.26.1-alpine`|oapi-codegen, mockgen, sqlc, migrate|
|`node_tools`|`node:24.14-alpine`|redocly-cli, js-yaml|
|`python_tools`|`python:3.14.2-slim`|sqlfluff|

## go_tools

Code generation tools for Go:

|Tool|Purpose|
|---|---|
|`oapi-codegen`|Generate Go server/types from OpenAPI spec|
|`mockgen`|Generate mocks from Go interfaces|
|`sqlc`|Generate type-safe Go code from SQL|
|`migrate`|Database migration CLI|

## node_tools

Tools for OpenAPI document processing:

|Tool|Purpose|
|---|---|
|`redocly-cli`|Bundle OpenAPI YAML (`$ref` resolution) and generate HTML docs|
|`js-yaml`|YAML processing for portal doc generation scripts|

## python_tools

SQL linting tools:

|Tool|Purpose|
|---|---|
|`sqlfluff`|SQL linter for migrations, DML, and seed files|

## docker-compose Services

```yaml
go_tool_runner:    # target: go_tools,    profile: generate
node_tool_runner:  # target: node_tools,  profile: generate
python_tool_runner: # target: python_tools, profile: generate
```

All tool runners mount the project root to `/app` and run as root.

## Execution

```bash
make gen        # Run all code generation
make gen-api    # OpenAPI bundle + Go code generation
make gen-query  # sqlc code generation
```

## When Adding a New Tool

1. Install the tool in the appropriate Dockerfile stage
2. Add a new service in `docker-compose.yaml` with `profiles: [generate]`
3. Add a Makefile target if needed
4. **Update the version recording script** — see `docs/maintenance/versions-generator.md`

## Notes

- Working directory is `/app` for all targets
- Tools are installed in a builder stage and copied to the runtime stage to minimize image size (`go_tools`)
- `@latest` is used during initial development — pin versions for CI parity
