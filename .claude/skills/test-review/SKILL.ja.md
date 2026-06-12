> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# Test Review

Go ユニットテストファイル (`*_test.go`) の adversarial / low-bias レビュースキル。read-only。「壊れて見える / 検証不足 / 過剰検証」を指摘し、修正は user に委ねる。

## 使うとき

- commit / PR 直前、現在の change に含まれる `*_test.go` に対して。
- `scaffold-test` 実行後、生成テストの独立 second opinion として。
- カバレッジ 100 % のまま回帰バグが出る場合（観点が構造的には準拠でも意味的に薄い signal）。
- 特定 test パッケージ / ファイルの単発監査として。

使わない場面:

- **実装コードのレビュー** → `code-review` / `local-review` / `arch-check`。
- **HTTP 統合テスト** (`internal/integration/` 配下) → 別 convention（`internal/integration/README.md` + `scaffold-integration-test`）。本スキルは同一パッケージ unit test 専用。
- 修正の適用 → 本スキルはファイル編集しない。指摘後、user が `scaffold-test` 実行 or 手で編集する。

## 読む / 書く

**常に読む**:

- `CLAUDE.md` の Testing Instructions。
- `.claude/skills/scaffold-test/SKILL.md` — 生成側 canonical ルール（parallel 必須 / `t.Run` per subcase / 正常系・異常系 グルーピング / 日本語命名 / require vs assert / mock 方針 / for-loop 方針 / 1 関数 = 1 `TestXxx`）。本スキルはこれらに対してレビューする（重複定義しない）。
- 対象ファイルから walk up で解決した層 README:
  - `internal/domain/README.md`（Testing strategy）
  - `internal/usecase/README.md`（Testing Strategy）
  - `internal/controller/handler/README.md`（Test Strategy）
  - `internal/infrastructure/README.md` + `internal/infrastructure/rdb/README.md`（Test Strategy）
  - `pkg/**` については `scaffold-test/SKILL.md` 参照 — `pkg/README.md` は意図的に Test Strategy 節を持たず、sibling tests + `pkg/<name>/README.md` から観点派生。**pkg では gap 警告を出さない**。
