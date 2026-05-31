---
name: verify-spec-domain
description: Validate `docs/spec/<feature>/domain.md` for format correctness and entity ↔ SQL migration correspondence before any scaffold-domain run. Confirms feature name via `AskUserQuestion` (or receives from `verify-spec` integrator), reads `.claude/scaffold-spec/domain-spec.md` for the required section list + `.claude/scaffold-spec/verify-rules.md` for verification scope, then performs: (1) format check — required H2 sections present, every YAML code block parses; (2) entity ↔ SQL soft check — Entity field names in spec match `database/migrations/*.sql` `CREATE TABLE` columns (snake↔camel), with method-form values / VO wrapping auto-recognized as legitimate; (3) Behavior Methods / Value Objects / Repository Methods YAML internal consistency. Reports all violations + suggestions in a single pass; no auto-fix. Read-only — touches no spec or source files. Standalone-callable; chained from `verify-spec` integrator skips feature-confirmation.
---

# Verify Spec — Domain

Validate `docs/spec/<feature>/domain.md` for format + entity ↔ SQL alignment.

A Japanese reference translation of this skill is available at `SKILL.ja.md` in the same directory (not loaded as a skill; for human reference only).

## When to Use

- Before invoking `scaffold-domain` to catch spec inconsistencies upfront.
- Standalone after editing `domain.md`.
- Chained from `verify-spec` integrator.

Do NOT use for:

- Implementation ↔ spec drift — that's `arch-check-domain`.
- Fixing inconsistencies — read-only, reports only.

## Source of Truth (read every run)

| Source | Purpose |
| --- | --- |
| `.claude/scaffold-spec/domain-spec.md` | Required H2 sections + YAML schema for domain.md |
| `.claude/scaffold-spec/verify-rules.md` | Verification scope (format + spec ↔ derivation source) |
| `docs/spec/<feature>/domain.md` | The spec file under validation |
| `database/migrations/*.sql` | `CREATE TABLE` for entity ↔ column check |

## First Step: Confirm Target Feature

If standalone, `AskUserQuestion`:

- 質問: 「検証対象の feature 名を選んでください」
- 選択肢: `docs/spec/` 直下のサブディレクトリを列挙 + 規約外パス用のフリーテキスト

If chained from `verify-spec` with feature name supplied, skip.

If `docs/spec/<feature>/domain.md` is missing, abort with clear message.

## Step 1. Format Check

1. Read `.claude/scaffold-spec/domain-spec.md` for the required H2 section list.
2. Verify every required H2 section is present in `domain.md`.
3. Parse every fenced YAML code block. Any YAML parse error → `violation`.
4. For each Entity field YAML, check required keys (`name`, `type`).
5. For each Behavior Method YAML, check required keys (`name`, `signature`).
6. For each Repository Method YAML, check required keys (`name`, `signature`).

## Step 2. Entity ↔ SQL Soft Check

Read `database/migrations/*.sql`, find `CREATE TABLE <aggregate_plural>` matching the Entity struct name in `domain.md`. Then:

- Map `snake_case` columns ↔ `camelCase` field names in Entity YAML.
- **Auto-recognized legitimate divergences** (no finding):
  - Method-form values (declared in Behavior Methods, not Entity) — naturally excluded
  - VO type fields that wrap multiple SQL columns (resolve VO from Value Objects YAML, treat wrapped columns as covered)
- **Report as `suggestion`** (not `violation`):
  - SQL column without matching Entity field or VO-wrapped equivalent → 「永続化されないカラム」
  - Entity field with no matching column and no VO resolution → 「SQL 対応カラムなし、計算値ならメソッド形式に書き換え推奨」
  - Type mismatch heuristic (`VARCHAR` vs `int` 等) → 確認推奨

## Step 3. Internal Consistency Check

- Behavior Methods が Entity fields を参照する場合（signature 内）、対応 field が Entity に存在するか
- Value Objects の factory が他 VO を参照する場合、その VO が定義されているか
- Cross-field Invariants で言及する field が Entity に存在するか

## Step 4. Report (Japanese)

```text
verify-spec-domain 結果（feature: <feature>）

[format] N 件
  - domain.md: 必須節 "Behavior Methods" が見つからない
  - domain.md L42 YAML パースエラー: ...

[entity ↔ SQL] K 件（suggestion only）
  - domain.md Entity に `phoneNumber` フィールドあり、SQL カラム未定義
    remediation: 計算値ならメソッド形式に書き換え、永続化必要なら migration 追加

[internal] M 件
  - Behavior Method `Deactivate` の signature が field `deactivatedAt` を参照、Entity 未定義

総計: violations <N+M>, suggestions <K>
```

Empty:

```text
verify-spec-domain 結果（feature: <feature>）
domain.md の違反は検出されませんでした（suggestions: 0）。
```

## Step 5. Closing

- 単独実行: exit 0（情報的）
- `verify-spec` から chain: 違反数 + suggestion 数を caller に返す
- 自動修正しない

## AI Modification Scope

Strictly read-only. No file modifications.

## Constraints

- ❌ Auto-fix violations
- ❌ Modify any spec or source file
- ❌ Hardcode the section list (always read `.claude/scaffold-spec/domain-spec.md`)
- ❌ Skip the target-confirmation `AskUserQuestion` when standalone
- ❌ Treat entity ↔ SQL mismatch as hard violation (always `suggestion`)
- ✅ Japanese output
- ✅ Cite source-of-truth document + line
- ✅ Run all checks in one pass

## Checklist

- [ ] Target feature confirmed or supplied
- [ ] `.claude/scaffold-spec/domain-spec.md` read this run
- [ ] `domain.md` format checked (sections + YAML)
- [ ] Entity ↔ SQL soft check executed
- [ ] Internal consistency checks done
- [ ] Findings cite source-of-truth
- [ ] Report Japanese
- [ ] No files modified
