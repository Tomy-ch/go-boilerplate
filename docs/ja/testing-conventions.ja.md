# テスト規約

本書はこのリポジトリで **テストをどう書くか** の唯一の source of truth です — 構造・命名・
並列化・アサーション・mock・カバレッジ例外のガバナンス。`scaffold-test`(生成)と
`test-review`(レビュー)の各 skill が runtime で本書を読むため、ここに一元化して drift を防ぎます。

スコープ分担(以下を跨いで重複させないこと):

- **本書** — 具体的な *どう書くか*(技法 / 規約)。
- [`rules.md` → *Testing & Definition of Done*](rules.ja.md) — 非交渉の *どうなれば完了か*(層ごとのテスト・90% ライン・「compiles ≠ done」・実行時 DI 検証・実アプリの smoke test・到達不能分岐の方針)。
- 各層 `README` → *Test Strategy* — 層ごとの **観点**(その層で何を検証するか)。

canonical な参照テストは [`internal/domain/user/user_domain_test.go`](../../internal/domain/user/user_domain_test.go) です。

## 1. 構造

- **1 つの関数 / メソッドにつき 1 つの `TestXxx` — 厳密に 1:1、束ねない。** getter / accessor も同様: `*_Accessors` / `*_Getters` に束ねず、accessor ごとに 1 つの `TestXxx` を用意する。この 1:1 対応は、production の全関数 / メソッドについて `internal/architest`（`TestUnitTestMappingCompleteness`）が機械的に強制する。分岐なしも対象で、`if` が無い body でも契約は持ちうる（ルート登録だけを行う `BindHandler` はメソッド + パスを固定する）。対象外は `main` / `init` と生成物のみ。
- **この写像は設計上、一方向である。** production の subject → テストの向きに走り、テストの**書き忘れ**を検知するために存在する。逆向きについては何も主張しない。subject が関数ではなく契約であるような `TestXxx` — データ・生成物・それらを説明するドキュメントの間の整合を固定するもの — は、構造上 production 側の対応物を持たず、1:1 違反ではない。
- **`t.Skip` は、対象が検証不可能であるために到達できない場合のみ許容する** — 例: 失敗経路が `tb.Fatalf` を呼ぶテストヘルパーは、呼び出し側テストの終了を伴うため直接検証できない。skip の理由文には **なぜ検証不可能なのか** を書く。allowlist は持たず、その理由はコードに残す。
- **「他のテストでカバー済み」は skip の理由にならない。** その skip は対象を別テストの実装に依存させる: カバー元が縮小・削除されても skip は green のままで、カバー元が本当にその分岐を通っているかを機械検証する手段も無く、後から増えた分岐は誰にも検証されないまま無言で通る — 1:1 対応が名前だけの殻になる。呼び出し元 / 統合 / DI グラフテストがたまたま通っていても、テスト可能な対象はテストする。他テストを名指しした skip 理由は `internal/architest`（`TestSkipReasonDoesNotNameCoveringTest`）が失敗させる。
- すべての論理分岐を網羅する。
- **`TestXxx` が `t.Run` サブケースを使う場合、その最外グループは literal な `正常系` / `異常系`** — `正常系_xxx` の prefix 形も、`TestXxx` 直下に振る舞いの文を置く形も使わない。その内側にさらに `t.Run` サブケースをネストする。`境界ケース` は第 3 のグループではなく **観点**(§10)であり、境界のケースは各側が生む結果に応じて `正常系` / `異常系` へ振り分ける。
- **単一シナリオの `TestXxx` はグループを省略してよい。** DI グラフの妥当性検証(`fx.ValidateApp`)・配線そのものが契約である分岐なし provider・リポジトリ全体を走査する contract test には分割する先が無く、`正常系` で包んでも分類の情報は増えず body が一段深くなるだけである。この例外は **subject 単位**であり層単位ではない — エラー分岐を持つ subject をこの形に押し込むことは §10 の意味網羅違反になる。
- **`正常系` / `異常系` の振り分けは、subject 自身が失敗するかで決める** — 入力が失敗の題材かどうかでは決めない。`OnStart` の失敗イベントを記録する logger はエラーを返さないため、そのケースはすべて `正常系` に属する。
- グループ構造は `internal/architest`（`TestSubtestGroupPolicy`）が機械的に強制する。どちら側に属すべきかは機械検証できないため、§10 とレビューが担う。

