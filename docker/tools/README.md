# Tools Container

English | [日本語](README.ja.md)

This Dockerfile provides **code generation and bundling tool containers** for the project. It uses multi-stage builds to offer Go, Node.js, and Python tool environments.

## Role

`docker/tools/Dockerfile` packages every code-generation / linting / security / documentation tool the build needs (oapi-codegen, mockgen, sqlc, migrate, trivy, actionlint, hadolint, gitleaks, godoc, godoc-static, redocly-cli, markdownlint-cli2, js-yaml, sqlfluff) into language-isolated runner images. Developers and CI invoke these containers from `make` targets (`make gen-api`, `make gen-query`, `make sql-lint`, etc.) so nobody has to install Go, Node, or Python toolchains locally. This keeps tool versions reproducible across machines and locks generated output to a known toolchain set.

## Build Targets

|Target|Base Image|Included Tools|
|---|---|---|
|`go_tools`|`golang:1.26.4-alpine`|oapi-codegen, mockgen, sqlc, migrate, trivy, actionlint, hadolint, gitleaks, godoc, godoc-static|
|`node_tools`|`node:24.14-alpine`|redocly-cli, markdownlint-cli2, js-yaml, esbuild (+ portal bundling libs)|
|`python_tools`|`python:3.14.2-slim`|sqlfluff|

## go_tools

Code generation, linting, security, and documentation tools for Go:

|Tool|Purpose|
|---|---|
|`oapi-codegen`|Generate Go server/types from OpenAPI spec|
|`mockgen`|Generate mocks from Go interfaces|
|`sqlc`|Generate type-safe Go code from SQL|
|`migrate`|Database migration CLI|
|`trivy`|Vulnerability and misconfiguration scanner|
|`actionlint`|GitHub Actions workflow linter|
|`hadolint`|Dockerfile linter|
|`gitleaks`|Secret scanner for committed credentials|
|`godoc`|Serve/generate Go package documentation|
|`godoc-static`|Generate static HTML from godoc output|

## node_tools

Tools for OpenAPI document processing and portal frontend bundling:

|Tool|Purpose|
|---|---|
|`redocly-cli`|Bundle OpenAPI YAML (`$ref` resolution) and generate HTML docs|
|`markdownlint-cli2`|Markdown linter for docs (`make md-lint`)|
|`js-yaml`|YAML processing for portal doc generation scripts|
|`esbuild`|Bundle the portal frontend (`docs/portal/src/main.jsx`) into `docs/portal/dist/` (`make gen-portal-build`)|
|`react` / `react-dom` / `marked` / `fuse.js` / `mermaid` / `highlight.js`|Portal frontend runtime libraries bundled by esbuild (replacing the former CDN + in-browser Babel setup). `mermaid` is also reused by `scripts/mermaid-lint.mjs` to validate ` ```mermaid ` fences (`make md-lint`).|
|`linkedom`|Headless DOM that lets `mermaid.parse` run in Node for the Markdown mermaid syntax lint (`scripts/mermaid-lint.mjs`)|

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

## Notes

- Working directory is `/app` for all targets
- Tools are installed in a builder stage and copied to the runtime stage to minimize image size (`go_tools`)
- Tool versions are pinned in `mise.toml` (the version SSOT); update them there so local and CI images stay in sync
