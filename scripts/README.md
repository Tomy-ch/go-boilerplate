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
├── skill-lint.mjs              # Validate .claude/** skill / agent definitions against reality and their .codex/** counterparts
├── pr-comment-secret-lint.mjs  # Reject a secret in a workflow job that posts a PR comment
├── genctxkey/                  # Context key code generator (Go)
├── actions-shellcheck/         # Check the `run:` scripts of composite actions with shellcheck (Go)
├── pin-actions/                # Pin GitHub Actions `uses:` references to commit SHAs (Go)
├── pin-images/                 # Pin Dockerfile `FROM` base images to digests (Go)
└── setup/                     # Initial project setup scripts
    ├── replace-module.mjs
    ├── replace-app-metadata.mjs
    ├── replace-license-copyright.mjs
    ├── replace-repository-reference.mjs
    ├── replace-codeowners.mjs
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
|`skill-lint.mjs`|Check the skill / agent definitions under `.claude/**` semantically: frontmatter (`name` matches the directory / file name, `name` + `description` present), translation pairs (`SKILL.ja.md` exists, carries no frontmatter, opens with a sync note, and its heading-level sequence matches `SKILL.md`), and reference existence (every `` `make <target>` `` resolves against `Makefile` / `.makefiles/**`, every repo-root-relative path in inline code exists). Also checks that each skill / agent exists in `.codex/**` too. Dependency-free ESM. Fills the gap that a skill definition is an agent instruction sheet whose prose nothing else checks against reality, and that a skill landing on only one of the two AI environments goes unnoticed. See [Skill Lint](#skill-lint) for scope and the ignore directive.|`make md-lint` / `make md-skill-lint`|
|`actions-shellcheck/`|Parse every `action.yaml` / `action.yml` under `.github/actions/**`, extract `runs.steps[].run` from the composite ones and check each script with `shellcheck` over stdin, remapping every finding back to its line in the `action.yaml`. Fills the gap that `actionlint` walks only `.github/workflows` and cannot be pointed at an action manifest (handed one directly, it parses it as a workflow and fails), so the shell inside a composite action was checked by nothing. The dialect comes from the step's `shell:` — passed to shellcheck as a shebang, which also settles the target shell without a `-s` flag; `pwsh` / `python` / `cmd` and an expression-valued `shell:` are counted as skipped instead. `${{ }}` expressions are masked to a placeholder that preserves the line count, the same approach `actionlint` takes for workflow `run:`. Exits non-zero when it checked zero steps, so a broken extractor cannot pass as a clean run.|`make actions-lint` / `make actions-shellcheck`|
|`pr-comment-secret-lint.mjs`|Split every workflow in `.github/workflows/` into jobs and fail when a job using `./.github/actions/upsert-pr-comment` references a secret other than `GITHUB_TOKEN`, workflow-wide `env:` included. Dependency-free ESM. Enforces a rule `actionlint` cannot express — see [`.github/workflows/README.md`](../.github/workflows/README.md) for why the rule exists. Reach: direct `secrets` references inside a `${{ }}` expression, whether `secrets.NAME`, `secrets['NAME']`, or the whole context (`toJSON(secrets)`); a secret read in one job and handed on through `needs.<job>.outputs` is beyond static reach and passes.|`make actions-lint` / `make actions-comment-secret-lint`|

#### Skill Lint

`skill-lint.mjs` only asserts what can be derived mechanically from the Makefile target list, the
filesystem, and heading extraction — it never judges wording. Reference checks read **inline code
spans outside fenced blocks** (a fence is an example or a sample output, so it guarantees nothing).

A path reference is checked only when it is unambiguously a path: its first segment is an existing
repo-root entry, and it either ends with `/` or has a dotted basename. That deliberately leaves Go
import paths (`database/sql`), package-qualified symbols (`pkg/ptr.Copy`), ellipses
(`internal/controller/handler/...`), and context-relative filenames (`SKILL.md`) unchecked — none of
them resolve to a unique path. `<placeholder>` / `*` / `**` / `{a,b}` are resolved as patterns, and a
path is also tried relative to the referencing file so a skill can point at its own bundled
`scripts/`.

For a reference that is intentionally absent — a hypothetical example, an optional location, a file
that lives in the counterpart AI tool's repository — put the ignore directive anywhere on that line:

```markdown
- `internal/controller/handler/debug/README.md` → `docs/portal/guides/controller-handler-debug.md` (if it were added) <!-- skill-lint-ignore -->
```

Across the two AI environments the script checks **existence only**: every `.claude/skills/<name>/`
has a `.codex/skills/<name>/` and vice versa, every `.claude/agents/<name>.md` has a
`.codex/agents/<name>.toml` and vice versa, and every Codex skill carries the `SKILL.md` +
`agents/openai.yaml` pair its README documents. Codex-side `SKILL.ja.md` is optional, so it is
checked as a translation pair only when present.

Body correspondence is deliberately **not** checked. `sync-ai` performs a semantic port, not a
verbatim copy, so `CLAUDE.md` ↔ `AGENTS.md` rewording, adaptation of Claude-only mechanisms, and
condensed rewrites leave permanent intentional differences — for a substantial minority of the
shared skills the two heading sets do not overlap at all. Existence parity has few enough
exceptions to declare, and still catches the failure that matters: a skill merged into one
environment only.

A skill that intentionally lives in one environment goes in the `PLATFORM_ONLY_SKILLS` map in
`skill-lint.mjs` **with a reason**. An entry with an empty reason fails, and so does one whose skill
has since appeared in both environments (or in neither) — so the exception list cannot outlive the
exception. **This is the canonical description of the mechanism; other documents link here rather
than restating it.**

Agent roles have no such escape hatch: their parity is unconditional. No agent has ever been
deliberately one-sided, so an exception map for them would carry no entries — add the mechanism
when a real case appears, not before.

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
|`replace-codeowners.mjs`|Replace the owner of every rule in `.github/CODEOWNERS`. Comment lines keep their example owner, and a rule whose owner field is unrecognizable is reported instead of rewritten.|
|`remove-sample-api.mjs`|Remove the sample API (`user`/`product`/`order`): deletes paths declared in `lib/sample-api.mjs` and strips `sample-api` marker blocks from the shared DI modules and `openapi.yaml`. Run via `make setup-remove-sample-api` to also regenerate/format/lint. <!-- sample-api:line -->|

All setup scripts support `--dry-run` for preview.
<!-- sample-api:begin -->

The deletion targets and markers for `remove-sample-api.mjs` are declared in [`lib/sample-api.mjs`](setup/lib/sample-api.mjs). The sample spans three domains (`user` is full-stack; `product`/`order` are DB stubs to be expanded), so expanding the sample only requires appending paths to the matching domain block and wrapping interleaved lines with the `sample-api:begin … sample-api:end` markers (or `sample-api:line`).
<!-- sample-api:end -->

## Notes

- Documentation scripts require Node.js with `js-yaml` (installed via `docker/tools/`)
- Setup scripts are one-time use — run when creating a new project from the boilerplate
- AI agents should not modify this directory unless explicitly instructed
