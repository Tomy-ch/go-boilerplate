# Agents Documentation

This file is the **agent operational contract** for this repository — what an AI agent may
touch, how it must behave, and where the canonical detail lives. Architecture / rules /
flows are **not restated here**; they live under `docs/` and are linked below. Keep this file
lean: it is loaded every turn.

This repository is a lightweight Onion Architecture RESTful-API scaffold
(`controller → usecase → domain`; infrastructure implements domain interfaces). Do not
introduce new architectural patterns unless explicitly instructed.

## Instruction Priority

On conflict, follow this order:

1. `AGENTS.md` (this file)
2. `docs/rules.md`
3. `docs/architecture.md`
4. User instructions

## Canonical Documentation (read before changing code)

The source of truth for design, rules, and flows is under `docs/` and the per-package
`README.md`. Read the relevant one before implementing; do not duplicate its content here.

| Need | Read |
| --- | --- |
| System structure & layer responsibilities | `docs/architecture.md` |
| Non-negotiable rules (layer deps, generated code, domain constraints, DTO boundary, tx, errors, comments) | `docs/rules.md` |
| How to perform a change (API / DB / business-logic flows, finding related code) | `docs/development-flow.md` |
| Testing conventions (structure, naming, `require`/`assert`, mocks, coverage exceptions) | `docs/testing-conventions.md` (DoD in `docs/rules.md`) |
| Technology rationale (per-file ADR: onion / OpenAPI-first / sqlc / echo / fx / worker / o11y) | `docs/adr/` (log: `docs/adr/README.md`) |
| Per-layer / per-package detail | `internal/**/README.md`, `pkg/**/README.md` |
| Subsystem design references (rest / worker / job / outbox / idempotency / observability) | `docs/design/README.md` |

**Documentation scope for agents** — the canonical sources are the English `README.md` and
`docs/**/*.md`. **Never read `*.ja.md` files: they are human-facing Japanese translations of
those canonical sources — read the canonical English original instead.** Also ignore the
documentation-portal UI assets:

```txt
**/*.ja.md
docs/ja/**
docs/portal/**
```

## Task Execution Protocol

Before implementing any change:

1. Identify the change type (API / DB / Business Logic) and follow the matching flow in `docs/development-flow.md`.
2. Locate related code (onion flow: OpenAPI → handler → usecase → domain → infrastructure → SQL; see `docs/development-flow.md` / `docs/architecture.md`).
3. Verify no existing implementation already covers it — search the same layer first; prefer editing an existing file over creating a new one.

For API changes: OpenAPI is defined first. For DB changes: the migration + SQL exist first.

## Layer Rules (hard constraints — enforced by `golangci-lint` depguard)

Boundaries are enforced in CI, not just documented. Full table + rationale: `docs/rules.md`.

- **Dependencies point inward.** `controller → usecase → domain`; `infrastructure` implements domain interfaces; never bypass a layer.
- **Domain** is pure: no framework / infrastructure / logging / DI; no env / I/O / DB clients; no dependence on time, randomness, or system state directly (abstract via domain interfaces implemented in outer layers). **No `context.Context` in domain logic** (Repository interface signatures may declare it for propagation only). The only permitted `internal/` dependencies are `internal/apperror` and the domain shared kernel `internal/domain/kernel` (cross-aggregate business-semantic value objects that cannot live in `pkg/`; admission is narrow — see `docs/rules.md` / ADR-0104); a domain package must not import another aggregate. Otherwise use `pkg/`.
- **`pkg/`** must not depend on infrastructure or framework-specific packages, must stay framework-agnostic, must not import `internal/**`, and holds no feature-specific business logic.
- **Usecase** depends only on domain interfaces (never infrastructure), owns transaction boundaries, and maps domain models to DTOs — never exposes domain entities to outer layers.
- **Controller** handlers stay lightweight: request/response only, no business logic, no infrastructure imports.

## Forbidden Shortcuts

AI agents MUST NOT:

- Call infrastructure directly from a handler
- Put business logic in a handler
- Skip the OpenAPI definition for a new API
- Modify generated files
- Introduce new architectural patterns without instruction

## YAGNI vs Regression Safeguards

- **Functional YAGNI applies to production code**: do NOT add speculative features, config, or code
  paths that no caller exercises. Unreached "might-need-it-later" code is dead weight — untested,
  coverage-dragging, and prone to rot.
