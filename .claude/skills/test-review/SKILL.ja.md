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
- エラーは `require.*`（testifylint `require-error`）、終端値は `assert.*`。`require.NotNil` / `require.True` 等は、**後続コードをガードする**場合（そのまま使うと panic / 無意味になる。例: `require.NotNil(fn); fn(...)` や `require.NotNil(rec); _ = rec.Field`）のみ正しい。値を以降で使わないなら終端なので `assert.*` にすべき — 終端の `require.NotNil` / `require.True` / `require.Equal` は違反として指摘。
- 生成 mock を駆動するサブテストは mock controller 経由で assert している扱い: `EXPECT()`（メソッドが呼ばれないことを示す意図的な *no-EXPECT* や `.Times(0)` を含む）がアサーション。`assert.*` / `require.*` 行が無いというだけで「アサーション無し」と指摘しない。
- `for` ループ / table-driven が出てくる場合は明確な可読性理由が見える、それ以外は逐次 `t.Run` 前提。
- **subject 関数 ↔ `TestXxx` の 1:1 対応。** ここでの *subject* はペアになる本番ソース — バイナリにビルドされる非テスト・非生成の `.go` ファイル。`*.gen.go` / `*.sql.go` / `*_mock.go` とテスト専用ヘルパは対象外（手書き `TestXxx` を期待しない）。両方向を確認:
  - *順方向*: 各 `TestXxx` は 1 subject 関数 / メソッド対応。複数束ねる `TestXxx` は `scaffold-test/SKILL.md` 通り 1 行 rationale コメント必須。
  - *逆方向*: 各 subject 関数 / メソッドは 1 つの `TestXxx` に対応。1 つの subject が複数 `TestXxx` に分裂している（例: `TestFoo` + `TestFoo_Metrics` + `TestFoo_CloseError`、または `Test_foo` / `TestFoo_foo` の命名ゆらぎペア）のは finding → `正常系` / `異常系` group で variant を吸収する単一 `TestXxx` へ統合する (Rule 7)。`TestXxx` が 1 つも無い public subject 関数は網羅ギャップ（その分岐は Lens 4 軸A でも surface する）。
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
- **ケース名がアサートを過剰約束**: `t.Run` ケース名が本体で検証していない性質を名乗る — 例: 本体は `NotNil` だけなのに `"…を保持した収集器を返す"`。保持値が unexported フィールドにある / 別ユニットの責務、という場合に起きる。固有の性質を assert するか、検証している内容にケース名を合わせる。系: **分岐なしの pass-through / 配線関数**（入力をコンストラクタへ素通しするだけの DI provider 等）は honest な 1 ケースで十分 — 入力を変えて同じ `NotNil` を再実行するケースは分岐で区別されず網羅を増やさないので畳む。
- **内部結合の脆さ**: public API で済むのに unexported フィールドを読みに行く、`errors.Is` を使わず error メッセージ文字列で assert する。
- **過剰モック**: pure 実装で済む collaborator まで全て mock 化、call count の粒度が実装にロックインしている。
- **時刻リテラル漏れ**: `time.Now()` を assert 内で呼ぶ（固定 `baseTime` でなく）、system clock 依存比較。
- **`TestXxx` 責務肥大**: 1 `TestXxx` が複数 subject を回しているのに rationale が弱い（ルール違反でもあるが意味的 smell でもある）。
- **ヘルパ重複**: 5 行以上の同一 fixture が 3 `TestXxx` で繰り返されている → `t.Helper()` 化候補。
- **冗長コメント**: コードを言い換える／振る舞いでなく*なぜ*を語るインラインコメント。本プロジェクトはテストコメントを最小に保つ＝ケースの意図は日本語 `t.Run` 名に載せコメントに書かない。識別子言い換えコメントやテスト都合の説明がテスト本体に残っていれば指摘（1行 godoc 形式の宣言コメントは対象外、`-race` 直列ブロック例外コメントは必須＝冗長ではない）。

Output: `file:line` + 「なぜ弱い / 脆いか」の 1 文説明 finding リスト。

### Lens 4: 観点ギャップ — 分岐 × 意味 網羅 (subject 駆動)

subject ソースを直接読み、**関数 / メソッドごと**に 2 軸の網羅マトリクスを構築する。 カバレッジ ≠ 意味のあるカバレッジ: ある分岐が「その分岐固有の何か」を一切 assert しないケースで実行されているだけのことがあり、その穴を切り分けるのが本 lens の目的。 各 subject で両軸を走らせる — 分岐は「到達済み (軸A)」かつ「固有に assert 済み (軸B)」で初めて完了。

**軸A — 分岐網羅**: subject の各論理分岐が最低 1 件のケースで到達されているか。

