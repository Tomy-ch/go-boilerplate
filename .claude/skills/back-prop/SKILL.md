---
name: back-prop
description: Integrator skill for drift detection across all layers. Confirms scope (changed files vs full repo) + detection categories (A/B/C, multi-select) once via `AskUserQuestion`, detects which layers are touched, then chains only relevant per-layer skills (`back-prop-domain` / `-usecase` / `-controller` / `-infra` / `-pkg`) passing scope + category context so each child skips its own scope question. Aggregates findings into a single Japanese report grouped by layer. Per-item user approval still happens inside each child skill (AI shows reasoning + draft → user confirms → write). Does NOT modify implementation code — all writes are to READMEs / Skill SKILL.md files only, and only with explicit per-item approval. Recommended trigger: after multi-layer refactor or before major PR review, to confirm doc + skill remain in sync with code reality (priority README > Code > SKILL).
---

# Back-Prop

Integrator for drift detection across layers. Chains per-layer skills based on scope.

A Japanese reference translation of this skill is available at `SKILL.ja.md` in the same directory (not loaded as a skill; for human reference only).

## When to Use

- After multi-layer refactor, before PR review
- Periodic hygiene sweep (catch undocumented conventions / skill bloat / README drift across all layers)
- When introducing a new layer-wide convention (to see where it's already followed / not yet)

Use a per-layer skill directly (`back-prop-<layer>`) when only one layer needs check.

Do NOT use for:

- Implementation code fixes (surface only; per-layer skills don't write code either)
- Architecture compliance per file — `arch-check` (with TODO hand-off)
- Spec validation — `verify-spec`

## Per-Layer Skills Chained

| Skill | Layer | Notes |
| --- | --- | --- |
| `back-prop-domain` | `internal/domain/**` | README has Implementation notes / Aggregate Design / Testing strategy / Do / Don't sections |
| `back-prop-usecase` | `internal/usecase/**` | README has Implementation Example at bottom |
| `back-prop-controller` | `internal/controller/**` | handler README has reference snippet |
| `back-prop-infra` | `internal/infrastructure/**` | 3-level READMEs (infra / rdb / pgerror), no full snippet (sibling code de facto reference) |
| `back-prop-pkg` | `pkg/**` | layer README + mandatory sub-package READMEs |

## First Step: Confirm Scope + Detection Categories

`AskUserQuestion` with two batched questions (defaults auto-detected by git diff):

1. 質問: 「back-prop のスコープを選んでください」
   - 選択肢: 「変更ファイルのみ（ベースブランチとの diff、touched layer のみ chain）」 / 「リポジトリ全体（5 layer 全部 chain）」 / 「特定 layer のみ（layer を続けて指定）」 / 「キャンセル」

2. 質問: 「検出する drift 種別を選んでください（multi-select、既定 3 種類すべて）」
   - 選択肢: 「(A) README → Code drift」 / 「(B) Code → README undocumented pattern」 / 「(C) Skill ↔ README duplication」

Detection categories are propagated to all chained child skills.

## Step 1. Resolve Layers in Scope

For "changed files" mode:

```sh
BASE=$(gh pr view --json baseRefName -q '.baseRefName' 2>/dev/null || gh repo view --json defaultBranchRef -q '.defaultBranchRef.name')
git diff --name-only "origin/${BASE}...HEAD" -- '*.go' | grep -vE '\.gen\.go$|\.sql\.go$|_mock\.go$|_test\.go$'
```

Map to layers by path prefix:

| Path prefix | Skill to chain |
| --- | --- |
| `internal/domain/` | `back-prop-domain` |
| `internal/usecase/` | `back-prop-usecase` |
| `internal/controller/` | `back-prop-controller` |
| `internal/infrastructure/` | `back-prop-infra` |
| `pkg/` | `back-prop-pkg` |

For "full repo": chain all 5. For "specific layer": ask user which.

## Step 2. Chain Per-Layer Skills

For each layer in scope, invoke matching `back-prop-<layer>` via `Skill` tool, passing scope + categories context (so children skip their own `AskUserQuestion`).

Each child runs independently with **per-item user approval inside the child** (AI shows reasoning + draft → user confirms → write README / Skill).

Collect each child's report.

## Step 3. Aggregate Report (Japanese)

```text
back-prop 統合結果（scope: <X>, 種別: A/B/C）

[domain]   findings <N>, README 更新 <X>, Skill 簡略化 <Y>, コード修正委任 <Z>, 無視 <W>
[usecase]  ...
[controller] ...
[infra]    ...
[pkg]      ...

総 finding: <sum>, 書き込み: <sum>, コード修正委任: <sum>
最終 make md-lint OK
```

If all clean:

```text
back-prop 統合結果（scope: <X>, 種別: A/B/C）
全 layer で drift は検出されませんでした（チェック済み: <layer list>）。
```

## Step 4. Closing

- Integrator は何も書かない、すべて child skill 経由
- 実装コードへの書き込みは child も含めて一切なし
- README / Skill 書き込みは child 内で user 承認後

## AI Modification Scope

Integrator itself writes nothing. All write scope delegated to per-layer skills (each scoped to its layer's READMEs + skill SKILL.md only, with per-item user approval + reasoning shown).

## Constraints

- ❌ ソースファイル直接読まない（child 任せ）
- ❌ scope + 種別 `AskUserQuestion` をスキップ
- ❌ Integrator 自身による file 書き込み
- ❌ child の per-item 承認をスキップさせない（integrator は children に context 渡すだけ）
- ✅ Japanese aggregated report
- ✅ Chain only touched layers (changed-files mode)
- ✅ child が独立 standalone 動作可能であることを維持
- ✅ Categories propagation to all children

## Checklist

- [ ] Scope + 種別を `AskUserQuestion` で確認
- [ ] Layer 検出（changed files / full repo / specific layer）
- [ ] touched layer の back-prop-<layer> を chain
- [ ] 各 child が自身の README + 実装 + skill を読み、(A)(B)(C) 検出
- [ ] 各 child が per-item で reasoning + user 承認 + 書き込み
- [ ] 集約 Japanese report 出力
- [ ] Integrator 自身による file 書き込みなし
- [ ] commit / push なし
