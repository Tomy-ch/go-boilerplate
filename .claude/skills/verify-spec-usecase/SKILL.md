---
name: verify-spec-usecase
description: Validate `docs/spec/<feature>/usecase.md` for format correctness, cross-spec references to `domain.md`, naming convention, and internal consistency before any scaffold-usecase run. Confirms feature name via `AskUserQuestion` (or receives from `verify-spec` integrator), reads `.claude/scaffold-spec/usecase-spec.md` for required section list + `.claude/scaffold-spec/verify-rules.md` for verification scope + `internal/usecase/README.md` for naming convention, then performs: (1) format check — required H2 sections present, every YAML code block parses; (2) cross-spec reference check — `calls:` entries resolve to `domain.md` Repository Methods + Behavior Methods + boundary in `Dependencies`; (3) naming convention check (lean A) — Usecase interface name + method names follow the project's convention (verb-prefix unified, e.g., List / Create / Get / Update / Delete) so downstream scaffold can derive mappings deterministically; (4) Workflow internal consistency (tx_required + boundary calls correctness). Does NOT check OpenAPI operationId coverage — that violates dependency direction (usecase doesn't know about HTTP/OpenAPI); the OpenAPI ↔ usecase mapping is verified by `scaffold-controller` at scaffold time. Reports violations + suggestions in a single pass; no auto-fix. Read-only. Standalone-callable; chained from `verify-spec` integrator skips feature-confirmation.
---

# Verify Spec — Usecase

