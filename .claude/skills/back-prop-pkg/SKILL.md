---
name: back-prop-pkg
description: Drift detection skill for the pkg layer. Reads `pkg/README.md` + each `pkg/<name>/README.md` (sub-package READMEs are mandatory in pkg — each utility documents Public API / Wraps / Notes) + `pkg/**/*.go` + `arch-check-pkg` skill body, surfaces three drift categories: (A) README → Code drift (e.g., `pkg/` importing `internal/`, framework dependency, business-logic leak), (B) Code → README undocumented pattern (3+ files), (C) Skill ↔ README duplication (slim-down candidates). Each finding with **explicit reasoning**, per-item user decision. AI drafts approved README / Skill changes with reasoning shown, writes after confirmation. Does NOT modify implementation code. Recommended trigger: after editing files in `pkg/`, before commit. Standalone-callable; chained from `back-prop` integrator skips scope question.
---

# Back-Prop — Pkg

Detect drift between pkg READMEs (layer + sub-package), pkg implementation, and pkg-related skill body.

A Japanese reference translation of this skill is available at `SKILL.ja.md` in the same directory (not loaded as a skill; for human reference only).

## When to Use

- After editing files in `pkg/` and before `/commit`
- Periodic hygiene check
- Chained from `back-prop` integrator

Do NOT use for: implementation code fixes / single-file architecture compliance (`arch-check-pkg`).

## What This Skill Reads / Writes

**Reads (always)**:

- `pkg/README.md` — layer canonical (Policy / Package List / Package Details / Checklist for Adding a New Package)
- `pkg/<name>/README.md` — sub-package READMEs (Public API required; **Wraps** appears when wrapping a third-party lib; **Notes** appears when caveats need to be documented). All sub-packages have a README per pkg layer convention
- `pkg/**/*.go` — implementation (excluding `*.gen.go`, `*_test.go`)
- `.claude/skills/arch-check-pkg/SKILL.md`

**Writes (only with explicit per-item approval + reasoning shown)**:

- `pkg/README.md` + `pkg/<name>/README.md`
- `arch-check-pkg/SKILL.md`

**Never touches**: implementation code, other layers.

## First Step: Confirm Scope + Detection Categories

`AskUserQuestion` with two batched questions:

1. 「back-prop-pkg のスコープを選んでください」 / 「変更ファイルのみ」 / 「pkg/ 全体」 / 「キャンセル」
2. 「検出する drift 種別を選んでください（multi-select、既定 3 種類すべて）」 / (A) / (B) / (C)

Chained: skip.

## Step 1. Load Inputs

1. Read `pkg/README.md` + all sub-package READMEs (`pkg/<name>/README.md`)
2. Parse in-scope `.go` files (imports especially, public API, file structure)
3. Read `arch-check-pkg/SKILL.md`

## Step 2. Detection (A) README → Code Drift

pkg-specific drift checks per README:

- No `internal/**` imports in any `pkg/*` file (top-level constraint)
- No framework imports (echo / fx / gorm / etc.) unless sub-package README explicitly allows
- No business logic specific to a single feature (Policy)
- Sub-package README present for every sub-package, with `Public API` section required (Wraps / Notes are conditional — Wraps when wrapping third-party lib, Notes when caveats need to be documented)
- Sub-package listed in top-level `pkg/README.md` Package List

Each violation: `(rule, file:line, reasoning, choices: fix code / relax README / exception)`.

## Step 3. Detection (B) Code → README Undocumented Pattern

Recurring patterns (≥3 files / packages):

- Common helper signature patterns
- Common test patterns (table-driven, t.Parallel placement)
- Wrapping conventions for 3rd-party libraries

Each finding: `(pattern, count, sample, reasoning, choices: document / ignore / refactor)`.

## Step 4. Detection (C) Skill ↔ README Duplication

For each rule enumerated in `arch-check-pkg/SKILL.md`:

- Check if same rule is in `pkg/README.md` Constraints / Policy
- Duplicates: `(rule, skill location, README location, reasoning, choices: slim / keep both)`

## Step 5. Aggregated Report (Japanese)

```text
back-prop-pkg 結果（scope: <X>, 種別: A/B/C）

[A] README → Code drift  N 件
[B] undocumented pattern  M 件
[C] Skill duplication  K 件

総計 N+M+K 件。per-item 承認 / 棄却を判断してください。
```

## Step 6. Per-Item User Decision

Each finding: `AskUserQuestion`. Approved doc/skill change → AI shows reasoning + draft → user final confirm → write.

## Step 7. Closing

```text
back-prop-pkg 完了。
  処理 finding: N
    README 更新: <X> / Skill 簡略化: <Y> / コード修正委任: <Z> / 無視: <W>
  最終 make md-lint OK
```

## AI Modification Scope

Write: `pkg/README.md` + sub-package READMEs + `arch-check-pkg/SKILL.md` (approved + reasoned).
Never: implementation code, other layers.

## Constraints

- ❌ 実装コードへの書き込み
- ❌ user 承認なしの自動更新
- ❌ 理由を述べずに draft 実行
- ❌ scope + 種別 `AskUserQuestion` をスキップ
- ❌ recurring threshold 3 未満を surface
- ✅ Japanese output, per-item reasoning, per-item 承認制
- ✅ READMEs が canonical (pkg は layer + sub-package READMEs の二層構造)

## Checklist

- [ ] Scope + 種別確認 / 受領
- [ ] layer + sub-package READMEs + 実装 + skill 読み込み
- [ ] (A)(B)(C) 検出
- [ ] 各 finding に reasoning 明示
- [ ] per-item 承認 → 理由提示 → draft → 最終確認 → 書き込み
- [ ] 実装コード触らない
- [ ] サマリ surface
