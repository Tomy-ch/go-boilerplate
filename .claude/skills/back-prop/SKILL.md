---
name: back-prop
description: >-
  Integrator skill for drift detection across all layers. Confirms scope (changed files vs full repo) + detection categories (A/B/C/D, multi-select) once via `AskUserQuestion`, detects which layers are touched, resolves the per-layer file list, then fans out the relevant read-only `drift-detector-*` subagents (`drift-detector-domain` / `-usecase` / `-controller` / `-infra` / `-pkg`, plus the corpus-driven `drift-detector-ddd`) IN PARALLEL via the Agent tool — passing scope + resolved files + categories so each detector skips its own scope question. Detectors are STRICTLY read-only and only surface findings (A: README→Code, B: Code→README undocumented pattern, C: Skill↔README duplication, D: DDD pattern ledger ↔ ADR/README corpus) with reasoning + candidate options. The integrator then runs the per-item user-approval loop itself (single-threaded) and performs the README / SKILL / ledger writes after explicit per-item confirmation. Does NOT modify implementation code — code fixes are surfaced and left to the user. To check a single layer, run this integrator and pick that layer in the scope question. Recommended trigger: after multi-layer refactor or before major PR review, to confirm doc + skill remain in sync with code reality (priority README > Code > SKILL).
---

# Back-Prop

Integrator for drift detection across layers. Fans out per-layer **read-only drift-detector subagents** in parallel based on scope, aggregates, then drives the per-item approval + write loop itself.

A Japanese reference translation of this skill is available at `SKILL.ja.md` in the same directory (not loaded as a skill; for human reference only).

## When to Use

