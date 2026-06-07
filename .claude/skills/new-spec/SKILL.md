---
name: new-spec
description: Integrator skill that creates the 2-layer spec set under `docs/spec/<feature>/` by chaining per-layer skills (`new-spec-domain` / `new-spec-usecase`). lean A 構成では controller / infra に spec を持たず OpenAPI + sqlc gen + 命名規約から導出するため、spec は domain + usecase の 2 ファイルだけ。Confirms feature name (kebab-case) once via `AskUserQuestion`, then asks which layers to scaffold (multi-select, default both) and chains the selected per-layer skills in dependency order (`domain` → `usecase`). Each per-layer skill still collects its own layer-specific identity (aggregate / package / interface name) and writes a Markdown file with YAML code-block placeholders and TODO markers. Skips layers whose target file already exists rather than aborting the chain. Use a per-layer skill (`new-spec-<layer>`) directly when scaffolding a single layer template. Does not invent business content — gathers identity only. All structural section definitions are read from `.claude/scaffold-spec/<layer>-spec.md` at runtime by each per-layer skill, so spec format changes propagate automatically.
---

# New Spec

Integrator for creating the 2-layer spec template set for one feature (lean A: `domain.md` + `usecase.md`).

A Japanese reference translation of this skill is available at `SKILL.ja.md` in the same directory (not loaded as a skill; for human reference only).

## When to Use

- Starting a brand-new feature and want both spec templates created in one chained flow.
- A feature directory exists with one spec and you want to add the missing one.

Do NOT use this skill for:

- Creating a single layer template — invoke the matching per-layer skill directly:
  - `new-spec-domain` / `new-spec-usecase`
- Editing existing spec files — open in the editor.
- Generating Go code from spec — that's `scaffold-endpoint` (or per-layer `scaffold-<layer>`).
- Validating spec consistency — that's `verify-spec`.

## Why only 2 specs (lean A)

controller / infra layers are not spec-driven — they are derived at scaffold time from:

- OpenAPI gen + naming convention → controller
- domain Repository IF + sqlc gen function names → infra

The convention itself is enforced by `arch-check-controller` / `arch-check-infra` (handler/Repository body purity, name correspondence). See `.claude/scaffold-spec/lifecycle.md` for the full rationale.

## What This Skill Reads / Writes

This skill itself **does not read or write spec files directly**. All structural work is delegated to per-layer skills, each of which:

- Reads its matching `.claude/scaffold-spec/<layer>-spec.md` at runtime for the section list.
- Asks for layer-specific identity via its own `AskUserQuestion`.
- Writes one Markdown file under `docs/spec/<feature>/`.

This skill is purely orchestration: it confirms feature name + layer selection, then invokes per-layer skills in dependency order.

## First Step: Confirm Feature and Layer Selection

This skill **MUST call `AskUserQuestion` immediately after invocation**. Use a single batched call with two questions:

### Question 1: Feature name

- Free-text: 「feature 名（kebab-case）。例: `user-management`, `order-fulfillment`」
- Validate `^[a-z][a-z0-9-]*$`. Re-ask on invalid input.

### Question 2: Layers to scaffold

- Multi-select, 2 options:
  - 「domain」
  - 「usecase」
- Default: both. The user may deselect either (e.g., `domain` already exists and only `usecase` is needed).

After the answers come back:

1. Build the work plan: feature name + ordered list of layers to scaffold (dependency order: `domain` → `usecase`).
2. For each selected layer, check whether `docs/spec/<feature>/<layer>.md` already exists. If it does, mark it as **skip** rather than failing the entire chain.
3. Show the plan in Japanese with skip markers, then confirm via `AskUserQuestion`:
   - Question: 「以下の順番で per-layer skill を chain します。進めますか？」
   - Options: 「進める」 / 「キャンセル」

If after applying skips the executable list is empty, report it and stop:

```text
全 layer の spec ファイルが既に存在します。実行対象がありません。
```

## Step 1. Chain Per-Layer Skills in Dependency Order

For each layer remaining in the plan (in order `domain` → `usecase`):

1. Invoke the matching skill: `new-spec-<layer>` via the `Skill` tool.
2. Pass the feature name in the chain context so the child skill can skip its own feature-name `AskUserQuestion` and only ask for layer-specific identity (aggregate / package / interface name).
3. The child still asks the user to confirm and writes its own file.
4. If the child reports failure (e.g., user cancels the per-layer confirmation), stop the chain and surface the status; do NOT auto-rollback already-written layers.

Each per-layer skill is independent — they only share the feature name and the layer-execution order. They do not pass aggregate or interface names between each other.

## Step 2. Closing Report

After all selected layers have been processed (chain completed or stopped early), print a Japanese summary:

```text
new-spec 完了（feature: <feature>）。
  ✓ domain   : 作成済み (docs/spec/<feature>/domain.md)
  ✓ usecase  : 作成済み (docs/spec/<feature>/usecase.md)

次のアクション:
  - editor で TODO を埋める
  - 2 spec 揃ったら verify-spec で format / 派生元 / cross-layer 参照を検証
  - 検証通過後に scaffold-endpoint で実装一括生成
    (controller / infra は OpenAPI / sqlc gen から自動導出される)
```

Adjust marks per actual result:

- ✓ = newly created
- `-` = skipped (file already existed)
- ✗ = failed / cancelled

Do NOT commit. Do NOT trigger any scaffold skill automatically.

## AI Modification Scope

This skill itself writes no files. All write scope is delegated to per-layer skills, each scoped to `docs/spec/<feature>/<layer>.md`.

## Constraints

- ❌ Write spec files directly (always delegate to per-layer skills)
- ❌ Hardcode section lists (per-layer skills read `.claude/scaffold-spec/<layer>-spec.md` at runtime)
- ❌ Abort the entire chain because one layer's file already exists — mark as skip and continue
- ❌ Auto-rollback earlier created layers if a later layer fails
- ❌ Skip the feature-confirmation `AskUserQuestion`
- ❌ Run per-layer skills out of dependency order
- ❌ Offer controller / infra as spec options (lean A: those layers are derived, not spec-driven)
- ✅ Japanese user-facing output
- ✅ Multi-select layer choice (domain / usecase) with both default
- ✅ Chain in dependency order: `domain` → `usecase`
- ✅ Surface skip / fail status per layer in the final report

## Checklist

- [ ] Feature name + layer selection confirmed via `AskUserQuestion`
- [ ] Existing layer files marked as skip (not failure)
- [ ] Per-layer skills invoked in dependency order
- [ ] Each per-layer skill ran its own identity `AskUserQuestion`
- [ ] Per-layer failure stopped the chain without auto-rollback
- [ ] Final Japanese summary uses ✓ / - / ✗ marks
- [ ] No direct file writes by this skill
- [ ] No commit / scaffold auto-trigger
