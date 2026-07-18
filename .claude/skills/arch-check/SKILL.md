---
name: arch-check
description: Integrator skill for architectural compliance checks. Confirms scope + TODO opt via `AskUserQuestion` (changed files vs full repo), detects which layers are touched, resolves the file list and runs `make lint` ONCE, then fans out the relevant read-only `arch-auditor-*` subagents (`arch-auditor-domain` / `-usecase` / `-controller` / `-infra` / `-pkg`) IN PARALLEL via the Agent tool — passing scope + resolved files + the shared lint output so each auditor skips its own scope question and does not re-run lint. Aggregates findings into a single Japanese report grouped by layer. Each auditor enforces its own README rules + lean A conventions (controller / infra have additional convention enforcement since they're scaffold-derived, not spec-driven). Detection is delegated to the read-only auditor subagents; the integrator itself only writes the optional `// TODO:` hand-off comments (single-threaded, after aggregation) when the user opts in. To audit a single layer, run this integrator and pick that layer in the scope question.
---

# Arch Check

Integrator for layer-scoped architectural compliance checks. Fans out 1〜5 per-layer **read-only auditor subagents** in parallel based on scope, then aggregates.

A Japanese reference translation of this skill is available at `SKILL.ja.md` in the same directory (not loaded as a skill; for human reference only).

## When to Use

- About to commit / push and want comprehensive layer-compliance check across all touched layers.
- Reviewing a feature branch that touches multiple layers.
- CI gate before merge.

To audit a single layer, run this integrator and pick that layer in the scope question (「特定 layer のみ」). Use it when 1+ layers are touched and you want parallel fan-out + a single aggregated report.

Do NOT use this skill for:

- Style / formatting — `make fix` / `make lint`.
- General code review — `/review` / `/ultrareview` / `impl-review`.
- Spec validation — `verify-spec`.

## Architecture: parallel auditor subagents

Detection is delegated to five **read-only worker subagents** under `.claude/agents/`, one per layer. The integrator runs them concurrently via the Agent tool (`subagent_type`), so per-layer audits no longer execute sequentially:

| Auditor subagent | Layer | Lean A enforcement |
| --- | --- | --- |
| `arch-auditor-domain` | `internal/domain/**` | entity ↔ SQL カラム soft 対応（method 形式 / VO ラップは逸脱許容、suggestion のみ） |
| `arch-auditor-usecase` | `internal/usecase/**` | thin orchestrator + boundary 利用 + tx 境界 |
| `arch-auditor-controller` | `internal/controller/**` | handler pure template + operationId ↔ method 一致 |
| `arch-auditor-infra` | `internal/infrastructure/**` | Repository pure template + sqlc gen soft 対応（multi-query / switch dispatch / JOIN 許容）+ pgerror 利用 |
| `arch-auditor-pkg` | `pkg/**` | `internal/` 依存禁止 + framework 非依存 |

These auditors are the per-layer audit workers. They are **strictly read-only** (no TODO writes) so that running five in parallel never produces concurrent writes to source. Any source write (TODO hand-off) is performed by **this integrator**, single-threaded, after aggregation.

## First Step: Confirm Scope + TODO opt

This skill **MUST call `AskUserQuestion` immediately after invocation** with 2 questions (batched).

Default-detect scope by checking branch vs base (`gh repo view --json defaultBranchRef -q '.defaultBranchRef.name'`):

- 未マージのコミットあり → 「変更ファイルのみ」を既定
- main / release/* / no diff → 「リポジトリ全体」を既定

```text
質問 1: どのスコープでアーキ検査を実行しますか？
選択肢:
  - 変更ファイルのみ（ベースブランチとの diff、touched layer のみ fan-out）
  - リポジトリ全体（5 auditor 全部 fan-out）
  - 特定 layer のみ（layer を続けて指定）
  - キャンセル

質問 2: suggestion 検出箇所に TODO hand-off コメントを追加しますか？
選択肢:
  - 追加する（既定） — 集約後に integrator が逸脱位置へ `// TODO:` を書き込む（人間に解決を委ねる）
  - 追加しない（read-only） — レポートのみ、コード一切触らない
```

TODO opt は domain / controller / infra の suggestion 検出にのみ適用。usecase / pkg は violation 中心なので TODO 書き込み対象外（opt 関係なく read-only）。

## Step 1. Resolve Scope to Layers + File Lists

For "changed files" mode:

```sh
BASE=$(gh pr view --json baseRefName -q '.baseRefName' 2>/dev/null || gh repo view --json defaultBranchRef -q '.defaultBranchRef.name')
git diff --name-only "origin/${BASE}...HEAD" -- '*.go' | grep -vE '\.gen\.go$|\.sql\.go$|_mock\.go$|_test\.go$' || true
```

Map to layers by path prefix, and **keep the per-layer file list** (you pass it to each auditor so the auditor does not re-resolve git):

| Path prefix | Auditor subagent |
| --- | --- |
| `internal/domain/` | `arch-auditor-domain` |
| `internal/usecase/` | `arch-auditor-usecase` |
| `internal/controller/` | `arch-auditor-controller` |
| `internal/infrastructure/` | `arch-auditor-infra` |
| `pkg/` | `arch-auditor-pkg` |

Other `internal/` paths (cli / system / di / config 等) → 報告のみ、専用 auditor 無し（CLAUDE.md guidance を直接適用）。

For "full repo" mode: fan out all 5 auditors (each resolves its own full-layer file list, or pass `scope=full`).

For "specific layer" mode: ask user which layer(s), then fan out only those.

If no layers detected (changed-files mode with no Go changes) → exit cleanly with message.

## Step 2. Run `make lint` ONCE (shared baseline)

`make lint` covers the whole repo, so run it a single time and share the output with every auditor — never let each auditor re-run it (that would be N concurrent full-repo lints):

```sh
make lint 2>&1 | tee /tmp/arch-check-lint.out
```

If lint fails for reasons unrelated to the audited layers, surface the verbatim failure and stop (do not fan out auditors against a broken baseline).

## Step 3. Fan Out Auditor Subagents IN PARALLEL

For each layer in scope, spawn its auditor with the **Agent tool**, all in **a single message with multiple tool calls** so they run concurrently. Pass each auditor:

- `scope` — `changed` or `full`
- `files` — the pre-resolved newline list of in-scope `.go` files for that layer (from Step 1)
- `baseRef` — the base branch (fallback if the auditor must resolve files itself)
- `lintOutput` — `/tmp/arch-check-lint.out` (the shared run from Step 2 — auditors filter this instead of re-running lint)

Example (conceptual — adjust the touched set to the resolved layers):

```text
Agent(subagent_type="arch-auditor-domain",     prompt=<scope/files/baseRef/lintOutput for domain>)
Agent(subagent_type="arch-auditor-usecase",    prompt=<...usecase>)
Agent(subagent_type="arch-auditor-controller", prompt=<...controller>)
Agent(subagent_type="arch-auditor-infra",      prompt=<...infra>)
Agent(subagent_type="arch-auditor-pkg",        prompt=<...pkg>)
```

Each auditor's final message **is** its findings (Japanese, structured). Collect them with their layer label. An auditor that returns "違反なし" contributes an empty section.

> If the `arch-auditor-*` subagents cannot be spawned in the current environment, follow each `arch-auditor-<layer>.md` procedure inline instead — the integrator still performs the TODO write single-threaded afterward.

## Step 4. Aggregate Report

Combine all auditor findings into a single Japanese report:

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
```

If all clean:

```text
arch-check 統合結果（スコープ: <scope>）
全 layer で違反は検出されませんでした（チェック済み: <layer list>）。
```

## Step 5. TODO Hand-off Insertion (integrator-side, opt-in)

Auditor subagents are read-only and never write. If the user opted into "TODO 追加" at Step 0, **the integrator** now inserts the `// TODO:` hand-off comments — single-threaded, so there is no write contention. Apply only to **domain / controller / infra** `suggestion`-level findings (usecase / pkg are out of scope):

For each such suggestion (`file:line`):

1. Locate the deviation point in source (struct field line / handler method line / Repository method line).
2. Check the 3 lines immediately above for an existing comment block → if found, **skip** (de-dup).
3. Otherwise insert a `// TODO:` comment immediately before the deviation describing what was detected and listing resolution options for the human. Standard `// TODO:` prefix only — no AI-identifying prefix (`// TODO(arch-check):` 等は禁止).

The comment is a **hand-off baton**, not AI's judgment: the AI does not decide whether the deviation is intentional. `violation`-level findings do NOT get TODO comments (those must be fixed, not deferred). Report the add / skip counts:

```text
TODO hand-off: 追加 <sum> 件, スキップ <sum> 件（既存コメント）
```

If the user opted "TODO 追加なし", skip this step entirely (strictly read-only run).

## Step 6. Closing

- 検出は read-only auditor subagent に委譲。integrator の唯一の書き込みは opt-in 時の TODO hand-off コメントのみ
- `/commit` から chain 時は violations > 0 で non-zero status
- 単独実行時は情報的、exit 0
- 自動修正なし（violation の自動 fix はしない）

## AI Modification Scope

- 読み込み: 各 layer の README + 関連ファイル（auditor subagent が実施）、`make lint`（integrator が1回実行、`/tmp/arch-check-lint.out`）
- 書き込み: user opt 時のみ、`internal/{domain,controller,infrastructure}/**/*.go` の suggestion 位置への `// TODO:` hand-off コメント追加（**integrator が単一スレッドで実施**）。auditor subagent は一切書き込まない。

## Constraints

- ❌ auditor を逐次起動する（必ず1メッセージ内で複数 Agent 呼び出し＝並列）
- ❌ 各 auditor に `make lint` を再実行させる（共有 `lintOutput` を渡す）
- ❌ Skip scope + TODO opt `AskUserQuestion`
- ❌ Heuristic findings (handler bloat 等) を hard violation 扱い（auditor が `suggestion` ラベル付け、integrator は respect）
- ❌ violation 位置への TODO 書き込み（fix 必須、defer 不可）
- ❌ TODO に AI 識別 prefix を使う（`// TODO:` のみ）
- ❌ 既存コメントの上書き（3 行以内に既存あれば skip）
- ✅ Japanese aggregated report
- ✅ Fan out only touched layers (changed-files mode で効率化)
- ✅ Per-layer auditor / skill が独立 standalone 動作可能であることを維持
- ✅ TODO 書き込みは integrator 単一スレッドのみ（並列 auditor は read-only）

## Checklist

- [ ] Scope + TODO opt confirmed via `AskUserQuestion`
- [ ] Layer detection + per-layer file list resolved from changed files / full repo
- [ ] `make lint` を1回だけ実行し `/tmp/arch-check-lint.out` に保存
- [ ] Touched layers の `arch-auditor-*` を **1メッセージ内で並列起動**（scope / files / baseRef / lintOutput を渡す）
- [ ] 各 auditor が自身の README + lean A 規則を適用（read-only）
- [ ] Aggregated Japanese report 出力
- [ ] TODO hand-off は opt-in 時のみ integrator が単一スレッドで実施（domain / controller / infra の suggestion、既存コメントは skip）
- [ ] No commit / push
