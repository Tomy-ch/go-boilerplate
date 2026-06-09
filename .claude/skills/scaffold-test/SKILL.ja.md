> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# Scaffold Test

既存の関数 / メソッドに対する Go ユニットテストファイルを生成するスキル。書き方は `internal/domain/user/user_domain_test.go` から抽象化したパターン（全階層 `t.Parallel()` ＋ ネスト `t.Run` ＋ 日本語ケース名 ＋ 最外殻 `正常系` / `異常系` グループ、table-driven の `for` ループは原則禁止）に従う。

## 使うとき

- 既に実装が存在しコンパイルが通っている関数 / メソッドに対し、ユニットテストを追加するとき。
- `make test` のカバレッジが 90 % を下回ったパッケージを埋めるとき。
- 手動編集で挙動が変わった関数について、既存テストが意図を反映しなくなったとき。
- `scaffold-domain` / `scaffold-usecase` / `scaffold-controller` / `scaffold-infra-db` から chain される一段として（親が target + 観点を渡すので、First Step と Step 2 はスキップ）。

使わない場面:

- **HTTP 境界の統合テスト**（`internal/integration/` 配下）→ `scaffold-integration-test` を使う。本スキルは同一パッケージ内ユニットテスト専用。
- 既存テストファイル全体の書き直し → 手で編集する。
- 実装コードの生成 → 本スキルは test ファイルしか書かない。
- 生成ファイル (`*.gen.go` / `*_mock.go` / `*.sql.go`) に対するテスト。

## 読む / 書く

**常に読む**:

- `CLAUDE.md` の Testing Instructions（parallel 必須・命名・require vs assert・mock 方針・層構造ルール）。
- 対象ソースファイル（シグネチャ・引数・戻り値・エラーセンチネル・package 内ヘルパを抽出）。
- ファイルパスから自動解決する層別 README:
  - `internal/domain/README.md` （`internal/domain/**`）
  - `internal/usecase/README.md`（＋ `internal/usecase/boundary/README.md`）（`internal/usecase/**`）
  - `internal/controller/README.md`（＋ `internal/controller/handler/README.md`）（`internal/controller/handler/**`）
  - `internal/infrastructure/README.md`（＋ `internal/infrastructure/rdb/README.md`）（`internal/infrastructure/**`）
  - `pkg/README.md`（＋ 最寄りの `pkg/<name>/README.md`）（`pkg/**`）
- 同一パッケージ内の他 test ファイル（import 構成、ヘルパスタイル `newValidUser(t)` 等、assertion 文体、fixture 慣例）。README と矛盾するときは **README 優先**。
- `<package>/mock/*_mock.go` （対象が DI 注入インタフェースを使う場合のみ）。

**書く（承認後のみ）**:

- 対象と同じ階層に 1 つの test ファイル `<subject>_test.go`（`<subject>` は元ファイルから拡張子を除いた basename）。既存ファイルがある場合は **末尾追記** モードに切り替え、事前に承認を取る。

**`make` 経由で発火**:

- `make fix`（生成 test ファイルの自動整形）
- `make test`（生成テストの実行 + パッケージ全体カバレッジが落ちないことの確認）

**触らない**:

- 対象ソースファイル本体（`<subject>.go`）。
- 生成成果物（`**/*.gen.go` / `*_mock.go` / `*.sql.go`）。
- `*/mock/` 配下（読むだけ）。
- 他パッケージの test ファイル。

## First Step: 対象解決

`scaffold-*` から chain されている場合を除いて、最初に `AskUserQuestion`:

- 質問: 「テストを書きたい対象を指定してください」
- 選択肢（single-select）:
  - 「対象ファイル全体」 — free-text path。ファイル内の export 済み top-level 関数 / メソッドを全て列挙し、それぞれに 1 つの `TestXxx` を生成。
  - 「ファイル内の特定関数 / メソッドのみ」 — free-text `<file>:<symbol>`。指定の 1 件に対してのみ `TestXxx` 生成。
  - 「キャンセル」。

