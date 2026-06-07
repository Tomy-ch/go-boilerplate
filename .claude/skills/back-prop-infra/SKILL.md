---
name: back-prop-infra
description: Drift detection skill for the infrastructure layer. Reads `internal/infrastructure/README.md` + `internal/infrastructure/rdb/README.md` + `internal/infrastructure/rdb/pgerror/README.md` (canonical, principles-focused — infra READMEs do not carry full reference snippets, sibling Repository code is de facto reference) + `internal/infrastructure/**/*.go` + `arch-check-infra` / `scaffold-infra-db` skill bodies, surfaces three drift categories: (A) README → Code drift (e.g., missing `pgerror.NormalizeError`, missing tracer span), (B) Code → README undocumented pattern (3+ files), (C) Skill ↔ README duplication (slim-down candidates). Each finding with **explicit reasoning**, per-item user decision. AI drafts approved README / Skill changes with reasoning shown, writes after confirmation. Does NOT modify implementation code. Recommended trigger: after editing files in `internal/infrastructure/`, before commit. Standalone-callable; chained from `back-prop` integrator skips scope question.
---

# Back-Prop — Infra

Detect drift between infra READMEs (3 levels: infra / rdb / pgerror), infra implementation, and infra-related skill bodies.

A Japanese reference translation of this skill is available at `SKILL.ja.md` in the same directory (not loaded as a skill; for human reference only).

## When to Use

- After editing files in `internal/infrastructure/` and before `/commit`
- Periodic hygiene check
- Chained from `back-prop` integrator

Do NOT use for: implementation code fixes / single-file architecture compliance (`arch-check-infra`) / Repository generation (`scaffold-infra-db`).

## What This Skill Reads / Writes

**Reads (always)**:

- `internal/infrastructure/README.md` + `internal/infrastructure/rdb/README.md` + `internal/infrastructure/rdb/pgerror/README.md` — canonical (principles, no full snippets — code drift detection is structural)
- `internal/infrastructure/**/*.go` — implementation (excluding `*.gen.go`, `_mock.go`, `*_test.go`)
- `.claude/skills/arch-check-infra/SKILL.md` + `.claude/skills/scaffold-infra-db/SKILL.md`

**Writes (only with explicit per-item approval + reasoning shown)**:

- infra READMEs (3 levels)
- infra-related skill SKILL.md files

**Never touches**: implementation code, other layers, SQL files, sqlc generated files.

## First Step: Confirm Scope + Detection Categories

`AskUserQuestion` with two batched questions:

1. 「back-prop-infra のスコープを選んでください」 / 「変更ファイルのみ」 / 「internal/infrastructure/ 全体」 / 「キャンセル」
2. 「検出する drift 種別を選んでください（multi-select、既定 3 種類すべて）」 / (A) / (B) / (C)

Chained: skip.

## Step 1. Load Inputs

1. Read 3-level infra READMEs (infra / rdb / pgerror)
2. Parse in-scope `.go` files (Repository struct, constructor, sqlc calls, pgerror usage, tracer span)
3. Read infra-related skill SKILL.md

## Step 2. Detection (A) README → Code Drift

Infra-specific drift checks per READMEs:

- Every Repository method ends sqlc call result with `pgerror.NormalizeError(err)` (single-normalization-point principle, pgerror README)
- Every Repository method starts with `tracer.Start(ctx); defer endSpan()` (infra README Observability)
- Repository constructor returns domain Repository IF (not concrete struct) (rdb README)
- DI: `fx.Provide(<pkg>.New)` in `internal/di/module/infrastructure.go`
- No business logic in Repository body (Prohibited section)
- Repository body = data orchestration (sqlc + convert + pgerror only)

Each violation: `(rule, file:line, reasoning, choices: fix code / relax README / exception)`.

## Step 3. Detection (B) Code → README Undocumented Pattern

Recurring patterns (≥3 files):

- Conversion helper patterns (multi-row → slice helpers)
- Common `gen.New(r.db.NewLoggingDB(ctx))` boilerplate location
- Specific dispatching patterns (switch by status)

Each finding: `(pattern, count, sample, reasoning, choices: document / ignore / refactor)`.

## Step 4. Detection (C) Skill ↔ README Duplication

For each rule enumerated in infra-related skills:

- Check if same rule is in any of the 3 infra READMEs
- Duplicates: `(rule, skill location, README location, reasoning, choices: slim / keep)`

## Step 5. Aggregated Report (Japanese)

```text
back-prop-infra 結果（scope: <X>, 種別: A/B/C）

[A] README → Code drift  N 件
[B] undocumented pattern  M 件
[C] Skill duplication  K 件

総計 N+M+K 件。per-item 承認 / 棄却を判断してください。
```

## Step 6. Per-Item User Decision

Each finding: `AskUserQuestion`. Approved doc/skill change → AI shows reasoning + draft → user final confirm → write.

## Step 7. Closing

```text
back-prop-infra 完了。
  処理 finding: N
    README 更新: <X> / Skill 簡略化: <Y> / コード修正委任: <Z> / 無視: <W>
  最終 make md-lint OK
```

## AI Modification Scope

Write: infra READMEs (3 levels) + infra-related skill files (approved + reasoned).
Never: implementation code, SQL files, sqlc generated files, other layers.

## Constraints

- ❌ 実装コードへの書き込み
- ❌ user 承認なしの自動更新
- ❌ 理由を述べずに draft 実行
- ❌ scope + 種別 `AskUserQuestion` をスキップ
- ❌ recurring threshold 3 未満を surface
- ✅ Japanese output, per-item reasoning, per-item 承認制
- ✅ READMEs が canonical（infra は textual principles のみ、sibling コードが de facto reference）

## Checklist

- [ ] Scope + 種別確認 / 受領
- [ ] 3-level infra READMEs + 実装 + skill 読み込み
- [ ] (A)(B)(C) 検出
- [ ] 各 finding に reasoning 明示
- [ ] per-item 承認 → 理由提示 → draft → 最終確認 → 書き込み
- [ ] 実装コード触らない
- [ ] サマリ surface