- 各条件分岐（ポジ / ネガ）に最低 1 件の `t.Run` ケースがあるか。
- 宣言 / 返却される error sentinel (`ErrInvalid*` / `apperror.*`) が最低 1 件のケースで到達されているか。
- 境界制約を持つフィールドについて min-1 / min / max / max+1 の境界値が exercise されているか。
- zero 値 / nil 入力を防御している constructor / factory に reject ケースがあるか。
- テストの harness が**実行しない** constructor / provider / `fx.Invoke` 本体を通ってしか到達できない分岐は未カバー扱い。特に `fx.ValidateApp` は依存グラフを検証するがコンストラクタやライフサイクルフックを実行しない — グラフ検証テストでは provider / invoke 本体はカバーされず、分岐網羅には直接の単体テスト（関数を実際に呼ぶ）が必要。
- 「再現できない」と理由付けされた `t.Skip` は受け入れず軸A のギャップとして疑う: 統合スタイルの harness で到達できないか確認する（例: 実 DB のロック競合 / `55P03` は、直列化するテスト用 tx ヘルパでは並行を表現できないため 2 本の独立コネクション + トランザクションが要る）。skip 分岐を具体的な再現経路つきで 追加検討 として surface する。

カバーケースが**全く無い**分岐は **分岐未カバー** finding → severity **追加検討**（proactive）。 未カバー分岐の subject `file:line` + 提案 `t.Run` ケース名を引用。

**軸B — 意味網羅**: カバー済みの各分岐のケースが、単に実行されたことではなく*その分岐固有*の outcome を assert しているか。

- error 分岐は `require.Error` 止まりでなく `require.ErrorIs` で固有 sentinel を assert。
- 成功分岐は `require.NoError` / `assert.NotNil` 止まりでなく、他分岐と区別される結果値 / state を assert。
- state mutate メソッドのケースは「呼べた」止まりでなく mutate 後の不変条件 / 変化フィールドを assert。
- `ptr.Copy` を使うポインタ返却 getter は値等価だけでなく不変性を assert。
- 境界ケースは accept 側だけでなく境界の両側（accept vs reject）で異なる outcome を assert。

カバーは**されているが**固有 outcome を区別 assert していない分岐は **分岐カバー済み・意味未検証** finding → severity **再考**（pass して coverage は上がるが何も明らかにしない）。 finding は当該 subject 分岐 + それを名目上カバーしている test ケースに紐付ける。

Output: subject ごとに、(1) 未カバー分岐 + 提案 `t.Run` ケース名（軸A → 追加検討）、(2) カバー済みだが意味未検証の分岐 + 不足している区別アサーション（軸B → 再考）。 各々 subject 分岐の `file:line` とカバーしている test ケースを引用。

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

## 観点ギャップ: 分岐網羅（追加検討）
- <file> に対して subject <subject path> から導出:
  - 分岐未カバー: <subject file:line の分岐>
  - 提案: t.Run("<case name>", ...) — カバーする分岐 / sentinel: <reason>
  - verifier: CONFIRMED / PLAUSIBLE

## 観点ギャップ: 意味網羅（再考）
- <file> に対して subject <subject path> から導出:
  - 分岐カバー済み・意味未検証: <subject file:line の分岐> を <test file:line のケース> がカバーするが固有 outcome 未 assert
  - 不足アサーション: <あるべき distinctive assertion>
  - verifier: CONFIRMED / PLAUSIBLE

## 補遺
- pkg/ 層は Test Strategy 節を意図的に持たないため、sibling tests を比較基準にしています（gap 警告なし）。
- <その他、レビュー過程で気付いた README 補完候補 / SKILL の改訂候補>
```

severity マッピング:

- **修正必須** （構造準拠）: `CLAUDE.md` / `scaffold-test/SKILL.md` のハードルール違反。 CONFIRMED → 修正必須 / PLAUSIBLE → 確認推奨。
- **補完推奨** （観点カバレッジ）: README で宣言されているのに exercise されていない観点。 CONFIRMED → 補完推奨 / PLAUSIBLE → 確認推奨。
- **再考** （意味的品質 + 観点ギャップ 軸B）: pass はするが意味が薄い — 弱アサーション、または分岐はカバーされているが固有 outcome を区別 assert していない。 CONFIRMED → 再考 / PLAUSIBLE → 補強候補。
- **追加検討** （観点ギャップ 軸A）: subject 検査から派生する未カバー分岐の proactive 提案。 CONFIRMED → 追加検討 / PLAUSIBLE → 提案。

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
- [ ] Lens 4 は subject ごとに両軸を走らせた — 軸A 分岐網羅（未カバー分岐 → 追加検討）/ 軸B 意味網羅（カバー済みだが意味未検証 → 再考）。
- [ ] 各 finding が `review-verifier` を通過した（`skip_verifier: true` 親指定以外）。
- [ ] REFUTED は落とし、 CONFIRMED / PLAUSIBLE のみ残した。
- [ ] 最終レポートが日本語、 lens 別、 severity tag 付き。
- [ ] 次アクション提案が 1 つの具体提案。
- [ ] ファイル編集していない。