```go
func TestNewUser(t *testing.T) {
    t.Parallel()
    t.Run("正常系", func(t *testing.T) {
        t.Parallel()
        t.Run("全ての入力が正しい場合、エンティティが生成される", func(t *testing.T) { /* ... */ })
    })
    t.Run("異常系", func(t *testing.T) {
        t.Parallel()
        t.Run("IDがゼロ値の場合、エラーを返す", func(t *testing.T) { /* ... */ })
    })
}
```

## 2. 命名

- テストケース名はすべて **日本語**で、振る舞い **と** 分岐条件を記述する。
- グループがある場合、それは素の literal `正常系` / `異常系`。**その内側のサブケース名に `正常系_` / `異常系_` の prefix は付けない** — 振る舞いの文として書く(例: `firstNameの文字数が最小値未満の場合、エラーを返す`)。

## 3. 並列化

- **すべてのネスト階層**で `t.Parallel()` を呼ぶ。
- 例外は共有可変状態のケースと env / CWD の変更(`t.Setenv`・`config.EnsureRepoRootAndEnv`)で、これらは `t.Parallel()` と非互換。`//nolint:paralleltest` と一行の理由を付ける。並列パッケージで衝突する固定 listen ポートは、テストを直列化するのではなく ephemeral アドレス(`127.0.0.1:0`)を優先する。

## 4. table-driven な `for` ループ禁止

**逐次の `t.Run` 兄弟をケースごとに 1 つずつ**書く — `for _, tc := range cases` ループは使わない。各ケースが独立した名前付き `t.Run` を持つことで、失敗時に該当シナリオが名指しされ、並列もケース単位になる。

## 5. アサーション

- **前提条件**・致命チェック・**全てのエラーアサーション**(`NoError` / `Error` / `ErrorIs` / `ErrorContains`)には `require` を使う。`testifylint` の `require-error` ルールが強制し、`assert.ErrorIs` 等は lint で落ちる。
- **後続コードを保護する**チェック(例: dereference 前の `require.NotNil`)にも `require` を使う。
- 後続を保護しない**終端の値検証**(`Equal` / `Len` / `Contains` / `True` / `False` / `Empty`)には `assert` を使い、1 回の実行で不一致を一度に洗い出す。

```go
require.NoError(t, err)            // 前提（失敗で以降無意味）
require.ErrorIs(t, err, ErrX)      // エラー系は require（testifylint require-error）
assert.Equal(t, expected, actual)  // 終端の値検証は assert
```

## 6. mock と生成物

- `*/mock/` 配下の **生成 mock**(`go.uber.org/mock`)を使う。テストファイルに手書き mock を作らない。
- 生成物は編集しない: `**/*.gen.go`・`*.sql.go`・`*_mock.go`。
- テストは公開インターフェースと生成物のみに依存する。

## 7. テストにおけるアーキテクチャ制約

テストも本番コードと同じ onion 境界を守る:

- domain テストは infrastructure 実装を使わない。
- usecase テストは domain の repository を mock する。
- controller テストは usecase を mock する。
- レイヤをバイパスしない。

## 8. カバレッジ

- `make test`(カバレッジ付き)を実行する。カバレッジは現行 baseline から **低下させない**。新規 / 変更パッケージは **90%** 超、handler は ~100% に近づける。
- ラインを下回るパッケージは不足分岐テストを追加する — 満たすまで止めない。(「完了」の定義は [`rules.md` → Testing & Definition of Done](rules.ja.md)。)

## 9. カバレッジ例外とガバナンス

一部の未被覆行は正当であり、contrived テストで **追わない**:

- **構造上到達不能** — あり得ない `switch default`、失敗し得ない前提を守る `panic`、網羅ループ後のコンパイラ必須 `return`。
- **失敗しない防御分岐** — 実質失敗しない演算のエラー return(例: `[]string` の `json.Marshal`)。
- **write-once インフラ(超法規的措置)** — 一度実装するとほぼ触らない `internal/observability` のようなパッケージ。その防御分岐は、**被覆に追加の本番コード・署名変更・runtime スタック操作を要する場合に限り** 未被覆のまま許容する(= 現状のまま到達可能な分岐のみをテストする)。

