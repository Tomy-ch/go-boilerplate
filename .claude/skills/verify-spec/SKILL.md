---
name: verify-spec
description: Integrator skill for spec validation. Confirms feature name via `AskUserQuestion` (or receives from `scaffold-endpoint`), detects which spec files exist under `docs/spec/<feature>/` (lean A: `domain.md` and/or `usecase.md`), then chains the matching per-spec skills (`verify-spec-domain` / `verify-spec-usecase`) — each handles format check, internal consistency, plus its specific cross-validation (entity ↔ SQL for domain; cross-spec refs + naming convention for usecase). Does NOT check OpenAPI operationId coverage — that violates dependency direction (usecase doesn't know about HTTP/OpenAPI); the OpenAPI ↔ usecase mapping is verified by `scaffold-controller` at scaffold time. Aggregates findings into a single Japanese report. Read-only orchestration — all checks delegated to per-spec skills which are themselves read-only. When chained from `scaffold-endpoint`, aborts the downstream chain on `violation`; standalone exits 0 (informational).
---

# Verify Spec

Integrator for spec validation. Chains per-spec skills based on which spec files exist under `docs/spec/<feature>/`.

A Japanese reference translation of this skill is available at `SKILL.ja.md` in the same directory (not loaded as a skill; for human reference only).

## When to Use

- Before invoking `scaffold-endpoint` to catch spec inconsistencies upfront (`scaffold-endpoint` auto-chains this).
- Standalone after editing specs, to confirm all checks pass.
- During spec authoring as a quick check.

Use the per-spec skill directly (`verify-spec-domain` / `verify-spec-usecase`) when you only care about one spec.

Do NOT use for:

- Verifying generated code — that's `make test`.
- Implementation ↔ spec drift — that's `arch-check`.
- Fixing inconsistencies — read-only, reports only.

## Per-Spec Skills Chained

| Skill | Spec | Checks |
| --- | --- | --- |
| `verify-spec-domain` | `docs/spec/<feature>/domain.md` | format + entity ↔ SQL soft + internal consistency |
| `verify-spec-usecase` | `docs/spec/<feature>/usecase.md` | format + cross-spec to domain + 命名規約 + Workflow consistency |

lean A 構成では controller.md / infra.md は存在しないため、それらの spec 検証は不要（controller / infra は実装時に OpenAPI + sqlc gen から導出され、verify は `arch-check-controller` / `arch-check-infra` が implementation 側で実施）。

## First Step: Confirm Target Feature

This skill **MUST call `AskUserQuestion` immediately after invocation** (unless invoked from `scaffold-endpoint` with the feature name already in context):

- 質問: 「検証対象の feature 名を選んでください」
- 選択肢: `docs/spec/` 直下のサブディレクトリを列挙 + 規約外パス用のフリーテキスト

If the feature directory is missing or contains no spec files, abort with clear message.

## Step 1. Detect Existing Spec Files

For the confirmed feature, check existence of:

- `docs/spec/<feature>/domain.md`
- `docs/spec/<feature>/usecase.md`

If neither exists → abort with message.

If only one exists → chain only the matching per-spec skill. Note that cross-spec checks (e.g., usecase → domain refs) will surface as "domain.md not found" if `usecase.md` exists alone.

## Step 2. Chain Per-Spec Skills

In order (domain first since usecase references it):

1. If `domain.md` exists → invoke `verify-spec-domain` with feature name supplied
2. If `usecase.md` exists → invoke `verify-spec-usecase` with feature name supplied

Each child returns findings (violations + suggestions counts). Collect with spec label.

## Step 3. Aggregate Report (Japanese)

```text
verify-spec 統合結果（feature: <feature>）

[domain] violations: N, suggestions: K
  - <findings from verify-spec-domain>

[usecase] violations: N, suggestions: K
  - <findings from verify-spec-usecase>

総計: violations <sum>, suggestions <sum>
```

All clean:

```text
verify-spec 統合結果（feature: <feature>）
全 spec で違反は検出されませんでした（チェック済み: <spec list>）。
```

## Step 4. Closing

- **Standalone invocation**: print the report and exit. Even with violations, exit status is 0 (informational).
- **Chained from `scaffold-endpoint`**: when `violations > 0`, the parent skill must abort the downstream chain. Surface a clear "scaffold can not safely proceed" message.

## AI Modification Scope

Strictly read-only. Touches no spec or source files. All checks delegated to per-spec skills (themselves read-only).

## Constraints

- ❌ Hardcode rules — always delegate to per-spec skills (which read `.claude/scaffold-spec/<layer>-spec.md` + `verify-rules.md`)
- ❌ Auto-fix violations
- ❌ Modify any file
- ❌ Skip the target-confirmation `AskUserQuestion`
- ❌ Chain a per-spec skill when its target file is missing
- ✅ Japanese aggregated report
- ✅ Chain only existing spec files
- ✅ Per-spec skill が独立 standalone 動作可能であることを維持
- ✅ Run all per-spec checks in one pass (no fail-fast)

## Checklist

- [ ] Target feature confirmed via `AskUserQuestion` (or supplied by `scaffold-endpoint`)
- [ ] Existing spec files detected (domain.md / usecase.md)
- [ ] Per-spec skills chained for existing files only
- [ ] Each child ran its own validation
- [ ] Aggregated Japanese report emitted
- [ ] No file modifications