Validate `docs/spec/<feature>/usecase.md` for format + cross-spec refs + naming convention. OpenAPI ↔ usecase mapping is verified by `scaffold-controller` at scaffold time (usecase doesn't know about HTTP/OpenAPI — dependency direction).

A Japanese reference translation of this skill is available at `SKILL.ja.md` in the same directory (not loaded as a skill; for human reference only).

## When to Use

- Before invoking `scaffold-usecase` to catch spec inconsistencies upfront.
- Standalone after editing `usecase.md`.
- Chained from `verify-spec` integrator.

Do NOT use for:

- Implementation ↔ spec drift — that's `arch-check-usecase`.
- Fixing inconsistencies — read-only, reports only.

## Source of Truth (read every run)

| Source | Purpose |
| --- | --- |
| `.claude/scaffold-spec/usecase-spec.md` | Required H2 sections + YAML schema for usecase.md |
| `.claude/scaffold-spec/verify-rules.md` | Verification scope (format + cross-spec + naming) |
| `docs/spec/<feature>/usecase.md` | The spec file under validation |
| `docs/spec/<feature>/domain.md` | Referenced for cross-spec `calls:` resolution |
| `internal/usecase/README.md` | Naming convention (verb-prefix, Usecase interface naming) |
| `internal/usecase/<sibling>/*.go` | Fallback for naming convention if README is silent — observe existing patterns |

## First Step: Confirm Target Feature

If standalone, `AskUserQuestion`:

- 質問: 「検証対象の feature 名を選んでください」
- 選択肢: `docs/spec/` 直下のサブディレクトリを列挙 + 規約外パス用のフリーテキスト

If chained from `verify-spec` with feature name supplied, skip.

If `docs/spec/<feature>/usecase.md` is missing, abort with clear message.

## Step 1. Format Check

1. Read `.claude/scaffold-spec/usecase-spec.md` for required H2 section list.
2. Verify every required H2 section is present in `usecase.md`.
3. Parse every fenced YAML code block. Any YAML parse error → `violation`.
4. For each Interface method YAML, check required keys (`name`, `signature`).
5. For each Workflow entry YAML, check required keys (`tx_required`, `steps`, `calls`, `errors`).
6. For each Dependency YAML, check that the entry is a recognized boundary or Repository reference.

## Step 2. Cross-Spec Reference Check

Read `docs/spec/<feature>/domain.md` and build inventory:

| Inventory | Source |
| --- | --- |
| `domain.repository_methods` | `domain.md` Repository Methods (name list) |
| `domain.behavior_methods` | `domain.md` Behavior Methods (name list) |
| `domain.factory` | `domain.md` Entity struct name + Value Objects factory names |
| `usecase.dependencies` | `usecase.md` Dependencies (boundary names) |

Then for each `usecase.md` Workflow entry's `calls:` list, classify each call:

- `<aggregate>.Repository.<Method>` → must exist in `domain.repository_methods` → else `violation`
- `<aggregate>.<BehaviorMethod>` or `<aggregate>.New` → must exist in `domain.behavior_methods` or `domain.factory` → else `violation`
- `<boundary>.<Method>` (e.g., `clock.Now`, `tx.Do`) → boundary must appear in `usecase.dependencies` → else `violation` (method itself is compile-time)

## Step 3. Naming Convention Check (lean A)

Why this step exists: scaffold-controller mechanically derives OpenAPI operationId → usecase method mapping at scaffold time. For that derivation to work deterministically, usecase methods must follow a consistent naming convention. This check verifies the spec **without** referencing OpenAPI itself (which would violate dependency direction — usecase doesn't know about HTTP).

Source of truth (read in order):

1. `internal/usecase/README.md` — explicit naming convention if documented
2. Existing `internal/usecase/<sibling>/*.go` — observed patterns (action verbs used in current code base) as fallback

Then for `usecase.md` Interface:

- **Usecase interface name**: should follow project convention (typically `Usecase` per package, or as documented)
- **Method names**: should use a recognized action verb prefix (e.g., `List`, `Create`, `Get`, `Update`, `Delete`, `Register`, `Activate`, `Deactivate` etc. — derived from README/sibling, not hardcoded here)
- **No body/HTTP terminology in method names** (e.g., `Post`, `Put`, `Patch` → suggest renaming to domain action verb)

Each finding → `suggestion`（規約への準拠は推奨、blocker ではない。命名規約違反は scaffold-controller 側で mapping 失敗として最終的に surface される）。

## Step 4. Workflow Internal Consistency

- `tx_required: true` の Workflow entry が `tx.Manager` boundary を `calls` に含むか確認
- `errors` リストが domain で定義された error 型を参照しているか（部分一致でいい、命名規則チェック）

## Step 5. Report (Japanese)

```text
verify-spec-usecase 結果（feature: <feature>）

[format] N 件
  - usecase.md: 必須節 "Workflow" が見つからない
  - usecase.md L42 YAML パースエラー: ...

[cross-spec] M 件
  - usecase.md CreateUser calls 'user.Repository.Save' が domain.md Repository Methods に存在しない
  - usecase.md ActivateUser calls 'clock.Now' だが Dependencies に clock 無し

[naming convention] K 件（suggestion）
  - usecase Interface method `PostUser` は HTTP verb 由来命名
    source: internal/usecase/README.md / 既存 sibling pkg のパターン
    remediation: `CreateUser` 等の action verb prefix に rename 推奨

[internal] L 件
  - Workflow `Register` の tx_required:true だが calls に tx.Manager 無し

総計: violations <N+M>, suggestions <K+L>
```

Empty:

```text
verify-spec-usecase 結果（feature: <feature>）
usecase.md の違反は検出されませんでした（suggestions: 0）。
```

## Step 6. Closing

- 単独実行: exit 0（情報的）
- `verify-spec` から chain: 違反数 + suggestion 数を caller に返す
- 自動修正しない

## AI Modification Scope

Strictly read-only. No file modifications.

## Constraints

- ❌ Auto-fix violations
- ❌ Modify any spec or source file
- ❌ Hardcode the section list (always read `.claude/scaffold-spec/usecase-spec.md`)
- ❌ Skip the target-confirmation `AskUserQuestion` when standalone
- ❌ Check OpenAPI operationId coverage — that violates dependency direction (usecase doesn't know about HTTP/OpenAPI). OpenAPI ↔ usecase mapping is `scaffold-controller`'s responsibility.
- ❌ Treat naming convention violations as hard violations (always `suggestion`)
- ✅ Japanese output
- ✅ Cite source-of-truth document + line
- ✅ Run all checks in one pass

## Checklist

- [ ] Target feature confirmed or supplied
- [ ] `.claude/scaffold-spec/usecase-spec.md` read this run
- [ ] `usecase.md` format checked (sections + YAML)
- [ ] `domain.md` read for cross-spec inventory
- [ ] Cross-spec `calls:` references validated
- [ ] Naming convention checked (from `internal/usecase/README.md` + sibling pkg patterns)
- [ ] Workflow internal consistency checked
- [ ] Findings cite source-of-truth
- [ ] Report Japanese
- [ ] No files modified
