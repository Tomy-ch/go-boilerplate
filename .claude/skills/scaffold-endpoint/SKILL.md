---
name: scaffold-endpoint
description: Orchestrator skill that scaffolds a complete onion-architecture endpoint (domain + infra-db + usecase + controller) for one feature. **lean A constitution**: only `domain.md` + `usecase.md` are spec-driven; infra-db is derived from domain Repository IF + sqlc gen, and controller is derived from OpenAPI gen + usecase Interface (no `controller.md` / `infra.md` spec files). Confirms the feature name via `AskUserQuestion`, then chains: (1) `verify-spec` to validate the 2 spec files (domain + usecase) format + cross-spec references — aborts the chain on any violation; (2) `scaffold-domain` — entity / Repository IF / VOs / constants / errors / tests; (3) `scaffold-infra-db` — Repository impl wrapping sqlc gen; (4) `scaffold-usecase` — Application Service implementation; (5) `scaffold-controller` — handler implementing the OpenAPI-generated interface (halts with hand-off if any operationId can't map to a usecase method); (6) final `make fix` + `make test` to confirm cross-layer integrity. Each child skill runs with its own confirmations and test-perspective subagent. Failure halts the chain with a clear "scaffold can not safely proceed" message but does NOT auto-rollback. Prerequisites: OpenAPI YAML, SQL migrations, domain.md, usecase.md all written by the user, and `make gen-query` + `make gen-api` have been run.
---

# Scaffold Endpoint

Top-level orchestrator that builds a complete onion-architecture endpoint by chaining the per-layer scaffold skills in the correct dependency order. Use this when starting a new feature end-to-end after the OpenAPI + SQL + spec inputs are ready.

A Japanese reference translation of this skill is available at `SKILL.ja.md` in the same directory (not loaded as a skill; for human reference only).

## When to Use

- Starting a new feature end-to-end with the 2 specs (`domain.md` + `usecase.md`) + OpenAPI YAML + SQL prepared.
- Want all layers built consistently with the same conventions and one consolidated FB report at the end.

Do NOT use this skill for:

- Modifying an existing feature — pick the specific layer skill (`scaffold-domain` / `-infra-db` / `-usecase` / `-controller`) and run standalone.
- Bootstrapping the input artifacts (OpenAPI YAML, SQL, specs) — those are human-written preconditions.

## What This Skill Reads / Writes

**Reads (always)**:

- `docs/spec/<feature>/{domain,usecase}.md` — via child skills (lean A: 2 spec files only).
- OpenAPI gen + sqlc gen + domain Repository IF — derivation sources for controller / infra (no spec).
- `.claude/scaffold-spec/lifecycle.md` — for the canonical workflow / scaffold execution order.

**Writes**: nothing directly. All writes happen inside the chained child skills, each within its own scope.

## Preconditions (must be true before invocation)

| # | Precondition | Verifier |
| --- | --- | --- |
| 1 | `domain.md` + `usecase.md` exist under `docs/spec/<feature>/` (lean A: 2 spec files only) | verify-spec |
| 2 | Spec format valid + cross-spec references consistent + naming convention satisfied | verify-spec |
| 3 | OpenAPI YAML written and `make gen-api` produced `internal/controller/handler/<path>/gen/` | scaffold-controller precondition |
| 4 | SQL files under `database/dml/...` written and `make gen-query` produced sqlc gen files | scaffold-infra-db precondition |
| 5 | DB running + migrated + seeded (DB schema matches the new SQL) | manual: start env with `make serve`, then `make db-init` |
| 6 | Boundary interfaces the usecase spec depends on exist under `internal/usecase/boundary/` | scaffold-usecase precondition |

If any precondition fails, the relevant child skill will surface it and this skill aborts the chain.

> **Environment note (preconditions 3–5):** `make gen-query` dumps the live DB schema via `pg_dump`, so the database **must be running** — `make gen-query` (and `make test`) fail with `could not translate host name "database"` when it is not. Bring the environment up with the **dedicated make targets, not raw `docker compose`**: `make serve` (starts the development profile incl. the `database` service), then **`make db-init`** — which migrates **and seeds** both the local and test DBs (the test suite assumes seed data exists; piecemeal `db-*-migrate-up` alone is insufficient) — and only then `make gen-query` / `make gen-api`.
>
> **Toolchain note (final `make fix` / `make test`):** if `make fix` or `make lint` fails on a **tool version mismatch** (e.g. `golangci-lint` reporting "you are using a configuration file for golangci-lint v2 with golangci-lint v1"), do **not** work around it — align the local toolchain with `make install-tools` (installs the versions pinned in `tools.yaml`; run `make sync-tools` first if `tools.yaml` itself changed), then re-run. Do not hand-edit `PATH` or invoke version-specific binaries as a substitute.

## First Step: Confirm Feature

This skill **MUST call `AskUserQuestion` immediately after invocation**:

- Question: 「対象 feature 名 (kebab-case)」
- Free-text. The skill validates `docs/spec/<feature>/` exists and has at least `domain.md` + `usecase.md` (lean A required minimum).

If the feature directory is missing or specs are incomplete, suggest the user run `/new-spec` (integrator) to scaffold the 2 spec set.

## Step 1. Verify Specs (auto-chain)

Invoke the `verify-spec` skill with the feature name. If `verify-spec` reports `violations > 0`, abort the chain:

```text
scaffold can not safely proceed: verify-spec で <N> 件の違反が検出されました。
spec を修正してから再度 /scaffold-endpoint を実行してください。
```

If only warnings are reported, continue (warnings do not block).

## Step 2. Chain Child Skills in Dependency Order

Invoke each child skill in turn, passing the feature name in context so each child can resolve its spec path automatically:

1. **`scaffold-domain`** — entity + Repository IF + VOs + constants + errors + tests (+ `make gen-api` for mock).
2. **`scaffold-infra-db`** — Repository impl wrapping sqlc gen (requires `make gen-query` already run, verified internally).
3. **`scaffold-usecase`** — Application Service + DTOs + tests (+ `make gen-api` for Usecase mock).
4. **`scaffold-controller`** — Handler implementing `ServerInterface` + tests.

Between each child skill, propagate the success / failure status:

- **Child success** → proceed to next.
- **Child failure** → halt the chain, surface the child's FB summary, do NOT proceed.

Each child skill independently:

- Asks its own plan confirmation `AskUserQuestion` (the user approves each layer's plan separately to keep judgment per-layer)
- Invokes its own test-perspective subagent
- Runs `make gen-api` if needed
- Writes its own files
- Runs `make fix` + `make test` after its writes
- Surfaces TODO + FB on failure

If you want a "fully unattended" mode, the user can add `--auto-approve` (future flag) — but the default is to confirm each layer to keep human-in-the-loop on judgment-heavy steps.

## Step 3. Final Integration Verification

After all 4 child skills succeed, run a final consolidated check:

```sh
make fix
make test
```

This confirms the cross-layer integration (handler → usecase → domain → infra) compiles and tests pass as a whole. Surface the per-package coverage line for the 4 packages this scaffold touched.

If `make test` fails at this final step (rare — child skills already ran their own), surface the failure with TODO + FB and stop.

## Step 3.5. Runtime Verification (curl + o11y)

`make test` (Step 3) uses **mocked usecases/repositories**, so it does NOT construct the real Fx graph, run the HTTP middleware (auth / OpenAPI validation), or touch the DB. A whole class of bugs only surfaces at runtime: a missing `security:` declaration (endpoint reachable without auth), an unregistered/mis-wired `BindHandler`, a DI provider mismatch, or a SQL filter that behaves differently against a real DB. **This is the correct place to curl** — all layers + DI now exist. Per-layer skills cannot do this (their lower layers / DI may be absent, so the app would not even boot).

Preconditions:

- `make serve` is running and the `api_server` log reached `[Fx] RUNNING` (the DI boot check from `scaffold-controller`).
- `make db-init` has run (seeds local + test DB — the canonical setup).

Steps:

1. **Pick or create a target in a known state.** If the operation needs an existing row, use a seeded id or create one first via the create endpoint. For credential/state-sensitive checks (e.g. password change), create a row whose plaintext/state you control (a `psql` insert with a known bcrypt hash is acceptable for local verification).
2. **curl the new endpoint(s)** (auth header for local: `Authorization: Bearer debug:<subject>`) and assert:
   - the route is reachable — a non-404-from-router response proves the handler is registered and DI is wired;
   - the happy path returns the expected status/body;
   - the key error paths: NotFound (404), validation (400/422), and — **if the operation declares `security:`** — no-token ⇒ 401 (verify the endpoint is actually protected);
   - the mutation **actually took effect** (re-read / re-auth with the new state), not merely a 2xx.
3. **Read the o11y logs once** for a single request: confirm the trace spans the full stack (controller → usecase → infra) and the emitted SQL is what you expect. After this one capture, further re-checks can rely on o11y instead of re-curling.

Shared-schema impact: if the change edits a **shared** OpenAPI component (e.g. a `components/schemas/*` or `components/requests/*` referenced by more than one operation), the runtime check must cover **every** endpoint that references it — not just the new/changed one. Grep the spec for `$ref`s to the edited file and curl each consumer. A shared-schema edit can silently break sibling endpoints the mocked tests never exercise (e.g. removing a property from a base used via `allOf`, or `additionalProperties: false` rejecting a sibling's property → `POST` create breaks while the new endpoint looks fine).

Destructive guard: if a curl mutates data and the only way to restore the prior state is `make db-init` (or equivalent), **confirm with the user before running it** (per `CLAUDE.md`). Clean up any rows you created for verification.

If any check fails, surface TODO + FB and stop (do NOT commit).

## Step 4. Closing

Print a Japanese summary:

```text
scaffold-endpoint 完了（feature: <feature>）。

  ✓ verify-spec: violations 0
  ✓ scaffold-domain: <N> ファイル作成、coverage 100%
  ✓ scaffold-infra-db: <N> ファイル作成、coverage <X>%
  ✓ scaffold-usecase: <N> ファイル作成、coverage 100%
  ✓ scaffold-controller: <N> ファイル作成、coverage 100%
  ✓ make test: 全体 OK
  ✓ ランタイム動作確認: curl 到達 / 認証 / 主要異常系 / o11y トレース OK

次のアクション:
  - /commit で 4 層 + DI 変更をコミット
  - /submit-pr で PR 作成
```

If any step failed, output the failure status table with the FB from the failing child, and the user decides whether to fix forward.

Do NOT commit. Do NOT push.

## AI Modification Scope

This skill itself writes no files. All scope is delegated to child skills, each within their own constraints (see their SKILL.md).

## Constraints

- ❌ Modify any source file directly (delegate to child skills)
- ❌ Skip `verify-spec` (Step 1) — it is the safety net for spec consistency
- ❌ Proceed past a failing child skill — halt and surface FB
- ❌ Auto-rollback files written by a successful earlier child when a later child fails — the user decides
- ❌ Skip the feature-confirmation `AskUserQuestion`
- ✅ Japanese user-facing output
- ✅ Run child skills in the documented dependency order (domain → infra-db → usecase → controller)
- ✅ Surface a consolidated final report covering all 5 chained steps + the final `make test`
- ✅ Let each child skill ask its own confirmation per layer (human-in-the-loop on judgment-heavy steps)
- ✅ Run the runtime curl + o11y verification (Step 3.5) — `make test` alone does not exercise DI / middleware / DB
- ✅ Confirm with the user before any destructive curl whose only restore path is `make db-init`

## Checklist

Before reporting completion, confirm:

- [ ] Feature name confirmed via `AskUserQuestion`
- [ ] `verify-spec` ran; chain aborted if any violation
- [ ] `scaffold-domain` ran successfully (or failed and chain halted)
- [ ] `scaffold-infra-db` ran successfully (or failed and chain halted)
- [ ] `scaffold-usecase` ran successfully (or failed and chain halted)
- [ ] `scaffold-controller` ran successfully (or failed and chain halted)
- [ ] Final `make fix` + `make test` run after all child skills
- [ ] Runtime verification (Step 3.5): curl reachability / auth / key error paths / o11y trace confirmed
- [ ] Consolidated Japanese summary with per-layer file counts and coverage
- [ ] No commits / pushes
- [ ] On any child failure, this skill did NOT auto-rollback already-written files