- **Deliberate regression safeguards are the encouraged exception**: a defensive branch/guard whose
  purpose is to catch a future mistake (e.g. mapping the right value per environment, refusing a
  dangerous operation) SHOULD be kept and **actively locked down with a test**. If it is unreachable
  through its current caller, extract the logic into a testable unit and cover every branch rather
  than deleting it. Never drop a meaningful safeguard just because it is currently unreached — make
  it testable and add the regression test.
- **Coverage % is a proxy, not the goal**: a test's worth is the contract it locks against
  regression, not the number it moves. A meaningful test can add 0 % (e.g. exercising an empty
  no-op default — an `{}` body has zero coverable statements, so its `0.0%` is a Go display artifact,
  NOT a gap). Do NOT chase whole-function `0.0%` that are empty/no-statement bodies, and do NOT
  delete a test that verifies a real contract just because it doesn't raise coverage. Conversely,
  code run only to hit lines with no meaningful assertion is coverage theater — that IS meaningless.
- **A test is meaningful only if** the contract it protects (correctness / invariant / boundary /
  safety) (1) can actually regress, (2) is not already locked elsewhere, and (3) is owned by that
  layer. The meaningless-test forms are the inverses: **wrong semantics** (tautology, asserting an
  incidental implementation detail, or coverage-only), **redundant duplication** (same path verified
  2×/3× across layers with no new viewpoint; re-testing a dependency / generated code), or **wrong
  layer** (verifying a concern the layer does not own).

## AI Modification Scope

AI agents may modify code only in these directories unless explicitly instructed otherwise:

- `internal/`
- `pkg/`
- `database/` (`database/dml/**`; `database/migrations/**` — new files only, never edit existing migrations)
- `openapi/`

Do NOT modify other top-level directories (e.g. `cmd/`, `docker/`, `scripts/`, `docs/`,
`vendor/`, `makefile`) unless the user explicitly requests it.

- **CLI command exception:** each CLI subcommand is a thin `cmd/<command>.go` shell (Cobra + real-dependency wiring) paired with its testable core under `internal/cli/<command>/`. Adding / modifying a command necessarily edits the matching `cmd/<command>.go` and that is in-scope; the restriction is about not arbitrarily restructuring `cmd/` entrypoint/build wiring.

**AI-tool configurations are out of scope** — do NOT create/modify/delete these unless the user explicitly requests it:

- Claude Code: `.claude/` (skills, `settings.json`, `settings.local.json`, …)
- OpenAI Codex CLI: `.agents/skills/`
- Cursor: `.cursor/` (incl. `.cursor/rules/*.mdc`), `.cursorrules`
- GitHub Copilot: `.github/copilot-instructions.md`, `.github/instructions/`, `.github/prompts/`
- Gemini CLI / Code Assist: `.gemini/`, `GEMINI.md`

### Exception: Skill Execution

Invoking a skill (Claude Code `/<skill-name>`, or an equivalent mechanism) counts as an
**explicit user instruction**. While the skill runs, the scope restrictions above are relaxed
for the paths the skill's defined procedure needs. Conditions:

- Relaxed only for the skill's duration and only to the scope the skill defines; the skill's `SKILL.md` still governs (honor any "confirm before touching X" step).
- **Hard-protected even during skill execution:**
  - `AGENTS.md`
  - Generated files: `**/*.gen.go`, `*.sql.go`, `*_mock.go`, `**/openapi.gen.yaml`
  - Generated content under `docs/`: `docs/openapi/**`, `docs/coverage/**`, `docs/db-schema/**`, `docs/godoc/**`, `docs/portal/docs.json`, `docs/portal/guides/**`
  - Anything under `permissions.deny` in `.claude/settings.json`
- Canonical Markdown under `docs/` (`architecture.md`, `rules.md`, `decisions.md`, `development-flow.md`, `testing-conventions.md`, `maintenance/**`, `ja/**`, `portal/manifest.yaml`, …) is NOT generated and remains editable per the skill's scope.
- Skills must not be a loophole. If a procedure touches a sensitive area (`docker/`, `.github/workflows/`), the skill must document it so the user is aware.

## Do Not Edit Generated Files

- `**/*.gen.go`
- `**/*.sql.go`
- `*_mock.go`
- `**/openapi.gen.yaml`
- Generated content under `docs/`: `docs/openapi/**`, `docs/coverage/**`, `docs/db-schema/**`, `docs/godoc/**`, `docs/portal/docs.json`, `docs/portal/guides/**`

## Git Rules for AI Agents

