---
name: arch-check
description: Integrator skill for architectural compliance checks. Confirms scope via `AskUserQuestion` (changed files vs full repo), detects which layers are touched, then chains only the relevant per-layer skills (`arch-check-domain` / `-usecase` / `-controller` / `-infra` / `-pkg`) in parallel-friendly order, passing scope context so each child skips its own scope question. Aggregates findings into a single Japanese report grouped by layer. Each per-layer skill enforces its own README rules + lean A conventions (controller / infra have additional convention enforcement since they're scaffold-derived, not spec-driven). Read-only orchestration — all writing is delegated to per-layer skills (which themselves are read-only).
---

# Arch Check

Integrator for layer-scoped architectural compliance checks. Chains 1〜5 per-layer skills based on scope.

A Japanese reference translation of this skill is available at `SKILL.ja.md` in the same directory (not loaded as a skill; for human reference only).

## When to Use

- About to commit / push and want comprehensive layer-compliance check across all touched layers.
- Reviewing a feature branch that touches multiple layers.
- CI gate before merge.

Use the per-layer skill directly (`arch-check-<layer>`) when you only care about one layer.

Do NOT use this skill for:

- Style / formatting — `make fix` / `make lint`.
- General code review — `/review` / `/ultrareview`.
- Spec validation — `verify-spec`.

## Per-Layer Skills Chained

| Skill | Layer | Lean A enforcement |
| --- | --- | --- |
| `arch-check-domain` | `internal/domain/**` | entity ↔ SQL カラム soft 対応（method 形式 / VO ラップは逸脱許容、suggestion のみ） |
| `arch-check-usecase` | `internal/usecase/**` | thin orchestrator + boundary 利用 + tx 境界 |
| `arch-check-controller` | `internal/controller/**` | handler pure template + operationId ↔ method 一致 |
| `arch-check-infra` | `internal/infrastructure/**` | Repository pure template + sqlc gen soft 対応（multi-query / switch dispatch / JOIN 許容）+ pgerror 利用 |
| `arch-check-pkg` | `pkg/**` | `internal/` 依存禁止 + framework 非依存 |

## First Step: Confirm Scope + TODO opt

This skill **MUST call `AskUserQuestion` immediately after invocation** with 2 questions (batched).

Default-detect scope by checking branch vs base (`gh repo view --json defaultBranchRef -q '.defaultBranchRef.name'`):

- 未マージのコミットあり → 「変更ファイルのみ」を既定
- main / release/* / no diff → 「リポジトリ全体」を既定

```text
質問 1: どのスコープでアーキ検査を実行しますか？
選択肢:
  - 変更ファイルのみ（ベースブランチとの diff、touched layer のみ chain）
  - リポジトリ全体（5 layer skill 全部 chain）
  - 特定 layer のみ（layer を続けて指定）
  - キャンセル

質問 2: suggestion 検出箇所に TODO hand-off コメントを追加しますか？
選択肢:
  - 追加する（既定） — 各 per-layer skill が逸脱位置に `// TODO:` を書き込む（人間に解決を委ねる）
  - 追加しない（read-only） — レポートのみ、コード一切触らない
```

TODO opt は domain / controller / infra の per-layer skill に伝播。usecase / pkg は violation 中心なので TODO 書き込み対象外（opt 関係なく read-only）。

## Step 1. Resolve Scope to Layers

For "changed files" mode:

```sh
BASE=$(gh pr view --json baseRefName -q '.baseRefName' 2>/dev/null || gh repo view --json defaultBranchRef -q '.defaultBranchRef.name')
git diff --name-only "origin/${BASE}...HEAD" -- '*.go' | grep -vE '\.gen\.go$|\.sql\.go$|_mock\.go$|_test\.go$' || true
```

Map to layers by path prefix:

| Path prefix | Skill to chain |
| --- | --- |
| `internal/domain/` | `arch-check-domain` |
| `internal/usecase/` | `arch-check-usecase` |
| `internal/controller/` | `arch-check-controller` |
| `internal/infrastructure/` | `arch-check-infra` |
| `pkg/` | `arch-check-pkg` |

Other `internal/` paths (cli / system / di / config 等) → 報告のみ、専用 layer skill 無し（CLAUDE.md guidance を直接適用）。

For "full repo" mode: chain all 5 skills.

For "specific layer" mode: ask user which layer(s), then chain only those.

If no layers detected (changed files mode with no Go changes) → exit cleanly with message.

## Step 2. Chain Per-Layer Skills

For each layer in scope, invoke its skill via `Skill` tool, passing the scope + TODO opt context so child skips its own `AskUserQuestion`:

- `arch-check-domain` （scope = changed-domain-files, TODO opt = yes/no）
- ... etc.

Per-layer skills run sequentially (lint output reused across them via `/tmp/arch-check-*.out`) or in parallel if independent. Sequential is simpler for output aggregation.

Each child returns its findings + TODO add count; collect them with layer label.

## Step 3. Aggregate Report

Combine all per-layer findings into a single Japanese report:

```text
arch-check 統合結果（スコープ: <scope>）

[lint baseline]
  make lint: OK / FAIL (<n>件)

[domain] violations: N, suggestions: K
  internal/domain/foo/bar.go:12 ...

[usecase] violations: N, suggestions: K
  ...

[controller] violations: N, suggestions: K (lean A)
  ...

[infra] violations: N, suggestions: K (lean A)
  ...

[pkg] violations: N, suggestions: K
  ...

総計: violations <sum>, suggestions <sum>
TODO hand-off: 追加 <sum> 件, スキップ <sum> 件（既存コメント）
```

If all clean:

```text
arch-check 統合結果（スコープ: <scope>）
全 layer で違反は検出されませんでした（チェック済み: <layer list>）。
```

## Step 4. Closing

- 統合 skill 自体は read-only（child も read-only）
- `/commit` から chain 時は violations > 0 で non-zero status
- 単独実行時は情報的、exit 0
- 自動修正なし

## AI Modification Scope

統合 skill 自体は何も書かない。read scope / write scope は per-layer skill に委譲:

- 読み込み: 各 layer の README + 関連ファイル
- 書き込み: user opt 時のみ、`internal/<layer>/**/*.go` の suggestion 位置への `// TODO:` hand-off コメント追加（per-layer skill が実施、integrator 経由でも child の制約に従う）

## Constraints

- ❌ ソースファイルを直接読まない（per-layer skill 任せ）
- ❌ Skip scope + TODO opt `AskUserQuestion`
- ❌ Heuristic findings (handler bloat 等) を hard violation 扱い（child skill が `suggestion` ラベル付け、integrator は respect）
- ❌ Modify any file directly（per-layer skill 経由の TODO 書き込みのみ、user opt 時）
- ✅ Japanese aggregated report
- ✅ Chain only touched layers (changed-files mode で効率化)
- ✅ Per-layer skill が独立 standalone 動作可能であることを維持
- ✅ TODO opt を child に propagate

## Checklist

- [ ] Scope + TODO opt confirmed via `AskUserQuestion`
- [ ] Layer detection from changed files / full repo
- [ ] Touched layers の per-layer skill を chain
- [ ] 各 child skill が自身の README + lean A 規則を適用
- [ ] Aggregated Japanese report 出力
- [ ] No file modifications by integrator itself (child skills may add TODO comments per opt)
- [ ] No commit / push
