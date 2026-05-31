---
name: back-prop-usecase
description: Drift detection skill for the usecase layer. Reads `internal/usecase/README.md` (canonical, with Implementation Example) + `internal/usecase/boundary/README.md` + `internal/usecase/**/*.go` + `arch-check-usecase` / `scaffold-usecase` / `new-spec-usecase` / `verify-spec-usecase` skill bodies, surfaces three drift categories: (A) README → Code drift (README rules code violates), (B) Code → README undocumented pattern (recurring patterns in 3+ files not in README), (C) Skill ↔ README duplication (skill body rules also in README — slim-down candidates). Each finding includes **explicit reasoning**; user decides per-item. AI drafts approved README / Skill changes with reasoning shown, writes after final confirmation. Does NOT modify implementation code. Recommended trigger: after editing files in `internal/usecase/`, before commit. Standalone-callable; chained from `back-prop` integrator skips scope question.
---

# Back-Prop — Usecase

Detect drift between usecase README + Implementation Example, usecase implementation, and usecase-related skill bodies.

A Japanese reference translation of this skill is available at `SKILL.ja.md` in the same directory (not loaded as a skill; for human reference only).

## When to Use

- After editing files in `internal/usecase/` and before `/commit`, to detect drift
- Periodic hygiene check
- When refactoring usecase conventions
- Chained from `back-prop` integrator

Do NOT use for:

- Fixing implementation code (surface only)
- Single-file architecture compliance (`arch-check-usecase`)
- Generating new usecase code (`scaffold-usecase`)

## What This Skill Reads / Writes

**Reads (always)**:

- `internal/usecase/README.md` — canonical (includes Implementation Example at bottom)
- `internal/usecase/boundary/README.md` — boundary conventions
- `internal/usecase/**/*.go` — implementation (excluding `*.gen.go`, `_mock.go`, `*_test.go`)
- `.claude/skills/arch-check-usecase/SKILL.md` — rule enumeration to cross-check
- `.claude/skills/scaffold-usecase/SKILL.md` — generation conventions vs README
- `.claude/skills/new-spec-usecase/SKILL.md` + `.claude/skills/verify-spec-usecase/SKILL.md` — secondary

**Writes (only with explicit per-item approval + reasoning shown)**:

- `internal/usecase/README.md` + `internal/usecase/boundary/README.md` — doc updates
- usecase-related skill SKILL.md files — slim-down updates

**Never touches**: implementation code (`internal/usecase/**/*.go`), other layers, generated files.

## First Step: Confirm Scope + Detection Categories

`AskUserQuestion` with two batched questions:

1. 「back-prop-usecase のスコープを選んでください」 / 「変更ファイルのみ」 / 「internal/usecase/ 全体」 / 「キャンセル」
2. 「検出する drift 種別を選んでください（multi-select、既定 3 種類すべて）」 / 「(A) README → Code drift」 / 「(B) Code → README undocumented pattern」 / 「(C) Skill ↔ README duplication」

When chained from `back-prop` integrator, both supplied; skip.

## Step 1. Load Inputs

1. Read `internal/usecase/README.md` (rules + Implementation Example) + `boundary/README.md`
2. Parse in-scope `.go` files (struct fields, methods, imports, tracer span usage, boundary calls, DTO conventions)
3. Read usecase-related skill SKILL.md

## Step 2. Detection (A) README → Code Drift

Common usecase-specific drift checks per README:

- `usecase` struct name (README "fixed") → check all packages use it
- Every public method starts with `tracer.Start(ctx); defer endSpan()` (Observability section)
- No `time.Now()` direct, no `math/rand` direct (Time Handling / Boundary Concept)
- No `internal/infrastructure/**` imports (Forbidden dependencies)
- No HTTP/echo imports (Implementation Notes)
- Tx via `tx.Manager.Do(...)` for multi-write workflows

Each violation: `(rule, file:line, reasoning, user choices: fix code / relax README / exception)`.

## Step 3. Detection (B) Code → README Undocumented Pattern

Scan for recurring patterns (≥3 files):

- DTO naming conventions actually used (e.g., `<X>MutableFields` / `<X>ParamsDTO`)
- Specific error wrapping patterns
- Common helper / private method patterns
- Constructor argument ordering

Each finding: `(pattern, occurrence count, sample files, reasoning, user choices: document in README / ignore / refactor away)`.

## Step 4. Detection (C) Skill ↔ README Duplication

For each rule enumerated in `arch-check-usecase/SKILL.md` (and other usecase skills):

- Check if same rule (verbatim or paraphrase) is in `internal/usecase/README.md`
- If duplicated: `(rule, skill location, README location, reasoning, user choices: slim skill / keep both)`

## Step 5. Aggregated Report (Japanese)

```text
back-prop-usecase 結果（scope: <X>, 種別: A/B/C）

[A] README → Code drift  N 件
[B] undocumented pattern  M 件
[C] Skill duplication  K 件

総計 N+M+K 件。per-item 承認 / 棄却を判断してください。
```

## Step 6. Per-Item User Decision

Each finding: `AskUserQuestion` with finding's choices. If user approves doc/skill change → AI shows reasoning + draft → user final confirm → write.

## Step 7. Closing

```text
back-prop-usecase 完了。
  処理 finding: N
    README 更新: <X> 件
    Skill 簡略化: <Y> 件
    コード修正委任: <Z> 件
    無視 / 棄却: <W> 件
  最終 make md-lint OK
```

## AI Modification Scope

Write: `internal/usecase/README.md`, `boundary/README.md`, usecase-related skill files (approved + reasoned).
Never: implementation code, other layers, generated files.

## Constraints

- ❌ 実装コードへの書き込み
- ❌ user 承認なしの自動更新
- ❌ 理由を述べずに draft 実行
- ❌ scope + 種別 `AskUserQuestion` をスキップ
- ❌ recurring threshold 3 未満を「規約候補」 surface
- ✅ Japanese output, per-item reasoning, per-item 承認制
- ✅ README が canonical の前提を貫く

## Checklist

- [ ] Scope + 種別確認 / 受領
- [ ] README + boundary README + 実装 + skill 読み込み
- [ ] (A)(B)(C) 検出（選択種別のみ）
- [ ] 各 finding に reasoning 明示
- [ ] per-item 承認 → 理由提示 → draft → 最終確認 → 書き込み
- [ ] 実装コード触らない
- [ ] サマリ surface
