# scripts

English | [日本語](README.ja.md)

`scripts/` contains **utility scripts** for code generation, documentation, versioning, and initial project setup.

## Directory Structure

```text
scripts/
├── gen-docs-json.mjs           # Generate docs.json for portal navigation
├── gen-portal-docs.mjs         # Copy docs to portal based on manifest.yaml
├── build-portal.mjs            # Bundle the portal frontend (src/main.jsx) with esbuild
├── semver.mjs                  # Semantic versioning helper (patch/minor/major)
├── stamp-openapi-version.mjs   # Sync openapi.yaml info.version from the release/vX.Y.Z branch name
├── sync-versions/              # Mirror mise.toml go / node / python values to go.mod and Dockerfile FROM (Go)
├── make_help.mjs                # Generate Make target help output
├── mermaid-lint.mjs            # Validate ```mermaid fences in Markdown with the real mermaid parser
├── genctxkey/                  # Context key code generator (Go)
├── pin-actions/                # Pin GitHub Actions `uses:` references to commit SHAs (Go)
├── pin-images/                 # Pin Dockerfile `FROM` base images to digests (Go)
└── setup/                     # Initial project setup scripts
    ├── replace-module.mjs
    ├── replace-app-metadata.mjs
    ├── replace-license-copyright.mjs
    ├── replace-repository-reference.mjs
    ├── remove-sample-api.mjs  # Remove the sample API (user/product/order) <!-- sample-api:line -->
    └── lib/                   # Shared utilities for setup scripts
```

## Script Categories

### Documentation Generation

|Script|Description|Invoked By|
|---|---|---|
|`gen-portal-docs.mjs`|Copy source docs to portal `guides/` based on `manifest.yaml`|`make gen-docs`|
|`gen-docs-json.mjs`|Generate `docs.json` navigation for the portal app|`make gen-docs`|
|`build-portal.mjs`|Bundle the portal frontend (`docs/portal/src/main.jsx`) into `docs/portal/dist/` (`bundle.js` / `bundle.css` + lazy chunks) with esbuild, and copy `mermaid.min.js` there too. Replaces the former CDN + in-browser Babel setup.|`make gen-portal-build`|

### Linting

|Script|Description|Invoked By|
|---|---|---|
|`mermaid-lint.mjs`|Extract every ` ```mermaid ` fence from the repo's Markdown (same exclusions as `markdownlint-cli2`) and validate each with the real `mermaid.parse` (DOM provided by `linkedom`). Exits non-zero on the first broken diagram. Fills the gap that `markdownlint` only checks Markdown shape, never the diagram grammar.|`make md-lint` / `make md-mermaid-lint`|

### Versioning

|Script|Description|Invoked By|
|---|---|---|
|`semver.mjs`|Bump semantic version (patch/minor/major)|Release workflow|
|`stamp-openapi-version.mjs`|Derive `X.Y.Z` from a `release/vX.Y.Z` branch name and write it into `openapi.yaml` `info.version` (first `version:` line only; idempotent; no-op for non-release refs). Contract version only — no SHA / build metadata (commit-level traceability is the runtime `/version`'s job). Dependency-free ESM; runs on the bare runner `node`.|`auto-generate-docs.yaml`|
|`sync-versions/`|Go-based sync utility. Parses `mise.toml` `[tools]` (table-scoped, no external deps) and propagates `go` / `node` / `python` versions to `go.mod` (`go` directive) + `docker/*/Dockerfile` `FROM golang:` / `FROM node:` / `FROM python:` lines. Pre-validates all rules (version present, file exists, expected match count) and writes per file atomically, so failures never leave a partial state.|`make sync-versions`|

All other tool versions are managed by [`mise.toml`](../mise.toml) as the single source of truth. Each environment (host / docker / CI) installs only what it needs via `mise install <tool>` — no sync script required for those.

### Makefile Support

|Script|Description|Invoked By|
|---|---|---|
|`make_help.mjs`|Parse `.makefiles/*.mk` and display target descriptions|`make help`|

### Code Generation

|Script|Description|Invoked By|
|---|---|---|
|`genctxkey/`|Generate Echo context key helpers (Go code generator). Driven by the `//go:generate` directives in `internal/controller/ctxhelper/generate.go`, run via `go generate ./...`.|`make gen-go-code`|

See [genctxkey/README.md](genctxkey/README.md) for details.

### CI / Supply Chain

|Script|Description|Invoked By|
|---|---|---|
|`pin-actions/`|Pin every external GitHub Actions `uses:` in `.github/workflows/**` and `.github/actions/**` to an immutable commit SHA. `resolve` walks the references and resolves each tag/branch to a SHA via `git ls-remote`, writing the lockfile `.github/actions-pin.toml` (SSOT) — with a supply-chain quarantine that refuses commits younger than `PIN_ACTIONS_MIN_AGE_DAYS` (default 14, keeping the existing pin instead). `apply` rewrites each `uses:` to `@<sha> # <tag>` from the lockfile. `check` runs the same comparison without writing and exits non-zero on any unpinned/stale/unregistered reference (for CI / hooks). Idempotent: an already-pinned line re-resolves off its trailing `# <tag>` comment.|`make pin-actions-resolve` / `pin-actions-apply` / `pin-actions-check`|
|`pin-images/`|Pin every `FROM` base image in `docker/*/Dockerfile` to an immutable digest. `resolve` collects each `image:tag` and resolves its current digest via `docker buildx imagetools inspect`, writing the lockfile `docker/images-pin.toml` (SSOT) — with a supply-chain cooldown that refuses digests whose image-config `created` is younger than `PIN_IMAGES_MIN_AGE_DAYS` (default 14). A mutable tag has no queryable history, so the step-back target is the tool's own prior lock entry; with none (bootstrap) the image is left tag-only. `apply` normalizes each `FROM` to `image:tag@sha256:...` from the lockfile and strips the digest back to tag-only for quarantined images. `check` runs the same comparison without writing and exits non-zero on drift (for CI / hooks). The tag stays inline as the version SSOT.|`make pin-images-resolve` / `pin-images-apply` / `pin-images-check`|

### Initial Setup (`setup/`)

Scripts for configuring the boilerplate when creating a new project from this template.

|Script|Description|
|---|---|
|`replace-module.mjs`|Replace Go module name across all `.go`, `go.mod`, etc.|
|`replace-app-metadata.mjs`|Replace app name/description in env files and OpenAPI spec|
|`replace-license-copyright.mjs`|Replace LICENSE copyright holder and year|
|`replace-repository-reference.mjs`|Replace GitHub repository references in READMEs and OpenAPI|
|`remove-sample-api.mjs`|Remove the sample API (`user`/`product`/`order`): deletes paths declared in `lib/sample-api.mjs` and strips `sample-api` marker blocks from the shared DI modules and `openapi.yaml`. Run via `make setup-remove-sample-api` to also regenerate/format/lint. <!-- sample-api:line -->|

All setup scripts support `--dry-run` for preview.
<!-- sample-api:begin -->

The deletion targets and markers for `remove-sample-api.mjs` are declared in [`lib/sample-api.mjs`](setup/lib/sample-api.mjs). The sample spans three domains (`user` is full-stack; `product`/`order` are DB stubs to be expanded), so expanding the sample only requires appending paths to the matching domain block and wrapping interleaved lines with the `sample-api:begin … sample-api:end` markers (or `sample-api:line`).
<!-- sample-api:end -->

## Notes

- Documentation scripts require Node.js with `js-yaml` (installed via `docker/tools/`)
- Setup scripts are one-time use — run when creating a new project from the boilerplate
- AI agents should not modify this directory unless explicitly instructed