解決後、ファイルパスから層キー（domain / usecase / controller / infra / pkg）を判定して以降のステップで使う。

対象ファイルが存在しない場合は中断してパス確認。

## Step 1. 層コンテキストを読む

1. 上述の層別 README を読む。命名・ヘルパスタイル・層固有のテスト規約（domain の `pkg/ptr.Copy` 不変性チェック、controller の `testecho` 利用、infra の `pgerror.NormalizeError` アサーション 等）はここに canonical。
2. 対象パッケージ内の全 `*_test.go` を読む。次を抽出:
   - top-level test ヘルパ（例: `newValidUser(t *testing.T) (*User, time.Time)`）。
   - `TestXxx` 関数冒頭で慣例的に宣言されている fixture 変数。
   - 実際に使われている assertion スタイルと import セット。
3. `CLAUDE.md` の Testing Instructions を 1 回読み、parallel 必須・命名・require vs assert・mock 方針・層構造制約として扱う。

sibling と README が矛盾する場合、**README 優先**（[[feedback-readme-priority]]）。

## Step 2. テスト観点 subagent

コード生成前に必ず subagent (`subagent_type: general-purpose`) を起動して **対象 subject に対する** テスト観点を列挙させる。このステップは必須でメインループにインライン化しない（観点は層 README の Test Strategy 節 + 対象シグネチャから派生させるもので、スキル自身の記憶からパターンマッチさせる対象ではない）。

このスキルは層別の観点シードリストをハードコードしない。観点の SSOT は層 README （canonical）で、本スキルは README が現時点で書いている内容にそのまま従う。README が進化すれば観点も自動的に追従する — スキル側の編集は不要。

プロンプト内容（日本語）:

- 対象 subject のシグネチャ + Doc コメント。
- 該当層 README の `Test Strategy` / `Testing strategy` 節（存在する見出しの方）の **全文**。Step 1 で読み取った内容をサブセクション見出しごとそのまま渡す。subagent の仕事はそれらの見出しを対象 subject に対する具体観点へ落とすこと。本スキル作成時点での READMEs の状況（記述的、固定マップではない）:
  - `internal/domain/README.md` → `## Testing strategy`（Getter contract / Immutable guarantee / Domain behavior / Error classification / Test design policy / Test Fixture / Invariant preservation）
  - `internal/usecase/README.md` → `## Testing Strategy`（Test dependencies / Testing goals / Test targets / Test structure / What not to test）
  - `internal/controller/handler/README.md` → `## Test Strategy`（Test Dependencies / Test Targets / Test Structure / Router Test / Handler Test / Error Test / Thin Controller Test Scope / Observability Test / Test Policy / Not Covered in Controller Tests / Test Kit testkit / testassert / testauth / testecho / testspan）
  - `internal/infrastructure/README.md` + `internal/infrastructure/rdb/README.md` → `## Test Strategy` / `### 7. Test Strategy (Integration-based)`
  - `pkg/README.md` → **意図的に Test Strategy 節を持たない**。`pkg/` は framework-agnostic な pure utility (`CLAUDE.md`) で、観点は標準 Go テストパターン（input-output 検証 / edge / boundary 値 / nil / zero ハンドリング）に帰着し、sibling tests（`pkg/datetime` / `pkg/envutil` / `pkg/ptr` / `pkg/uuid` / `pkg/xerrors` 等の既存テスト）でパターンが明示されている。subagent は sibling tests + `CLAUDE.md` から観点を派生する。ドキュメントの穴ではなく **層として正常**なので、gap 警告は出さない。 package 個別の不変条件は `pkg/<name>/README.md` を併読する。
  この一覧は本スキル作成時点の READMEs を記述したもので、固定マッピングではない — README が更新されれば subagent はその時点の見出しを読んで適応する。見出しが renamed / removed / added されても、subagent は実行時点の README をそのまま使う。
