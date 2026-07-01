# テスト規約

本書はこのリポジトリで **テストをどう書くか** の唯一の source of truth です — 構造・命名・
並列化・アサーション・mock・カバレッジ例外のガバナンス。`scaffold-test`(生成)と
`test-review`(レビュー)の各 skill が runtime で本書を読むため、ここに一元化して drift を防ぎます。

スコープ分担(以下を跨いで重複させないこと):

- **本書** — 具体的な *どう書くか*(技法 / 規約)。
- [`rules.md` → *Testing & Definition of Done*](rules.md) — 非交渉の *どうなれば完了か*(層ごとのテスト・90% ライン・「compiles ≠ done」・実行時 DI 検証・実アプリの smoke test・到達不能分岐の方針)。
- 各層 `README` → *Test Strategy* — 層ごとの **観点**(その層で何を検証するか)。

canonical な参照テストは [`internal/domain/user/user_domain_test.go`](../../internal/domain/user/user_domain_test.go) です。

## 1. 構造

- **1 つの関数 / メソッドにつき 1 つの `TestXxx`**。複数対象を 1 つのテスト関数に束ねる場合は、都度の明示的な正当化が必要。
- すべての論理分岐を網羅する。
- **最外の `t.Run` グループは literal な `正常系` / `異常系`** — `正常系_xxx` の prefix 形は使わない。その内側にさらに `t.Run` サブケースをネストする。

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
- 外側グループは素の literal `正常系` / `異常系`。**その内側のサブケース名に `正常系_` / `異常系_` の prefix は付けない** — 振る舞いの文として書く(例: `firstNameの文字数が最小値未満の場合、エラーを返す`)。

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
- ラインを下回るパッケージは不足分岐テストを追加する — 満たすまで止めない。(「完了」の定義は [`rules.md` → Testing & Definition of Done](rules.md)。)

## 9. カバレッジ例外とガバナンス

一部の未被覆行は正当であり、contrived テストで **追わない**:

- **構造上到達不能** — あり得ない `switch default`、失敗し得ない前提を守る `panic`、網羅ループ後のコンパイラ必須 `return`。
- **失敗しない防御分岐** — 実質失敗しない演算のエラー return(例: `[]string` の `json.Marshal`)。
- **write-once インフラ(超法規的措置)** — 一度実装するとほぼ触らない `internal/observability` のようなパッケージ。その防御分岐は、**被覆に追加の本番コード・署名変更・runtime スタック操作を要する場合に限り** 未被覆のまま許容する(= 現状のまま到達可能な分岐のみをテストする)。

例外の規則:

- これらの行を塗るためだけの contrived テストや追加実装は **行わない**。
- 各例外は所有パッケージの `README` に記録する(具体的なファイル / 関数の一覧)。
- **ガバナンス:** 新規例外は **任意に追加しない** — アーキテクト等の適切な承認者の承認を得た場合に限り記録する。免除された関数が後に(エラー配線ではない)実際の分岐ロジックを持つようになった場合は、他と同様にユニットテストで担保する。