例外の規則:

- これらの行を塗るためだけの contrived テストや追加実装は **行わない**。
- 各例外は所有パッケージの `README` に記録する(具体的なファイル / 関数の一覧)。
- **ガバナンス:** 新規例外は **任意に追加しない** — アーキテクト等の適切な承認者の承認を得た場合に限り記録する。免除された関数が後に(エラー配線ではない)実際の分岐ロジックを持つようになった場合は、他と同様にユニットテストで担保する。

## 10. 意味的品質のバー(アンチパターン)

section 1〜9 はテストを *well-formed*(整形式)にするが、*meaningful*(有意味)にはしない。テストは構造上 100% 準拠でカバレッジを上げても、被覆した分岐の何が固有なのかを一切 assert していないことがある。この節は全テストが満たすべき意味的なバーと、避けるべきアンチパターンを定める。ここが **テスト品質の唯一の出所**であり、`scaffold-test`(これを満たすテストを生成する)と `test-review`(これに違反するテストを検出する)の双方が読む — 一覧をここに集約することで生成器とレビュアのドリフトを防ぐ。

### 意味網羅 (meaning coverage)

分岐に到達するだけでは不十分 — 各ケースはその分岐の **固有の** outcome を assert しなければならない。単に実行されたことだけを示す「被覆済みだが意味未検証」の分岐は、pass してカバレッジを上げても何も明らかにしない。

- **エラー分岐** — `require.Error` ではなく `require.ErrorIs` で固有の sentinel を assert する。
- **成功分岐** — `require.NoError` / `assert.NotNil` ではなく、他分岐と区別できる結果の値 / 状態を assert する。
- **状態変更メソッド** — 呼び出しが返ったことではなく、変更後のフィールド / 不変条件を assert する。
- **不変な pointer / 参照 getter** — 値の一致ではなく、不変性(返り値を変更してもエンティティに影響しないこと)を assert する。
- **境界** — 受理側だけでなく、境界の両側で異なる outcome(受理 vs 拒否)を assert する。

### アンチパターン(書くときは避け、レビューでは検出する)

- **弱いアサーション** — 複雑な返り値に対し `assert.NotNil` だけ / 後続の状態 assert が無い `assert.NoError` / 要素を assert せず `assert.Equal(t, len(actual), 1)`。*例外:* 1-`TestXxx`-per-subject 規則(section 1)のために単独の `TestXxx` として保持している trivial constructor は *その場で強化*するケース — より強い assert を推奨するのであって、専用テストを削除したり他 subject のテストへ畳み込んだりしない。
- **名前がアサーションを過剰約束** — `t.Run` ケース名が本体で検証していない性質を主張している(例: 本体は `NotNil` だけなのに `"…を保持した収集器を返す"`)。性質を assert するか、ケース名を実際の検証内容に改名する。*系:* 分岐の無い pass-through / 配線関数(例: 入力を constructor に転送するだけの DI provider)は正直な 1 ケースで足りる。入力を変えて同じ `NotNil` を再実行する追加ケースはカバレッジを増やさず、まとめるべき。
- **内部への脆い結合** — 公開 API で足りるのに unexported フィールドを読む / `errors.Is` ではなくログ出力やエラーメッセージ *文字列* を assert する。
- **over-mocking** — 実(純粋)実装の方がより多くを露わにできる協調オブジェクトを mock する / 実装を固定してしまう粒度の呼び出し回数マッチャ。
- **time リテラル固定漏れ** — 固定 `baseTime` ではなくアサーション内で `time.Now()` を呼ぶ / システムクロックに依存する比較。
- **責務クリープ** — 1 つの `TestXxx` が複数 subject を駆動する(section 1 の 1:1 違反)。subject ごとに 1 つの `TestXxx` へ分解し、複数 subject を 1 テストに畳み込まない。
- **helper 重複** — 3 つ以上の `TestXxx` にまたがり 5 行以上の fixture が重複しており、`t.Helper()` 付き helper にすべき。
- **冗長なコメント** — コードを言い換える / *why* を語る inline コメント。ケースの意図はコメントではなく日本語の `t.Run` 名に持たせる([`rules.md`](rules.ja.md) の Comment Rules 準拠)。