- Step 1 で確認した sibling のテストパターン（補助参照）。
- `CLAUDE.md` Testing Instructions（プロジェクト横断ベースライン）。

期待される戻り値: 生成すべき `TestXxx → t.Run(正常系) → t.Run(case)` / `t.Run(異常系) → t.Run(case)` のパスを構造化したリスト（各 case に対応する README サブセクション or sibling test パターンを添付）。

フォールバック:

- **層が `pkg/**`** — README が意図的に Test Strategy 節を持たない。観点は標準 Go パターン（input-output / edge / nil / zero）に帰着するので、 sibling tests + 該当する `pkg/<name>/README.md`（あれば）+ `CLAUDE.md` から派生し、**警告は出さない**。これが pkg 層の正常モード。
- **層が `internal/<layer>/**` で Test Strategy が期待されているのに無い場合** — gap として user に surface（`「<README path> に Test Strategy 節がないため、sibling テストパターン + CLAUDE.md からフォールバックで観点を導出しています。README を補完する余地があります」`）。 現状 `internal/domain/` / `internal/usecase/` / `internal/controller/handler/` / `internal/infrastructure/` の各 README は Test Strategy 節を持っているので、ここで欠落していたらドキュメント側の補完候補としてユーザに知らせる。
- subagent が観点を返さない場合（層を問わず）、最小デフォルト（正常系成功 1 件 + 異常系全捕捉 1 件）にフォールバックしてユーザに警告。

## Step 3. テスト構造の設計

ハードルール:

1. **1 関数 / メソッド = 1 `TestXxx`**。 `Foo` → `func TestFoo(t *testing.T)`、`(*User).UpdateProfile` → `func TestUser_UpdateProfile(t *testing.T)`。 同一 subject に対する複数 `TestXxx` は絶対に作らない。
2. **複数 subject を 1 `TestXxx` に束ねるのは例外**。 subagent またはユーザが提案した場合（例: 全 getter を `TestEntity_Accessors` で一括検証する）、`AskUserQuestion`:
   - 質問: 「`<funcA>` / `<funcB>` / ... を 1 つの TestXxx にまとめる構成案ですが、原則は 1 関数 = 1 TestXxx です。束ねますか？」
   - 選択肢: 「束ねる（理由を 1 行で）」 / 「別々に作る（推奨）」。
   - 「束ねる」が選ばれた場合、1 行の rationale を取得し、束ねた `TestXxx` の直上に Go コメントとして残す。
3. **最外殻 2 つの `t.Run` は `正常系` / `異常系`**。 両方とも直後に `t.Parallel()` を呼ぶ。さらに細分化するためのネストグループ（例: `t.Run("firstNameが範囲外の場合、エラーを返す", ...)`) は可読性が上がるなら推奨。
4. **全ての `t.Run` の冒頭で `t.Parallel()` を呼ぶ**。例外: sibling ブロックと共有しているポインタを mutate する場合（`TestImmutableAccessors` の `building` / `deletedAt` ブロック等）は外側の `t.Run` を逐次にする。**ブロック直上にコメント必須**（`-race` で検出される競合を意図的に避けている旨を書く）。内部 case は引き続き `t.Parallel()`。
5. **table-driven `for` ループは原則生成しない**。 連続した `t.Run` sibling で書く（`user_domain_test.go` のパターン）。観点上 table 形式の方が明らかに可読性が上がる場合（同一本体で `(input, expected)` の組が長く列挙される 等）、`AskUserQuestion`:
   - 質問: 「`<case>` は table-driven (`for _, tc := range ...`) で書く方が可読性が高そうですが、原則は逐次 `t.Run` です。table 形式にしますか？」
   - 選択肢: 「逐次 `t.Run` で（推奨）」 / 「table-driven で書く」。
