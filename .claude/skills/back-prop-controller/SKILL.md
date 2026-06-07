---
name: back-prop-controller
description: Drift detection skill for the controller layer. Reads `internal/controller/README.md` + `internal/controller/handler/README.md` (canonical, with reference snippet) + OpenAPI gen + `internal/controller/**/*.go` + `arch-check-controller` / `scaffold-controller` skill bodies, surfaces three drift categories: (A) README → Code drift (e.g., handler not using `BindHandler` / `server` struct / `gen.NewStrictHandler` per README reference snippet), (B) Code → README undocumented pattern (3+ files), (C) Skill ↔ README duplication (slim-down candidates). Each finding with **explicit reasoning**, per-item user decision. AI drafts approved README / Skill changes with reasoning shown, writes after confirmation. Does NOT modify implementation code. Recommended trigger: after editing files in `internal/controller/`, before commit. Standalone-callable; chained from `back-prop` integrator skips scope question.
---

# Back-Prop — Controller

Detect drift between controller READMEs (with reference snippet), controller implementation, and controller-related skill bodies.

A Japanese reference translation of this skill is available at `SKILL.ja.md` in the same directory (not loaded as a skill; for human reference only).

## When to Use

- After editing files in `internal/controller/` and before `/commit`
- Periodic hygiene check
- Chained from `back-prop` integrator

Do NOT use for: implementation code fixes / single-file architecture compliance (`arch-check-controller`) / generation (`scaffold-controller`).

## What This Skill Reads / Writes

**Reads (always)**:

- `internal/controller/README.md` + `internal/controller/handler/README.md` — canonical (handler README has reference snippet)
- `internal/controller/**/*.go` — implementation (excluding `*.gen.go`, `_mock.go`, `*_test.go`)
- `internal/controller/handler/**/gen/server.gen.go` — OpenAPI gen (for operationId ↔ handler method cross-check)
- `.claude/skills/arch-check-controller/SKILL.md` + `.claude/skills/scaffold-controller/SKILL.md` — rule / generation pattern enumeration

**Writes (only with explicit per-item approval + reasoning shown)**:

- `internal/controller/README.md` + `internal/controller/handler/README.md`
- Controller-related skill SKILL.md files

**Never touches**: implementation code (`internal/controller/**/*.go`), other layers, generated `gen/` files.

## First Step: Confirm Scope + Detection Categories

`AskUserQuestion` with two batched questions:

1. 「back-prop-controller のスコープを選んでください」 / 「変更ファイルのみ」 / 「internal/controller/ 全体」 / 「キャンセル」
2. 「検出する drift 種別を選んでください（multi-select、既定 3 種類すべて）」 / 「(A) README → Code drift」 / 「(B) Code → README undocumented pattern」 / 「(C) Skill ↔ README duplication」

Chained: skip.

## Step 1. Load Inputs

1. Read controller README + handler README (rules + reference snippet)
2. Parse in-scope `.go` files (handler struct, BindHandler / New, gen.NewStrictHandler usage, tracer span, ctxhelper usage)
3. Read controller-related skill SKILL.md

## Step 2. Detection (A) README → Code Drift

Controller-specific drift checks per README:

- handler struct name `server` (lowercase) per reference snippet
- Constructor `BindHandler(echo, tf, uc)` (not `New`) per reference snippet
- Inside `BindHandler`: `gen.RegisterHandlers(e, gen.NewStrictHandler(&server{...}, nil))` pattern
- Every handler method starts with `s.tracer.Start(ctx); defer endSpan()`
- handler body = pure template (no business logic)
- DI: `fx.Invoke(<pkg>.BindHandler)` in `internal/di/module/controller.go` (not `fx.Provide`)
- operationId ↔ handler method 1:1 (camelCase)
- No Repository / infra imports

Each violation: `(rule, file:line, reasoning, choices: fix code / relax README / exception)`.

## Step 3. Detection (B) Code → README Undocumented Pattern

Recurring patterns (≥3 files):

- Multi-usecase orchestration patterns
- Response building patterns (paging utility usage etc.)
- Auth context handling patterns

Each finding: `(pattern, count, sample, reasoning, choices: document / ignore / refactor)`.

## Step 4. Detection (C) Skill ↔ README Duplication

For each rule enumerated in controller-related skills:

- Check if same rule is in `internal/controller/README.md` or `handler/README.md`
- Duplicates: `(rule, skill location, README location, reasoning, choices: slim / keep both)`

## Step 5. Aggregated Report (Japanese)

```text
back-prop-controller 結果（scope: <X>, 種別: A/B/C）

[A] README → Code drift  N 件
[B] undocumented pattern  M 件
[C] Skill duplication  K 件

総計 N+M+K 件。per-item 承認 / 棄却を判断してください。
```

## Step 6. Per-Item User Decision

Each finding: `AskUserQuestion` with finding's choices. Approved doc/skill change → AI shows reasoning + draft → user final confirm → write.

## Step 7. Closing

```text
back-prop-controller 完了。
  処理 finding: N
    README 更新: <X> / Skill 簡略化: <Y> / コード修正委任: <Z> / 無視: <W>
  最終 make md-lint OK
```

## AI Modification Scope

Write: controller READMEs + controller-related skill files (approved + reasoned).
Never: implementation code, other layers, generated `gen/`.

## Constraints

- ❌ 実装コードへの書き込み
- ❌ user 承認なしの自動更新
- ❌ 理由を述べずに draft 実行
- ❌ scope + 種別 `AskUserQuestion` をスキップ
- ❌ recurring threshold 3 未満を surface
- ✅ Japanese output, per-item reasoning, per-item 承認制
- ✅ README が canonical

## Checklist

- [ ] Scope + 種別確認 / 受領
- [ ] READMEs + 実装 + skill 読み込み
- [ ] (A)(B)(C) 検出
- [ ] 各 finding に reasoning 明示
- [ ] per-item 承認 → 理由提示 → draft → 最終確認 → 書き込み
- [ ] 実装コード触らない
- [ ] サマリ surface
