# scripts

English | [日本語](README.ja.md)

`scripts/` contains **utility scripts** for code generation, documentation, versioning, and initial project setup.

## Directory Structure

```text
scripts/
├── gen-docs-json.mjs           # Generate docs.json for portal navigation
├── gen-portal-docs.mjs         # Copy docs to portal based on manifest.yaml
├── semver.mjs                  # Semantic versioning helper (patch/minor/major)
├── sync-versions/              # Mirror mise.toml go / node / python values to go.mod and Dockerfile FROM (Go)
├── make_help.mjs                # Generate Make target help output
├── genctxkey/                  # Context key code generator (Go)
└── setup/                     # Initial project setup scripts
    ├── replace-module.mjs
    ├── replace-app-metadata.mjs
    ├── replace-license-copyright.mjs
    ├── replace-repository-reference.mjs
    └── lib/                   # Shared utilities for setup scripts
```

## Script Categories

### Documentation Generation

|Script|Description|Invoked By|
|---|---|---|
|`gen-portal-docs.mjs`|Copy source docs to portal `guides/` based on `manifest.yaml`|`make gen-docs`|
|`gen-docs-json.mjs`|Generate `docs.json` navigation for the portal app|`make gen-docs`|

### Versioning

|Script|Description|Invoked By|
|---|---|---|
|`semver.mjs`|Bump semantic version (patch/minor/major)|Release workflow|
|`sync-versions/`|Go-based sync utility. Parses `mise.toml` `[tools]` (table-scoped, no external deps) and propagates `go` / `node` / `python` versions to `go.mod` (`go` directive) + `docker/*/Dockerfile` `FROM golang:` / `FROM node:` / `FROM python:` lines. Pre-validates all rules (version present, file exists, expected match count) and writes per file atomically, so failures never leave a partial state.|`make sync-versions`|

All other tool versions are managed by [`mise.toml`](../mise.toml) as the single source of truth. Each environment (host / docker / CI) installs only what it needs via `mise install <tool>` — no sync script required for those.

### Makefile Support

|Script|Description|Invoked By|
|---|---|---|
|`make_help.mjs`|Parse `.makefiles/*.mk` and display target descriptions|`make help`|

### Code Generation

|Script|Description|Invoked By|
|---|---|---|
|`genctxkey/`|Generate Echo context key helpers (Go code generator)|`make gen-ctxkey`|

See [genctxkey/README.md](genctxkey/README.md) for details.

### Initial Setup (`setup/`)

Scripts for configuring the boilerplate when creating a new project from this template.

|Script|Description|
|---|---|
|`replace-module.mjs`|Replace Go module name across all `.go`, `go.mod`, etc.|
|`replace-app-metadata.mjs`|Replace app name/description in env files and OpenAPI spec|
|`replace-license-copyright.mjs`|Replace LICENSE copyright holder and year|
|`replace-repository-reference.mjs`|Replace GitHub repository references in READMEs and OpenAPI|

All setup scripts support `--dry-run` for preview.

## Notes

- Documentation scripts require Node.js with `js-yaml` (installed via `docker/tools/`)
- Setup scripts are one-time use — run when creating a new project from the boilerplate
- AI agents should not modify this directory unless explicitly instructed