6. **ケース名は日本語**。 最外殻は `正常系` / `異常系`。 サブケースは入力クラスと期待結果を 1 文で表す自由記述（`「<入力クラス>の場合、<結果>」`）。 そのまま読める文章になるように。
7. **`require` vs `assert`**（`CLAUDE.md` 準拠）:
   - `require.NoError` / `require.Error` / `require.ErrorIs` / `require.ErrorContains` — エラー系アサーション全般（testifylint `require-error` ルールが `assert.ErrorIs` を拒絶）。
   - `require.Not<Nil>` は以降の dereference をガードする場合のみ。
   - `assert.Equal` / `assert.Len` / `assert.Contains` / `assert.True` / `assert.False` / `assert.Empty` は終端の値検証（失敗時に以降を止めない）。
8. **mock は常に `<package>/mock/` から**（`go.uber.org/mock` + `make gen-api` の生成物）。手書き mock は `CLAUDE.md` で禁止。
9. **DB / HTTP / 外部 IO はユニットテストでは行わない**（domain / usecase / controller 層）。 infra の実 DB 結合は、同パッケージにすでに sibling test がいて慣行が確立されている場合のみ本スキルの守備範囲。それ以外は `scaffold-integration-test` への切替を案内する。

## Step 4. プランの確認

日本語で:

- 対象ファイルパス
- 検出した層
- 生成予定の各 `TestXxx` と、その配下の 正常系 / 異常系 サブケースのリスト
- 各 case を生んだ観点との対応理由
- 提案テストファイル冒頭 ~20 行のプレビュー
- 承認された例外（複数 subject 束ね / table-driven）と保存された rationale

を提示してから `AskUserQuestion`:

- 質問: 「以下の構成でテストを生成しますか？」
- 選択肢: 「生成する」 / 「修正したい箇所を指摘する」 / 「キャンセル」。

## Step 5. test ファイル書き出し

`internal/domain/user/user_domain_test.go` から抽象化した骨格に従う:

```go
func Test<Subject>(t *testing.T) {
    t.Parallel()

    // サブテスト間で共有する fixture をここで宣言。time は固定 baseTime に pin して
    // determinism を確保する。
    <fixture variables>

    t.Run("正常系", func(t *testing.T) {
        t.Parallel()

        t.Run("<日本語の入力クラス>の場合、<期待される結果>", func(t *testing.T) {
            t.Parallel()
            actual, err := Subject(<args>)
            require.NoError(t, err)
            assert.Equal(t, <expected>, actual.<field>)
        })
        // ... 追加の正常系
    })

    t.Run("異常系", func(t *testing.T) {
        t.Parallel()

        t.Run("<日本語の入力クラス>の場合、エラーを返す", func(t *testing.T) {
            t.Parallel()
            actual, err := Subject(<invalid args>)
            assert.Nil(t, actual)
            require.ErrorIs(t, err, <ErrSentinel>)
        })
        // ... 必要なら細分化のためにネスト
    })
}
```

追加の生成ルール:

- **テストヘルパ**は package 内既存のもの（例: `newValidUser(t)`）を再利用。 未定義かつ同じ fixture を 3 回以上繰り返すならば、test ファイル末尾に `t.Helper()` 付きの unexported ヘルパを生成。
- **`uuid.NewTestFromSalt(t, "<salt>")`** がテスト内で deterministic な UUID を取得する canonical 手段（`pkg/uuid`）。
- **`ptr.To(...)` / `ptr.Copy(...)`** は nullable ポインタフィールドで利用（domain README の "Handling time and ID" / "Invariants" 節 参照）。
- **mock セットアップ**: 対象が DI 注入インタフェースを使う場合、対応する `mock.NewMock<Interface>(t)` を生成し、 `.EXPECT().<Method>(...).Return(...)` を対象の呼び出し順序通りに並べる。
- **import**: 同一 package の sibling test と同じ構成に揃える。未使用 import は入れない。

test ファイル既存の場合は全書き換えしない — 末尾追記モード（既存ヘルパ宣言の後ろ）に切り替え、`AskUserQuestion` で事前承認を取る。

## Step 6. 検証

順次:

