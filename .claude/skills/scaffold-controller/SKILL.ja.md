> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# Scaffold Controller

1 feature の controller (HTTP handler) 層を生成するスキル。**lean A: spec ファイルなし** — handler は OpenAPI gen + usecase Interface から命名規約経由で導出。

## 使うとき

- OpenAPI YAML 書き込み済み + `make gen-api` で handler interface 生成済み
- 対象 usecase 実装済み（mock available）
- `scaffold-endpoint` の 4 番目（最後）の step として自動 chain
- controller 層のみ scaffold する standalone 利用

以下の用途には使いません:

- OpenAPI YAML 変更（人間が書く）
- 既存 handler パッケージの変更（skill は fresh package を仮定）
- 既存 handler に 1 endpoint 追加 — 手で edit
- 非 HTTP controller 生成（job CLI 等）

## 読み書き範囲

**読み込み（常時）**:

- `internal/controller/handler/<path>/gen/server.gen.go` — 生成 `ServerInterface`（operationId 一覧 + 署名の唯一の真実）
- `internal/usecase/<package>/<package>_usecase.go` — 対象 usecase Interface（利用可能 method）
- `internal/usecase/README.md` — mapping 導出用の命名規約（動詞接頭辞、Usecase interface 命名）
- `internal/controller/README.md` — layer 規約
- `internal/controller/handler/README.md` — handler subdir 規約
- `internal/controller/handler/<sibling>/<sibling>_handler.go` — 構造 template
- DI モジュールファイル（通常 `internal/di/server/extension/inbound/`）

**書き込み（承認後）**:

- `internal/controller/handler/<path>/<feature>_handler.go` — handler 実装
- `internal/controller/handler/<path>/<feature>_handler_test.go` — Echo ベーステスト
- `internal/di/module/controller.go`: 既存 `fx.Invoke(...)` 内に `<pkg>.BindHandler` 1 行追加

**Triggers (via `make`)**:

- `make fix` + `make test` — 最終検証

**触らない**:

- OpenAPI YAML（read-only）
- 生成 `server.gen.go` / `type.gen.go`
- usecase / domain / infra 層

## 前提条件

skill が書き込み前に検証:

1. `internal/controller/handler/<path>/gen/server.gen.go` 存在（OpenAPI gen 済み）
2. 対象 usecase パッケージが `internal/usecase/<package>/` に Interface + 生成 mock 込みで存在
3. `internal/controller/handler/<path>/<feature>_handler.go` **未存在**（あれば中断）

前提未充足時は明示メッセージで中断（`make gen-api`、`/scaffold-usecase`、手動 cleanup 等の案内付き）。

## 最初のステップ: identity 確認

`AskUserQuestion` を起動直後に必ず呼ぶ（`scaffold-endpoint` から呼ばれて context に feature + handler path + usecase package がある場合は除く）:

1. 質問: 「対象 feature 名 (kebab-case)」 — フリーテキスト
2. 質問: 「handler package パス (`internal/controller/handler/` 配下)」 — フリーテキスト、ディレクトリ存在検証
3. 質問: 「対応する usecase package 名」 — フリーテキスト、`internal/usecase/<name>/` 存在検証

## Step 1. 入力読み込み

1. `internal/controller/handler/<path>/gen/server.gen.go` を読んで `ServerInterface` メソッド一覧抽出（各メソッドの signature + request 型 + response 型）
2. `internal/usecase/<package>/<package>_usecase.go` を読んで `Usecase` Interface メソッド一覧抽出（signature）
3. `internal/usecase/README.md` と（必要なら）1〜2 個の sibling usecase パッケージから命名規約（このコードベースで使われている動詞接頭辞）を取得
4. `internal/controller/README.md` + `internal/controller/handler/README.md` から layer 規約取得
5. 1 個の sibling handler（`internal/controller/handler/v1/users/v1_users_handler.go` 等）を構造 template として参照

## Step 2. mapping 導出（lean A の核）

各 `ServerInterface` operationId について、対応する usecase method を name match heuristic で探す:

- exact match: `operationId == usecaseMethod`（camelCase） → ✓ mapped
- 動詞認識 match: `GetUsers` op vs `ListUsers` / `FindUsers` usecase method → 規約が pair を認識すれば ✓ mapped
- 単複バリアント: `GetUser` ↔ `GetUserByID` → 部分一致で ✓ mapped

mapping 不能な operationId について:

- `scaffold-controller can not safely proceed` で **中断**、hand-off メッセージ:

  ```text
  以下の operationId は usecase method にマップできません:
    - DeleteUser → usecase に対応する Delete* メソッドなし

  解決方法:
    1. usecase Interface に対応 method を追加（推奨命名: DeleteUser）
    2. OpenAPI から operationId を削除
    3. handler を手で書く（scaffold 対象外）

  解決後に再度 scaffold-controller を実行してください。
  ```

- AI が自動解決しない、skip しない、compile fail する handler stub を生成しない

Step 3 前に mapping レポート（matched count / unmapped count）を 1 件 surface。

## Step 3. test 観点 subagent

Agent tool を起動して controller 層 test 観点を実装前に列挙:

