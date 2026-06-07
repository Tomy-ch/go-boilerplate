> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# Back-Prop

全 layer drift 検出 integrator。scope に応じて per-layer skill を chain。

## 使うとき

- multi-layer refactor 後、PR review 前
- 定期 hygiene sweep（全 layer での未文書化 convention / skill 肥大 / README drift を一括検出）
- layer 横断の新 convention 導入時（どこで既に追従済み / 未対応か把握）

単一 layer のみのとき: `back-prop-<layer>` を直接使う。

以下の用途には使いません: 実装コード修正 / 単一ファイルアーキ準拠（`arch-check`） / spec validation（`verify-spec`）。

## chain される per-layer skill

| Skill | Layer | 備考 |
| --- | --- | --- |
| `back-prop-domain` | `internal/domain/**` | README は Implementation notes / Aggregate Design / Testing strategy / Do / Don't 等で構成 |
| `back-prop-usecase` | `internal/usecase/**` | README 末尾に Implementation Example あり |
| `back-prop-controller` | `internal/controller/**` | handler README に reference snippet あり |
| `back-prop-infra` | `internal/infrastructure/**` | 3-level READMEs（infra / rdb / pgerror）、完全 snippet 無し（sibling コードが de facto reference） |
| `back-prop-pkg` | `pkg/**` | layer README + sub-package READMEs 必須 |

## 最初のステップ: スコープ + 検出種別確認

`AskUserQuestion` 2 質問 batched（既定は git diff から自動推定）:

1. 質問: 「back-prop のスコープを選んでください」
   - 選択肢: 「変更ファイルのみ（ベースブランチとの diff、touched layer のみ chain）」 / 「リポジトリ全体（5 layer 全部 chain）」 / 「特定 layer のみ（layer を続けて指定）」 / 「キャンセル」

2. 質問: 「検出する drift 種別を選んでください（multi-select、既定 3 種類すべて）」
   - 選択肢: 「(A) README → Code drift」 / 「(B) Code → README undocumented pattern」 / 「(C) Skill ↔ README duplication」

検出種別は全 chained child に伝搬。

## Step 1. layer 解決

「変更ファイル」モード:

```sh
BASE=$(gh pr view --json baseRefName -q '.baseRefName' 2>/dev/null || gh repo view --json defaultBranchRef -q '.defaultBranchRef.name')
git diff --name-only "origin/${BASE}...HEAD" -- '*.go' | grep -vE '\.gen\.go$|\.sql\.go$|_mock\.go$|_test\.go$'
```

path prefix → layer:

| Path prefix | Skill |
| --- | --- |
| `internal/domain/` | `back-prop-domain` |
| `internal/usecase/` | `back-prop-usecase` |
| `internal/controller/` | `back-prop-controller` |
| `internal/infrastructure/` | `back-prop-infra` |
| `pkg/` | `back-prop-pkg` |

「リポジトリ全体」: 5 layer 全部 chain。「特定 layer」: user に指定を求める。

## Step 2. per-layer skill chain

scope 内 layer ごとに `back-prop-<layer>` を `Skill` tool で invoke（scope + 種別 context 渡し、child は自身の `AskUserQuestion` スキップ）。

各 child は独立に動作、**per-item の user 承認は child 内で実施**（AI が理由 + draft 提示 → user 確認 → README / Skill 書き込み）。

各 child の report 収集。

## Step 3. 集約レポート（日本語）

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

全 clean 時:

```text
back-prop 統合結果（scope: <X>, 種別: A/B/C）
全 layer で drift は検出されませんでした（チェック済み: <layer list>）。
```

## Step 4. クロージング

- Integrator は何も書かない、すべて child skill 経由
- 実装コードへの書き込みは child も含めて一切なし
- README / Skill 書き込みは child 内で user 承認後

## AI 修正スコープ

Integrator 自身は何も書かない。書き込みスコープは全 per-layer skill に委譲（各 child は自 layer の READMEs + skill SKILL.md のみ、per-item user 承認 + 理由明示済み）。

## 制約事項

- ❌ ソースファイル直接読まない（child 任せ）
- ❌ scope + 種別 `AskUserQuestion` をスキップ
- ❌ Integrator 自身による file 書き込み
- ❌ child の per-item 承認をスキップさせない（integrator は children に context 渡すだけ）
- ✅ Japanese aggregated report
- ✅ touched layer のみ chain（changed-files モード）
- ✅ child が standalone 動作可能であることを維持
- ✅ Categories propagation to all children

## チェックリスト

- [ ] Scope + 種別を `AskUserQuestion` で確認
- [ ] Layer 検出（changed files / full repo / specific layer）
- [ ] touched layer の back-prop-<layer> を chain
- [ ] 各 child が自身の README + 実装 + skill を読み、(A)(B)(C) 検出
- [ ] 各 child が per-item で reasoning + user 承認 + 書き込み
- [ ] 集約 Japanese report 出力
- [ ] Integrator 自身による file 書き込みなし
- [ ] commit / push なし
