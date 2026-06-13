> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# Back-Prop

layer 横断の drift 検出の統合スキル。scope に応じて per-layer **read-only drift-detector サブエージェント**を並列 fan-out し、集約後に per-item 承認＋書込ループを integrator 自身が回す。

## 使うとき

- 複数 layer の refactor 後、PR レビュー前
- 定期的な衛生チェック（未文書化規約 / skill 肥大 / README drift を横断検出）
- layer 横断の新規約を導入する時（どこで既に守られ / まだか確認）

単一 layer だけ確認したい時はこの統合スキルを実行し、scope 質問で「特定 layer のみ」を選ぶ。

以下には使わない:

- 実装コード修正（surface のみ、ここでは何もコードを書かない）
- ファイル単位のアーキ適合性 — `arch-check`（TODO hand-off 付き）
- spec validation — `verify-spec`

## アーキテクチャ: 並列 detector サブエージェント + integrator 側承認

検出は `.claude/agents/` 配下の **read-only ワーカーサブエージェント**（layer ごとに1つ）へ委譲。integrator は Agent tool（`subagent_type`）でこれらを並列起動する:

| detector サブエージェント | layer | canonical doc |
| --- | --- | --- |
| `drift-detector-domain` | `internal/domain/**` | `internal/domain/README.md` |
| `drift-detector-usecase` | `internal/usecase/**` | `internal/usecase/README.md` + `boundary/README.md` |
| `drift-detector-controller` | `internal/controller/**` | `internal/controller/README.md` + `handler/README.md`（reference snippet） |
| `drift-detector-infra` | `internal/infrastructure/**` | infra / rdb / pgerror README（principles 主体、sibling code が de facto reference） |
| `drift-detector-pkg` | `pkg/**` | `pkg/README.md` + 各 `pkg/<name>/README.md` |

これらは層別の drift 検出ワーカー。**厳密に read-only**: (A)(B)(C) findings を reasoning + 候補オプション付きで surface するが、**`AskUserQuestion` を呼ばず・書き込まない**。承認＋書込ループは**この integrator が集約後に単一スレッドで**回す。これにより read-only detector 5つを書込競合ゼロで並列 fan-out できる。優先順位は **README > Code > SKILL**。

## 最初のステップ: scope + 検出種別 確認

`AskUserQuestion` を 2 質問 batched（既定は git diff で自動検出）:

1. 質問: 「back-prop のスコープを選んでください」
   - 選択肢: 「変更ファイルのみ（diff、touched layer のみ fan-out）」 / 「リポジトリ全体（5 layer 全部 fan-out）」 / 「特定 layer のみ」 / 「キャンセル」

2. 質問: 「検出する drift 種別を選んでください（multi-select、既定 3 種類すべて）」
   - 選択肢: 「(A) README → Code drift」 / 「(B) Code → README undocumented pattern」 / 「(C) Skill ↔ README duplication」

検出種別は全 detector に伝播。

## Step 1. scope を layer + ファイルリストに解決

「変更ファイルのみ」モード:

```sh
BASE=$(gh pr view --json baseRefName -q '.baseRefName' 2>/dev/null || gh repo view --json defaultBranchRef -q '.defaultBranchRef.name')
git diff --name-only "origin/${BASE}...HEAD" -- '*.go' | grep -vE '\.gen\.go$|\.sql\.go$|_mock\.go$|_test\.go$'
```

path prefix で layer マッピングし、per-layer ファイルリストを保持（各 detector へ渡し git 再解決を防ぐ）:

| path prefix | detector サブエージェント |
| --- | --- |
| `internal/domain/` | `drift-detector-domain` |
| `internal/usecase/` | `drift-detector-usecase` |
| `internal/controller/` | `drift-detector-controller` |
| `internal/infrastructure/` | `drift-detector-infra` |
| `pkg/` | `drift-detector-pkg` |

「リポジトリ全体」: 5 detector 全部 fan-out。「特定 layer のみ」: layer を追加質問し該当のみ。changed-files で Go 変更なし → 明示メッセージで exit。

## Step 2. detector サブエージェントを並列 fan-out

scope 内の各 layer について、その detector を **Agent tool** で起動。**1メッセージ内に複数 tool 呼び出し**を並べて並列実行。各 detector に渡す:

- `scope` — `changed` or `full`
- `files` — その layer の in-scope `.go` ファイル改行リスト（Step 1）
- `baseRef` — base ブランチ（fallback）
- `categories` — 選択された `A` / `B` / `C` の部分集合