1. **NEVER commit directly** to `production`, `develop`, `staging`, or any `release/*` branch. Always cut a feature branch from the latest `release/*`.
2. Do NOT rebase, squash, or force-push unless the user explicitly requests it.
3. After amending an existing PR branch, do NOT auto-push — ask first: 「変更はローカルにコミット済みです。これらの変更をプルリクエストにプッシュしますか？」
4. **Syncing a feature branch with an advanced base — merge, never rebase (see rule 2).** When the base `release/*` has moved ahead and the feature branch must catch up, fast-forward the local base to its remote and **merge** it into the feature branch (`git merge origin/release/vX.Y.0`), rather than rebasing / force-pushing. Resolve conflicts in generated artifacts (`**/*.gen.*`, `docs/openapi/**`, `openapi/openapi.gen.yaml`, …) by **regenerating from the source of truth** (`make gen-api` / `make gen-query`), not by hand-editing the generated output. Rebase only when the user explicitly requests it.

**Commit / PR execution:** split into scoped commits using the prefix convention
(Feat / Fix / Refactor / Perf / Docs / Test / Build / CI / Chore / Style / Revert), bypass
per-commit hooks during the split then run verification (lint / test / sql-lint / migration
checks) once at the end, add the `Co-Authored-By` footer, and never commit to protected
branches. If your agent provides a dedicated command/skill for this workflow, prefer it over
manual steps (keep the concrete names in your own agent config, not here).

**Branch naming:** include the issue number when provided (`feature/1234-description`);
otherwise a descriptive hyphenated name (`feature/add-authentication-check`).

## Language Rules for AI Agents

Internal reasoning may be in English. **All visible outputs must be in Japanese** unless the
user explicitly requests English — test case names, code comments, PR messages, inline
documentation, and responses to the user.

## Recommended Commands

The **full `make` target registry** is `.makefiles/README.md` (targets grouped by area;
every target is self-documenting, so `make help` lists them). Common ones:

Code generation:

- `make gen-api` — generate API code from the OpenAPI spec (oapi-codegen + mock)
- `make gen-query` — generate SQL query code from SQL files (sqlc)

Format / lint / test:

- `make fix` — auto-format + auto-fix lint (run before committing; then fix what remains)
- `make lint` — Go static analysis (golangci-lint)
- `make test` — run all tests with coverage
- `make md-lint` / `make md-fix` — Markdown lint / auto-fix
- `make sql-lint` / `make sql-fix` — SQL lint / auto-fix

Run / DB:

- `make serve` — start the local dev environment (for runtime / `curl` verification)
- `make db-init` — migrate + seed both the local and test DBs (prerequisite for DB-backed tests)
- `make new-migrate-<name>` — scaffold a new migration (`.up.sql` / `.down.sql`)
- `make job NAME=<job> ARGS="<args>"` — run an application job

**Working in a `git worktree` (DB + serve isolation):** a single shared Postgres (fixed compose
project `gobp-shared`, host 5432) is shared by all worktrees; each leases a slot = its own
databases (`wt<N>_local` / `wt<N>_test`) inside that instance. Before DB-backed tasks or `make serve`
in a worktree, `make slot-acquire` to lease a slot (creates + rebuilds `wt<N>_local` / `wt<N>_test`,
propagates `DB_NAME_LOCAL` / `DB_NAME_TEST` / `API_HOST_PORT` `8080+N` / `MOCK_AUTH_HOST_PORT` `4000+N`),
then `make test` connects to `wt<N>_test` on localhost:5432, `make serve` isolates the app in
`gobp-wt-N` (curl `localhost:$API_HOST_PORT`) against the shared DB, and `make slot-free` when done —
do NOT start a duplicate DB stack or hijack another checkout's containers. To retire the worktree
entirely, `make slot-release` stops the app + removes its local images, frees the slot, and removes
the worktree, in that order. Without `slot-acquire`,
targets default to `local` / `test` on 5432 / 8080 / 4000 (single-stack, unchanged).
Details: `docs/maintenance/db-worktree-pool.md`.

**DB clean-up (worktree slot pool):** the pool shares one Postgres instance, so tables from another
branch's migrations can linger in a DB you reuse. As part of clean-up — at the start of DB-backed
work, and whenever the shared DB carries stale tables — rebuild your DB from THIS branch's migrations
so you always work against a clean, migration-faithful schema: `make slot-acquire` re-creates the
slot's `wt<N>` DBs this way, and `make db-local-reinit` / `db-test-reinit` drop every `public` table
then migrate-up + seed the shared `local` / `test`.

## Protected Documentation

`AGENTS.md` must be maintained by humans only. AI agents must NOT modify it unless explicitly
instructed by a human; changes must be intentional and reviewed carefully, as it defines
repository-wide development rules.