- 対象 `*_test.go`。
- 対応する subject ソース (`<subject>.go` / `<subject>_test.go` 対）— 観点ギャップ lens で「何が test されていないか」を判定するために必須。
- 同一パッケージの sibling test（成立済みの conventions の参考）。

**書く**:

- 何も書かない。read-only。 最終出力は conversation 上に展開される日本語レポート。

**`make` 起動**: なし（実行はしない、レビューだけ）。

**触らない**:

- test ファイル（自動 fix なし → `scaffold-test` or 手編集の領分）。
- subject ソース。
- 生成成果物 (`*.gen.go` / `*_mock.go` / `*.sql.go`)。
- `.claude/` 配下。

## First Step: スコープ解決

`AskUserQuestion`:

- 質問: 「test-review の対象スコープを指定してください」
- 選択肢（single-select）:
  - 「変更ファイル (HEAD-vs-working tree, 推奨)」 — `git diff --name-only` で `*_test.go` 抽出（`local-review` / `code-review` と同じ振る舞い）。新規追加 (`--diff-filter=A`) も含める。
  - 「ブランチ base 比較 (release/v1.x.y 以降の変更)」 — `git merge-base` で base 解決、その間に touched された `*_test.go`。 PR 単位で見たいとき。
  - 「特定パス / パッケージ (free-text)」 — user 指定。 ファイルでもディレクトリでも可。
  - 「キャンセル」。

解決後、対象ファイルリストを構築。スコープ内に `*_test.go` がなければ穏便に停止（レビュー対象なし）。

各 test ファイルについて **subject ソース**（同パッケージ・basename `_test` 抜き）も解決。観点ギャップ lens に必須。

## Step 1. 層コンテキスト読み込み

各対象 test ファイルにつき:

1. ファイルパスから層検出（band lookup は `scaffold-test/SKILL.md` と同じ）。
2. 該当層 README の `Test Strategy` / `Testing strategy` 節（全文、サブセクション見出し含む）。
3. `CLAUDE.md` Testing Instructions（1 回）。
4. `.claude/skills/scaffold-test/SKILL.md`（1 回、canonical ルール）。
5. subject ソース。
6. 同パッケージの sibling test（helper シグネチャ / fixture スタイル / mock 配線 等の確立 convention）。

`internal/<layer>/` で Test Strategy 節が期待されているのに無い場合は report に notes として記録（pkg は除外、警告ではなく documentation gap として）。

## Step 2. 4 つの adversarial reviewer を fan out

`adversarial-reviewer` subagent を 4 つ、**並列**起動 (`subagent_type: adversarial-reviewer`)。 既定で `sonnet`（Opus 実装者 ≠ reviewer を保つ。orchestrator がモデルを上書きして reviewer ≠ implementer を維持してもよい）。

各 subagent は Step 1 の context bundle（層 README / `CLAUDE.md` / `scaffold-test/SKILL.md` / 対象 test / subject / sibling tests）を共通で受け取り、レンズだけ違う prompt を受ける。

### Lens 1: 構造準拠

機械的なルール準拠を監査（`scaffold-test/SKILL.md` のハードルール）:

- 全 `t.Run` の冒頭が `t.Parallel()`（または `-race` 例外コメント付きの直列化）。
- 全サブケースが `t.Run`（裸のアサーション禁止）。
- `TestXxx` 最外殻 `t.Run` group の name は literal `正常系` / `異常系` （各 `TestXxx` に最大 1 個ずつ、 さらに細分するならその **内側に** ネストする）。 最外殻に `t.Run("正常系_xxx", ...)` 形式を使っているのは違反 → finding として出す。 内側のサブケース名にも `正常系_` / `異常系_` プレフィックスは禁止（外殻 group で既に区別済みのため二重ラベル）。
- ケース名は日本語。
- エラーは `require.*`（testifylint `require-error`）、終端値は `assert.*`。
- `for` ループ / table-driven が出てくる場合は明確な可読性理由が見える、それ以外は逐次 `t.Run` 前提。
- 各 `TestXxx` は 1 subject（関数 / メソッド）対応 — 例外的に複数束ねている場合は `scaffold-test/SKILL.md` 通り 1 行 rationale コメント必須。
- mock は `<package>/mock/*_mock.go` から（手書き mock 禁止）。
- 層別禁則 import（`pkg/**` test から `internal/` 参照禁止、`internal/domain/` test から infrastructure 参照禁止 等、`CLAUDE.md` ルール）。

Output: `file:line` 付きの違反 finding リスト + 違反したルール。

### Lens 2: 観点カバレッジ

層 README の Test Strategy サブセクションと実 test を突き合わせ:

- README の各サブセクション見出し（`### Getter contract test` / `### Immutable guarantee test` / `### Invariant preservation test` 等）に対応する `TestXxx → t.Run(case)` が 1 件以上あるか。
- README にサブセクションがない場合（`pkg/**`）は sibling test パターンを比較基準にする。
- 現状 READMEs から派生する具体例（記述的、ハードコードではない — README が変われば reviewer も追従）:
  - domain: ポインタ不変性 test（`Immutable guarantee test` と `TestImmutableAccessors` 参考）、状態遷移における不変条件保存、VO 境界。
  - usecase: orchestration の mock 呼び出し順序、トランザクション境界の適用、boundary 呼び出し。
  - controller: HTTP I/O 変換、validation、apperror → status、middleware が乗せる context（auth principal / request id）。
  - infra: SQL 実行経路、`pgerror.NormalizeError` 適用、row → entity 変換。

Output: README が宣言しているが test が exercise していない観点リスト（`file:section` で README 該当節を引用）。

### Lens 3: 意味的品質

アサーションが本当に意味を持っているかを監査:

- **弱いアサーション**: 複雑な戻り値に対する唯一の確認が `assert.NotNil(t, x)`、`assert.NoError` 後に state 確認なし、`assert.Equal(t, len(actual), 1)` のように要素自体を assert していない 等。
- **内部結合の脆さ**: public API で済むのに unexported フィールドを読みに行く、`errors.Is` を使わず error メッセージ文字列で assert する。
- **過剰モック**: pure 実装で済む collaborator まで全て mock 化、call count の粒度が実装にロックインしている。
- **時刻リテラル漏れ**: `time.Now()` を assert 内で呼ぶ（固定 `baseTime` でなく）、system clock 依存比較。
- **`TestXxx` 責務肥大**: 1 `TestXxx` が複数 subject を回しているのに rationale が弱い（ルール違反でもあるが意味的 smell でもある）。
- **ヘルパ重複**: 5 行以上の同一 fixture が 3 `TestXxx` で繰り返されている → `t.Helper()` 化候補。

Output: `file:line` + 「なぜ弱い / 脆いか」の 1 文説明 finding リスト。

### Lens 4: 観点ギャップ (subject 駆動)

subject ソースを直接読み、 既存 test がカバーしていない case を提案:

- subject の各条件分岐に対して、ポジ / ネガ少なくとも 1 件のカバーケースがあるか。
- subject 内で宣言 / 返却されている error sentinel (`ErrInvalid*` / `apperror.*`) が `require.ErrorIs` で少なくとも 1 件 assert されているか。
- subject が境界制約を持つフィールドについて、min-1 / min / max / max+1 の境界値 test があるか。
- domain で `ptr.Copy` を使ったポインタ返却 getter に不変性 test があるか。
- 状態 mutate メソッドに「mutate 後の不変条件 hold」確認があるか。
- 防御している場合、constructor / factory に「zero 値 / nil 入力 reject」 test があるか。

Output: 既存 test に追加すべき `t.Run` ケース提案リスト（提案ケース名 + カバーする分岐 / sentinel + rationale）。

## Step 3. 各 finding を検証

生き残った各 finding を独立した `review-verifier` subagent (`subagent_type: review-verifier`、 sonnet) に渡す。 verifier は:

- コードから結論を **再導出**（finder の出力は信用しない）。
- 既定で skeptical — 曖昧なら CONFIRMED より PLAUSIBLE / REFUTED 寄り。
- **CONFIRMED**（違反 / 漏れ / 弱アサーションが実在し再現可能） / **PLAUSIBLE**（妥当そうだが verifier が完全に再導出しきれなかった） / **REFUTED**（counter-evidence — 引用行がコメント、見出しが別層、文脈考慮で十分なアサーション 等）に分類。

検証は finding 並列実行 (`parallel(findings.map(f => () => agent(verifyPrompt(f))))`)。 reviewer ≠ implementer を保つために finder と verifier で異なるモデルを使うのもよい。

REFUTED は report から落とす（合計件数だけ summary に残す → noise floor を見せる）。 CONFIRMED / PLAUSIBLE は lens ごとに残してレポート合成。

## Step 4. レポート合成

単一の日本語レポート。推奨構造:

```text
# Test Review レポート

対象: <スコープ + ファイル一覧>
レンズ: 構造準拠 / 観点カバレッジ / 意味的品質 / 観点ギャップ
verifier 通過: CONFIRMED <n> 件 / PLAUSIBLE <m> 件 / REFUTED <k> 件 (フィルタ済み)

## サマリ
- 修正必須: <件数>
- 補完推奨: <件数>
- 再考: <件数>
- 追加検討: <件数>

## 構造準拠（修正必須）
- [<severity>] <file>:<line> — <violated rule>
  - 詳細: <description>
  - 出典: `<README path>` / `scaffold-test/SKILL.md` の該当節
  - verifier: CONFIRMED / PLAUSIBLE

## 観点カバレッジ（補完推奨）
- [<severity>] <test file> — <missing viewpoint>
  - 出典 README: `<README path>:<section heading>`
  - 提案: <suggested t.Run case name>
  - verifier: CONFIRMED / PLAUSIBLE

## 意味的品質（再考）
- [<severity>] <file>:<line> — <weak assertion / brittle coupling>
  - 詳細: <one-sentence why>
  - verifier: CONFIRMED / PLAUSIBLE

## 観点ギャップ（追加検討）
- <file> に対して subject <subject path> から導出:
  - 提案: t.Run("<case name>", ...) — カバーする分岐 / sentinel: <reason>
  - verifier: CONFIRMED / PLAUSIBLE

## 補遺
- pkg/ 層は Test Strategy 節を意図的に持たないため、sibling tests を比較基準にしています（gap 警告なし）。
- <その他、レビュー過程で気付いた README 補完候補 / SKILL の改訂候補>
```

severity マッピング:

- **修正必須** （構造準拠）: `CLAUDE.md` / `scaffold-test/SKILL.md` のハードルール違反。 CONFIRMED → 修正必須 / PLAUSIBLE → 確認推奨。
- **補完推奨** （観点カバレッジ）: README で宣言されているのに exercise されていない観点。 CONFIRMED → 補完推奨 / PLAUSIBLE → 確認推奨。
- **再考** （意味的品質）: pass はするが意味が薄い。 CONFIRMED → 再考 / PLAUSIBLE → 補強候補。
- **追加検討** （観点ギャップ）: subject 検査から派生する proactive 提案。 CONFIRMED → 追加検討 / PLAUSIBLE → 提案。

## Step 5. 次アクション提案

レポート末尾に 1 つだけ具体提案:

- 「修正必須」がある → 該当 subject に対して `/scaffold-test` 再実行（canonical 構造にリセット）または該当行を手で修正、を提案。
- 「補完推奨」 / 「追加検討」 のみ → 提案された `t.Run` を手で追加するか、対象 subject を絞った `/scaffold-test` で補完するか、を提案。
- verifier 通過後 0 件 → その旨明示（`「verifier 通過後 0 件です」`）。

## Chainability

PR レビューフローで `code-review` / `local-review` / `arch-check` と並んで使う想定。 現状で他スキルを *chain in* する経路は持たない — user がレポートを読んで `scaffold-test`（再生成）or 手編集を判断する。 将来 PR レビュー orchestrator が `code-review` + `arch-check` + `test-review` を並列 fan out して結果統合する可能性はあるが、 まだ存在しない。

将来そういう親から chain される場合、 親は最低限以下を渡す:

- `scope` — 解決済みファイルリスト（First Step の `AskUserQuestion` をスキップ）。
- `base_ref` — branch-vs-base モード時の base ref。
- `skip_verifier` — boolean。 親が速度優先で verify を切る場合（既定 `false` = verify する）。

## 制約（サマリ）

- ❌ ファイル編集（read-only）。
- ❌ `make test` 実行（本スキルはレビュー、実行は `make test` 別途）。
- ❌ verifier を通さず finder 出力をそのまま信頼（`skip_verifier: true` 親指定以外、verifier は必須）。
- ❌ 観点リストのハードコード（SSOT は層 README の Test Strategy 節、`pkg/` は明文化された例外）。
- ❌ `CLAUDE.md` / `scaffold-test/SKILL.md` のルール重複定義（runtime 読み込み）。
- ✅ verifier は skeptical 既定（曖昧時は PLAUSIBLE 寄り）。
- ✅ reviewer 既定モデル `sonnet`（Opus 実装者と異なる）。 必要なら orchestrator が上書きして reviewer ≠ implementer を維持。
- ✅ スコープ既定: 変更ファイル。 他選択肢あり。
- ✅ 最終レポートは日本語、 lens 別グルーピング、 severity tag。
- ✅ `pkg/` は意図的に「Test Strategy 節なし」層として扱い、 documentation gap として警告しない。

## チェックリスト

完了報告前に確認:

- [ ] スコープ解決済み（変更ファイル / base diff / 明示パス）。
- [ ] 各対象 `*_test.go` に対し subject ソースが解決済み。
- [ ] Step 1 で 層 README + `CLAUDE.md` + `scaffold-test/SKILL.md` + sibling tests を読んだ。
- [ ] 4 レンズが（並列で）走った。
- [ ] 各 finding が `review-verifier` を通過した（`skip_verifier: true` 親指定以外）。
- [ ] REFUTED は落とし、 CONFIRMED / PLAUSIBLE のみ残した。
- [ ] 最終レポートが日本語、 lens 別、 severity tag 付き。
- [ ] 次アクション提案が 1 つの具体提案。
- [ ] ファイル編集していない。