1. `make fix` — 生成 test ファイルを整形。対象外ファイルが整形されたら diff を提示。
2. `make test` — 生成テスト通過確認 + パッケージのカバレッジが下がっていないこと（`CLAUDE.md` の新規 / 変更パッケージ 90 % 閾値）。

`make test` 失敗時:

- 生成 test ファイルはそのまま残す（user 検査用）。
- 失敗テスト名 + assertion メッセージを提示。
- 自動 rollback はしない。テスト修正か subject 修正かは user 判断。

カバレッジが 90 % 未満に落ちた場合（テスト自体は通る場合も）は、不足観点を提示して再呼び出しを促す。

## Chainability

`scaffold-domain` / `scaffold-usecase` / `scaffold-controller` / `scaffold-infra-db` から chain された場合、親は最低限以下を渡す:

- `target_file` — テスト対象ソースの絶対パス。
- `target_subjects` — 親がテストを欲しい `<func or method name>` のリスト。
- `layer` — 解決済み層キー（`domain` / `usecase` / `controller` / `infra` / `pkg`）。
- `viewpoints` — 親が既に subagent から得たテスト観点（あれば）。

chain モードでは以下をスキップ:

- First Step（target 解決）。
- Step 2（test-perspective subagent）— `viewpoints` が非空のとき。
- Step 4 の `AskUserQuestion`（親が feature 単位の承認を既に取得済み）— 監査用に 1 行サマリは表示する。

ただし**複数 subject 束ね例外** と **table-driven 例外** の `AskUserQuestion` は chain モードでも必須（親が知らないルールなので）。

## 制約（サマリ）

- ❌ 同一関数 / メソッドに対する複数 `TestXxx`。
- ❌ `AskUserQuestion` 承認なしの複数 subject 束ね（および rationale コメント無し）。
- ❌ `AskUserQuestion` 承認なしの table-driven `for` ループ。
- ❌ 手書き mock（`<package>/mock/*_mock.go` を使う）。
- ❌ subject ソースファイルの編集。
- ❌ 生成物の編集（`*.gen.go` / `*_mock.go` / `*.sql.go`）。
- ❌ `assert.ErrorIs` / `assert.NoError`（testifylint `require-error`）。
- ❌ コメント付き `-race` 理由なしでの `t.Parallel()` 省略。
- ❌ サブケースでの `t.Run` 省略。
- ❌ 英語ケース名（`CLAUDE.md` の通り日本語）。
- ✅ 全階層 `t.Parallel()`（明文化された例外のみ可）。
- ✅ サブケース毎の `t.Run`。
- ✅ 日本語ケース名（最外殻 `正常系` / `異常系`）。
- ✅ エラーは `require` / 終端値は `assert`。
- ✅ mock は `*/mock/` 由来のみ。
- ✅ deterministic な fixture（固定 `baseTime`、`uuid.NewTestFromSalt(t, ...)`）。
- ✅ 生成 test ファイル内ヘルパに `t.Helper()`。

## チェックリスト

完了報告前に確認:

- [ ] 対象ファイル + 層が確定（standalone）または親 scaffold-* から受領（chain）した。
- [ ] Step 1 で層 README + `CLAUDE.md` Testing Instructions + sibling test を読んだ。
- [ ] Step 2 の test-perspective subagent を実行した（または親から `viewpoints` を受領）。
- [ ] 各 `TestXxx` が単一 subject に対応している、または `AskUserQuestion` で例外承認 + rationale が記録されている。
- [ ] 最外殻 `t.Run` は `正常系` / `異常系`。
- [ ] 全 `t.Run` の冒頭で `t.Parallel()` を呼んでいる、または `-race` 例外の説明コメントが付いている。
- [ ] `for` ループは生成していない、または `AskUserQuestion` で例外承認済み。
- [ ] エラー系は `require.*`、終端値は `assert.*`。
- [ ] mock は `<package>/mock/` 由来のみ。手書き mock を追加していない。
- [ ] subject ソースは編集していない。
- [ ] `make fix` + `make test` が pass し、カバレッジが落ちていない。