各 detector の最終メッセージ**が** findings（日本語、各 finding に reasoning + 候補オプション）。layer ラベル付きで収集。

> `drift-detector-*` を起動できない環境では、各 `drift-detector-<layer>.md` の手順を本文がインラインで実行する — 承認＋書込（Step 4）は集約後に integrator が単一スレッドで行う。

## Step 3. findings 集約（read-only チェックポイント）

全 detector findings を layer + category 別の日本語サマリに集約し、決定前に全体像を提示:

```text
back-prop drift 検出結果（scope: <X>, 種別: A/B/C）

[domain]     A <n> / B <m> / C <k>
  ...（各 finding: rule・reasoning・options）
[usecase]    ...
[controller] ...
[infra]      ...
[pkg]        ...

総 finding: <sum>。これから 1 件ずつ承認 / 棄却を確認します。
```

全 clean:

```text
back-prop drift 検出結果（scope: <X>, 種別: A/B/C）
全 layer で drift は検出されませんでした（チェック済み: <layer list>）。
```

## Step 4. per-item 承認 + 書込（integrator 側、単一スレッド）

detector は read-only。各 finding について **integrator** が決定を回す（単一スレッドなので書込競合なし）:

1. detector が surface した候補オプションで `AskUserQuestion`（例: コード修正 / README 更新 / ルール緩和 / skill 簡略化 / 無視）。
2. user が **doc / skill** 変更を承認した場合:
   - 理由を明示 → draft を diff（変更前 / 変更後）で提示。
   - 最終確認後、`Edit` / `Write` で書込 — 当該 layer の **canonical README** または関連 **skill `SKILL.md`** のみ（コードは決して触らない）。
3. user が **コード修正**を選んだ場合: user の作業として surface（このスキルは実装コードを書かない）。
4. 全 finding をループ。途中 abort 可。

書込スコープ: layer README（`internal/<layer>/README.md` とサブ README）と skill `SKILL.md` のみ。実装コード・生成物・`AGENTS.md` は不可。

書込後に `make md-lint`（必要なら `make md-fix` → `make md-lint`）で編集 Markdown を検証。

## Step 5. クロージングレポート（日本語）

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

- 検出は read-only detector サブエージェントに委譲。書込は integrator が per-item 承認後に単一スレッドで実施
- 実装コードへの書込は一切なし（surface のみ、コード修正は user 作業）
- commit / push なし

## AI 修正スコープ

- 読み込み: 各 layer の README + 実装 + 関連 skill 本体（detector サブエージェントが実施）
- 書き込み: **integrator のみ**、user の per-item 承認 + 理由明示 + draft 提示の後に、layer README / 関連 skill `SKILL.md` へ。detector サブエージェントは一切書き込まない。
- 触らない: 実装コード、生成物、`AGENTS.md`

## 制約事項

- ❌ detector を逐次起動（必ず1メッセージ内で複数 Agent 呼び出し＝並列）
- ❌ scope + 種別 `AskUserQuestion` をスキップ
- ❌ detector に書き込み / `AskUserQuestion` をさせる（read-only surface 専用）
- ❌ user 承認なしの README / skill 自動更新
- ❌ 理由を述べずに draft を実行
- ❌ 実装コードへの書き込み（surface のみ、修正は user）
- ❌ recurring threshold 3 未満の (B) pattern を surface（detector 側で抑止、integrator も respect）
- ✅ Japanese aggregated report
- ✅ touched layer のみ fan-out（changed-files mode）
- ✅ per-layer detector / skill が独立 standalone 動作可能であることを維持
- ✅ Categories を全 detector に propagate
- ✅ 書き込みは integrator 単一スレッドのみ（並列 detector は read-only）
- ✅ README が canonical の前提（README > Code > SKILL）

## チェックリスト

- [ ] scope + 種別を `AskUserQuestion` で確認
- [ ] layer + per-layer ファイルリスト解決（changed files / full repo / specific layer）
- [ ] touched layer の `drift-detector-*` を **1メッセージ内で並列起動**（scope / files / baseRef / categories を渡す）
- [ ] 各 detector が README + 実装 + skill を読み (A)(B)(C) を read-only 検出
- [ ] 集約サマリ出力（決定前のチェックポイント）
- [ ] integrator が per-item で reasoning + user 承認 + draft + 最終確認 + 書き込み（README / skill のみ）
- [ ] 実装コードへの書き込みなし
- [ ] 最終 make md-lint OK
- [ ] commit / push なし