- `subagent_type: general-purpose`
- prompt（日本語）: 導出した mapping + `internal/controller/README.md` test guidance + 期待される controller 観点:
  - HTTP request / response shape (status code, body, headers)
  - validation (OpenAPI schema-level + handler-side extras)
  - usecase mocked, handler に業務ロジックなし
  - apperror → HTTP status mapping (errorhandler middleware カバー確認)
  - middleware 連携（endpoint 固有 middleware が適用される場合）
  - auth context 取り扱い（auth-required operation）
- 出力: operation ごとの test case list（happy path + error paths + auth paths）

## Step 4. 計画と承認

日本語サマリ表示:

- 作成ファイル + DI モジュール更新対象
- 各 operationId: 生成 interface 由来の signature + 対応 usecase method + auth/middleware notes（sibling から推論）
- subagent 由来の test method list

質問:

- 「以下の構成で controller 層を生成しますか？」
- 選択肢: 「生成する」 / 「修正したい箇所を指摘する」 / 「キャンセル」

## Step 5. ファイル書き込み

順序:

1. `<feature>_handler.go` — handler struct + constructor + operation ごとの method
2. `<feature>_handler_test.go` — `testkit/testecho` + `testkit/testassert` パターン
3. `internal/di/module/controller.go` — 既存 `fx.Invoke(...)` 内に `<pkg>.BindHandler` 追加

実装ファイル規約（`internal/controller/handler/README.md` の reference snippet 準拠 — README が canonical）:

- `package <handler-package>`（lowercase）
- `type server struct { tracer observability.LayerTracer; <usecase-deps...> }`（struct 名は `server`、README 規約）
- **Constructor 名**: `BindHandler`（`New` ではない）:

  ```go
  func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc <pkg>.Usecase) {
      gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
          tracer: tf.Controller(),
          uc:     uc,
      }, nil))
  }
  ```

- 各 method:
  - 生成 `ServerInterface` signature と完全一致
  - 冒頭で `ctx, endSpan := s.tracer.Start(ctx); defer endSpan()`（tracer span）
  - 生成 request 型から request 抽出
  - Step 2 で resolve した usecase method 呼び出し
  - DTO → 生成 response 型変換（README reference snippet に従う 1:1 フィールド map、複雑な変換は TODO 化）
  - エラーは直接 return（errorhandler middleware が apperror 翻訳）
- auth-required operation は `ctxhelper.GetAuthn(ctx)` 利用（README 準拠）
- handler に業務ロジックなし

テストファイル規約:

- `testkit/testecho` + `testkit/testassert` 利用
- usecase は生成 mock package で mock
- 日本語 subtest 名

## Step 6. 検証

```sh
make fix
make test
```

handler package coverage 確認。handler test は 100% target（project 規約）。失敗時は surface + TODO + FB。

## Step 7. クロージング

```text
<Feature> controller 層を生成しました。<N> ファイル作成 + DI 1 行追加、make test OK、coverage <X>%。
全層が揃いました — feature の動作確認は `make serve` + curl 等で実機テストできます。
```

commit しない。

## AI 修正スコープ

「Exception: Skill Execution」clause により:

- 書き込み scope: `internal/controller/handler/<path>/<feature>_handler.go` + `_test.go` + DI 1 行追加
- handler ファイル既存時は中断

触らない:

- OpenAPI YAML
- 生成 `server.gen.go` / `type.gen.go`
- usecase / domain / infra 層

## 制約事項

- ❌ handler に業務ロジック含める（usecase or domain の責務）
- ❌ unmapped operationId に handler stub 生成（hand-off で中断）
- ❌ mapping gap を自動解決（dummy usecase method 作成等）
- ❌ OpenAPI YAML、生成ファイル、usecase コード変更
- ❌ test 観点 subagent をスキップ
- ❌ identity 確認 `AskUserQuestion` をスキップ
- ❌ 既存 handler ファイル上書き
- ❌ errorhandler middleware が既にカバーするエラー mapping を custom 追加
- ❌ 失敗時 auto-rollback（TODO + FB）
- ✅ ユーザー向け出力 + テストケース名は日本語
- ✅ 既存 handler を構造 template として利用
- ✅ 生成 `ServerInterface` と完全一致（signature、request/response 型）
- ✅ 同じ skill 実行内で DI 登録更新
- ✅ mapping 導出は命名規約 + sibling pattern のみ（AI が意図を推測しない）

## チェックリスト

- [ ] identity 確認（feature + handler path + usecase package）
- [ ] 前提検証（gen 存在、usecase 存在、handler ファイル未存在）
- [ ] OpenAPI gen `ServerInterface` 読み込み + usecase Interface 読み込み
- [ ] `internal/usecase/README.md` + sibling pkg から命名規約取得
- [ ] mapping 導出完了; **unmapped operationId あれば hand-off で中断** (Step 2)
- [ ] test 観点 subagent 起動、書き込み前に観点キャプチャ
- [ ] 計画を `AskUserQuestion` で確認
- [ ] 実装ファイル書き込み; method signature が生成 interface と完全一致
- [ ] テストファイル書き込み (testkit/testecho + testkit/testassert)
- [ ] `internal/di/module/controller.go` 更新（新 `BindHandler` を `fx.Invoke(...)` 内に追加）
- [ ] `make fix` + `make test` 実行; coverage 報告（or 失敗を TODO + FB surface）
- [ ] commit / push なし
- [ ] 最終サマリ日本語
