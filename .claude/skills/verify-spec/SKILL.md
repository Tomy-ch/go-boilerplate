---
name: verify-spec
description: Integrator skill for spec validation. Confirms feature name via `AskUserQuestion` (or receives from `scaffold-endpoint`), detects which spec files exist under `docs/spec/<feature>/` (lean A: `domain.md` and/or `usecase.md`), then fans out the matching read-only `spec-validator-*` subagents (`spec-validator-domain` / `spec-validator-usecase`) IN PARALLEL via the Agent tool — passing the feature name + spec paths so each validator skips its own feature confirmation. Each validator handles format check, internal consistency, plus its specific cross-validation (entity ↔ SQL for domain; cross-spec refs + naming convention for usecase). Does NOT check OpenAPI operationId coverage — that violates dependency direction (usecase doesn't know about HTTP/OpenAPI); the OpenAPI ↔ usecase mapping is verified by `scaffold-controller` at scaffold time. Aggregates findings into a single Japanese report. Read-only orchestration — validators never touch spec or source files. When chained from `scaffold-endpoint`, aborts the downstream chain on `violation`; standalone exits 0 (informational). To validate a single spec, run this integrator — it fans out only the validator(s) for the spec file(s) that exist.
---

# Verify Spec

Integrator for spec validation. Fans out per-spec **read-only validator subagents** in parallel based on which spec files exist under `docs/spec/<feature>/`.

A Japanese reference translation of this skill is available at `SKILL.ja.md` in the same directory (not loaded as a skill; for human reference only).

## When to Use

- Before invoking `scaffold-endpoint` to catch spec inconsistencies upfront (`scaffold-endpoint` auto-chains this).
- Standalone after editing specs, to confirm all checks pass.
- During spec authoring as a quick check.

To validate a single spec, run this integrator — it detects which of `domain.md` / `usecase.md` exist and fans out only the matching validator.

Do NOT use for:

- Verifying generated code — that's `make test`.
- Implementation ↔ spec drift — that's `arch-check`.
- Fixing inconsistencies — read-only, reports only.

## Architecture: parallel validator subagents

Validation is delegated to two **read-only worker subagents** under `.claude/agents/`, one per spec file. The integrator runs them concurrently via the Agent tool (`subagent_type`):

| Validator subagent | Spec | Checks |
| --- | --- | --- |
| `spec-validator-domain` | `docs/spec/<feature>/domain.md` | format + entity ↔ SQL soft + internal consistency |
| `spec-validator-usecase` | `docs/spec/<feature>/usecase.md` | format + cross-spec to domain + 命名規約 + Workflow consistency |

lean A 構成では controller.md / infra.md は存在しないため spec 検証は不要（controller / infra は実装時に OpenAPI + sqlc gen から導出され、verify は `arch-check`（controller / infra 監査）が implementation 側で実施）。

The validators are the per-spec validation workers and are **strictly read-only** (no auto-fix, no writes). The two validators read independently — `spec-validator-usecase` reads `domain.md` itself for cross-spec `calls:` resolution — so there is **no write dependency** between them and they can run in parallel (the old domain-first ordering was a read reference, not a barrier).

## First Step: Confirm Target Feature

This skill **MUST call `AskUserQuestion` immediately after invocation** (unless invoked from `scaffold-endpoint` with the feature name already in context):

- 質問: 「検証対象の feature 名を選んでください」
- 選択肢: `docs/spec/` 直下のサブディレクトリを列挙 + 規約外パス用のフリーテキスト

If the feature directory is missing or contains no spec files, abort with a clear message.

## Step 1. Detect Existing Spec Files

For the confirmed feature, check existence of:

- `docs/spec/<feature>/domain.md`
- `docs/spec/<feature>/usecase.md`

If neither exists → abort with message. If only one exists → fan out only the matching validator. (If `usecase.md` exists alone, its cross-spec check will surface "domain.md not found" as a `violation`.)

## Step 2. Fan Out Validator Subagents IN PARALLEL

For the existing spec files, spawn the matching validators with the **Agent tool**, all in **a single message with multiple tool calls** so they run concurrently. Pass each validator:

- `feature` — the confirmed feature name
- `specPath` — the spec file path (`docs/spec/<feature>/domain.md` or `.../usecase.md`)

Each validator's final message **is** its findings (Japanese), ending in a machine-readable `SUMMARY violations=<v> suggestions=<s>` line. Collect them with their spec label and parse the SUMMARY counts.

> If the `spec-validator-*` subagents cannot be spawned in the current environment, follow each `spec-validator-<layer>.md` procedure inline instead (domain first, since usecase references it).

## Step 3. Aggregate Report (Japanese)

```text
verify-spec 統合結果（feature: <feature>）

[domain] violations: N, suggestions: K
  - <findings from spec-validator-domain>

[usecase] violations: N, suggestions: K
  - <findings from spec-validator-usecase>

総計: violations <sum>, suggestions <sum>
```

All clean:

```text
verify-spec 統合結果（feature: <feature>）
全 spec で違反は検出されませんでした（チェック済み: <spec list>）。
```

## Step 4. Closing

- **Standalone invocation**: print the report and exit. Even with violations, exit status is 0 (informational).
- **Chained from `scaffold-endpoint`**: when aggregated `violations > 0`, signal the parent to abort the downstream chain with a clear "scaffold can not safely proceed" message. (Suggestions do not abort.)

## AI Modification Scope

Strictly read-only. The integrator and all validator subagents touch no spec or source files. The integrator only runs `AskUserQuestion` (feature confirmation, standalone) and spawns read-only validators.

## Constraints

- ❌ validator を逐次起動（必ず1メッセージ内で複数 Agent 呼び出し＝並列）
- ❌ Hardcode rules — validators read `.claude/scaffold-spec/<layer>-spec.md` + `verify-rules.md` every run
- ❌ Auto-fix violations / modify any file
- ❌ Skip the target-confirmation `AskUserQuestion` (unless supplied by `scaffold-endpoint`)
- ❌ Fan out a validator when its target spec file is missing
- ✅ Japanese aggregated report
- ✅ Fan out only existing spec files
- ✅ Per-spec validator / skill が独立 standalone 動作可能であることを維持
- ✅ Run all per-spec checks in one pass (no fail-fast); abort downstream only when chained from `scaffold-endpoint` with violations

## Checklist

- [ ] Target feature confirmed via `AskUserQuestion` (or supplied by `scaffold-endpoint`)
- [ ] Existing spec files detected (domain.md / usecase.md)
- [ ] Matching `spec-validator-*` を **1メッセージ内で並列起動**（feature / specPath を渡す）
- [ ] 各 validator の SUMMARY を集約
- [ ] Aggregated Japanese report emitted
- [ ] scaffold-endpoint から chain 時のみ violations>0 で downstream abort
- [ ] No file modifications