- After multi-layer refactor, before PR review
- Periodic hygiene sweep (catch undocumented conventions / skill bloat / README drift across all layers)
- When introducing a new layer-wide convention (to see where it's already followed / not yet)

To check a single layer, run this integrator and pick that layer in the scope question (「特定 layer のみ」).

Do NOT use for:

- Implementation code fixes (surface only; nothing here writes code)
- Architecture compliance per file — `arch-check` (with TODO hand-off)
- Spec validation — `verify-spec`

## Architecture: parallel detector subagents + integrator-side approval

Detection is delegated to five **read-only worker subagents** under `.claude/agents/`, one per layer. The integrator runs them concurrently via the Agent tool (`subagent_type`):

| Detector subagent | Layer | Canonical doc(s) |
| --- | --- | --- |
| `drift-detector-domain` | `internal/domain/**` | `internal/domain/README.md` |
| `drift-detector-usecase` | `internal/usecase/**` | `internal/usecase/README.md` + `boundary/README.md` |
| `drift-detector-controller` | `internal/controller/**` | `internal/controller/README.md` + `handler/README.md` (reference snippet) |
| `drift-detector-infra` | `internal/infrastructure/**` | infra / rdb / pgerror README（principles-focused、sibling code が de facto reference） |
| `drift-detector-pkg` | `pkg/**` | `pkg/README.md` + 各 `pkg/<name>/README.md` |

One further detector is keyed to the corpus rather than to a layer, and runs only when category (D) is selected:

| Detector subagent | 対象 | Canonical doc(s) |
| --- | --- | --- |
| `drift-detector-ddd` | `.agents/ddd-audit/pattern-ledger.yaml` ↔ ADR / README コーパス | 台帳自身の `corpus` グロブ |

これは README↔コードでなく**台帳↔正本**のドリフトを見る。台帳は「どの Evans パターンをどこで解釈したか」
の帳簿なので、正本が動いた瞬間に静かに嘘になる — しかも誰も読まないファイルなので、放置しても誰も気づかない。
Evans 原義に忠実かどうかは扱わない（`ddd-audit` / `ddd-origin-auditor` の担当）。

These detectors are the per-layer drift-detection workers. They are **strictly read-only**: they surface (A)(B)(C) findings with reasoning + candidate options, but they **never call `AskUserQuestion` and never write**. The approval + write loop runs in **this integrator**, **single-threaded after aggregation**, so the five read-only detectors can fan out in parallel with zero write contention. Priority remains **README > Code > SKILL**.

## First Step: Confirm Scope + Detection Categories

`AskUserQuestion` with two batched questions (defaults auto-detected by git diff):

1. 質問: 「back-prop のスコープを選んでください」
   - 選択肢: 「変更ファイルのみ（ベースブランチとの diff、touched layer のみ fan-out）」 / 「リポジトリ全体（5 layer 全部 fan-out）」 / 「特定 layer のみ（layer を続けて指定）」 / 「キャンセル」

2. 質問: 「検出する drift 種別を選んでください（multi-select、既定 4 種類すべて）」
   - 選択肢: 「(A) README → Code drift」 / 「(B) Code → README undocumented pattern」 / 「(C) Skill ↔ README duplication」 / 「(D) DDD 台帳 ↔ ADR/README コーパス」

Detection categories are propagated to every detector. (D) is the only one that does not depend on
Go files, so it is worth selecting even when the diff touches no code at all — an ADR-only change is
exactly the case that rots the ledger.

## Step 1. Resolve Layers + File Lists in Scope

For "changed files" mode:

```sh
BASE=$(gh pr view --json baseRefName -q '.baseRefName' 2>/dev/null || gh repo view --json defaultBranchRef -q '.defaultBranchRef.name')
git diff --name-only "origin/${BASE}...HEAD" -- '*.go' | grep -vE '\.gen\.go$|\.sql\.go$|_mock\.go$|_test\.go$'
```

Map to layers by path prefix, keeping the per-layer file list (you pass it to each detector so it does not re-resolve git):

| Path prefix | Detector subagent |
| --- | --- |
| `internal/domain/` | `drift-detector-domain` |
| `internal/usecase/` | `drift-detector-usecase` |
| `internal/controller/` | `drift-detector-controller` |
| `internal/infrastructure/` | `drift-detector-infra` |
| `pkg/` | `drift-detector-pkg` |

For "full repo": fan out all 5. For "specific layer": ask user which, fan out only those.

When (D) is selected, additionally resolve the **DDD corpus** — not `*.go`-filtered, since the corpus is prose. Read the `corpus` globs from `.agents/ddd-audit/pattern-ledger.yaml` (never hardcode them here) and intersect with the diff for `changed` scope. Add `drift-detector-ddd` to the fan-out whenever that intersection is non-empty, or always in `full` scope.

No Go changes **and** no corpus changes in changed-files mode → exit cleanly. Go-only changes with (D) selected still skip the DDD detector; a diff that touches no prose cannot rot the ledger.

## Step 2. Fan Out Detector Subagents IN PARALLEL

For each layer in scope, spawn its detector with the **Agent tool**, all in **a single message with multiple tool calls** so they run concurrently. Pass each detector:

- `scope` — `changed` or `full`
- `files` — the pre-resolved newline list of in-scope `.go` files for that layer (from Step 1)
- `baseRef` — the base branch (fallback)
- `categories` — the selected subset of `A` / `B` / `C`

`drift-detector-ddd` takes the corpus file list instead of a Go file list, and no `categories` (it
only ever detects (D)). Spawn it in the **same message** as the layer detectors so everything runs
concurrently.

Each detector's final message **is** its findings (Japanese, each with reasoning + candidate options). Collect them with their layer label.

> If the `drift-detector-*` subagents cannot be spawned in the current environment, follow each `drift-detector-<layer>.md` procedure inline instead — the integrator still runs the per-item approval + write loop (Step 4) single-threaded afterward.

## Step 3. Aggregate Findings (read-only checkpoint)

Combine all detector findings into a single Japanese summary grouped by layer + category, so the user sees the full surface before any decision:

```text
back-prop drift 検出結果（scope: <X>, 種別: A/B/C）

[domain]     A <n> / B <m> / C <k>
  ...（各 finding: rule・reasoning・options）
[usecase]    ...
[controller] ...
[infra]      ...
[pkg]        ...
[ddd]        D1 <n> / D2 <m> / D3 <k>

総 finding: <sum>。これから 1 件ずつ承認 / 棄却を確認します。
```

If all clean:

```text
back-prop drift 検出結果（scope: <X>, 種別: A/B/C）
全 layer で drift は検出されませんでした（チェック済み: <layer list>）。
```

## Step 4. Per-Item Approval + Write (integrator-side, single-threaded)

Detector subagents are read-only. For each finding, **the integrator** now drives the decision — there is no write contention because this loop is single-threaded:

1. `AskUserQuestion` with the candidate options the detector surfaced for that finding (e.g. コード修正 / README 更新 / ルール緩和 / skill 簡略化 / 無視).
2. If the user approves a **doc / skill** change:
   - State the reasoning explicitly, then show the draft as a diff (変更前 / 変更後).
   - After final confirmation, write via `Edit` / `Write` — only to the **canonical README** of that layer or the relevant **skill `SKILL.md`** (and never code).
3. If the user chooses a **code fix**: surface it as the user's task (this skill never writes implementation code).
4. Loop over all findings; the user may abort partway.

Write scope is restricted to: layer READMEs (`internal/<layer>/README.md` and sub-READMEs), skill `SKILL.md` files, and — for (D) findings only — `.agents/ddd-audit/pattern-ledger.yaml`. Never implementation code, never generated files, never `AGENTS.md`.

A (D) finding may tempt you to fix the corpus instead of the ledger — rewriting the ADR section the ledger points at, so the pointer becomes true again. Do not. The ledger is bookkeeping and yours to correct; an ADR is a decision record in the maintainer's voice, and a detector editing one would convert its own finding into the record of a decision nobody made. Surface that as the user's task.

After writes, run `make md-lint` (or `make md-fix` then `make md-lint`) to verify the edited Markdown.

## Step 5. Closing Report (Japanese)

```text
back-prop 完了（scope: <X>, 種別: A/B/C）

[domain]   findings <N>, README 更新 <X>, Skill 簡略化 <Y>, コード修正委任 <Z>, 無視 <W>
[usecase]  ...
[controller] ...
[infra]    ...
[pkg]      ...

総 finding: <sum>, README/Skill 書き込み: <sum>, コード修正委任: <sum>
最終 make md-lint OK
```

- 検出は read-only detector subagent に委譲。書き込みは integrator が per-item 承認後に単一スレッドで実施
- 実装コードへの書き込みは一切なし（surface のみ、コード修正は user 作業）
- commit / push なし

## AI Modification Scope

- 読み込み: 各 layer の README + 実装 + 関連 skill 本体（detector subagent が実施）
- 書き込み: **integrator のみ**、user の per-item 承認 + 理由明示 + draft 提示の後に、layer README / 関連 skill `SKILL.md` / (D) 時のみ `.agents/ddd-audit/pattern-ledger.yaml` へ。detector subagent は一切書き込まない。
- 触らない: 実装コード、生成物、`AGENTS.md`、ADR 本文

## Constraints

- ❌ detector を逐次起動（必ず1メッセージ内で複数 Agent 呼び出し＝並列）
- ❌ scope + 種別 `AskUserQuestion` をスキップ
- ❌ detector に書き込み / `AskUserQuestion` をさせる（read-only surface 専用）
- ❌ user 承認なしの README / skill 自動更新
- ❌ 理由を述べずに draft を実行
- ❌ 実装コードへの書き込み（surface のみ、修正は user）
- ❌ recurring threshold 3 未満の (B) pattern を surface（detector 側で抑止、integrator も respect）
- ❌ (D) の解消として ADR / README 本文を書き換える（台帳側を直す。正本の変更は user 作業）
- ❌ (D) で Evans 原義への忠実性を判定する（`ddd-audit` の担当）
- ✅ Japanese aggregated report
- ✅ Fan out only touched layers (changed-files mode)
- ✅ per-layer detector / skill が独立 standalone 動作可能であることを維持
- ✅ Categories propagation to all detectors
- ✅ 書き込みは integrator 単一スレッドのみ（並列 detector は read-only）
- ✅ README が canonical の前提（README > Code > SKILL）

## Checklist

- [ ] Scope + 種別を `AskUserQuestion` で確認
- [ ] Layer + per-layer ファイルリスト解決（changed files / full repo / specific layer）
- [ ] touched layer の `drift-detector-*` を **1メッセージ内で並列起動**（scope / files / baseRef / categories を渡す）
- [ ] (D) 選択かつコーパス変更ありなら `drift-detector-ddd` を同じメッセージで並列起動
- [ ] 各 detector が README + 実装 + skill を読み (A)(B)(C) を read-only 検出
- [ ] 集約サマリ出力（決定前のチェックポイント）
- [ ] integrator が per-item で reasoning + user 承認 + draft + 最終確認 + 書き込み（README / skill のみ）
- [ ] 実装コードへの書き込みなし
- [ ] 最終 make md-lint OK
- [ ] commit / push なし
