> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# Arch Check — Controller

`internal/controller/**` を controller 層アーキ規則 + **lean A 規約**で監査するスキル（handler body は pure template であるべき、controller 層は spec 駆動ではなく OpenAPI gen から導出される）。

## 使うとき

- `internal/controller/` 変更を commit 直前に layer + lean A チェック
- 新規 handler PR の pure-template 規約遵守確認
- 単独 or `arch-check` 統合 chain

## source of truth（毎回読む）

| ソース | 用途 |
| --- | --- |
| `CLAUDE.md` (Layer Rules / Forbidden Shortcuts) | 上位制約 |
| `internal/controller/README.md` | controller 責務 |
| `internal/controller/handler/README.md` | handler 固有規約（pure template） |
| `internal/controller/handler/<path>/gen/server.gen.go` | OpenAPI 生成 `ServerInterface` メソッド一覧（operationId ↔ handler チェック用） |
| `.golangci.yaml` `depguard:` | 既に強制済み |

## lean A 規約（本 skill が強制）

| 規約 | 理由 |
| --- | --- |
| handler メソッド名 = OpenAPI operationId（camelCase 一致） | scaffold-controller が OpenAPI gen から導出するため |
| handler body = pure template (Bind → usecase 呼び出し → response 変換) | 業務ロジックは usecase 側、handler は HTTP I/O 変換のみ |
| 関数長 ~30 行以内（heuristic） | template 通りなら自然に収まる |
| Repository / infra package import 禁止 | layer 境界違反 |

**実装コードに arch-check 専用 annotation を導入しない方針**: pure-template チェックは body 構造から一律実施し、不確実なケースは `suggestion` 止まりにする。escape hatch が必要なら Go 標準慣習（`//nolint:<linter>` 等）に揃える検討は将来余地、独自 prefix の annotation は導入しない。

## 最初のステップ: スコープ確認 + TODO opt

単独実行時 `AskUserQuestion` を 2 問:

1. 質問: 「controller 層のどのスコープで検査しますか？」 / 選択肢: 「変更ファイルのみ」 / 「internal/controller/ 全体」 / 「キャンセル」
2. 質問: 「suggestion 検出箇所に TODO hand-off コメントを追加しますか？」 / 選択肢: 「追加する（既定）」 / 「追加しない（read-only）」

`arch-check` 統合から chain 時は scope + TODO opt 提供済みのためスキップ。

## Step 1. ファイルスコープ解決

```sh
git diff --name-only "origin/${BASE}...HEAD" -- 'internal/controller/**/*.go' | grep -vE '\.gen\.go$|_mock\.go$|_test\.go$'
git ls-files 'internal/controller/**/*.go' | grep -vE '\.gen\.go$|_mock\.go$|_test\.go$'
```

空 → exit。

## Step 2. lint baseline（controller スコープ）

```sh
make lint 2>&1 | tee /tmp/arch-check-controller-lint.out
```

`internal/controller/` パスに絞り込み。

## Step 3. semantic check（lean A 強制）

各 handler ファイル（`*_handler.go`）について:

1. **import** を `internal/controller/README.md` で照合:
   - `internal/infrastructure/**` import → violation（usecase IF 依存にすべき、infra impl 直接 NG）
   - `internal/domain/<aggregate>/<aggregate>_repository.go` 直接 import → violation（usecase 経由）
   - DB パッケージ → violation
2. **operationId ↔ handler メソッドチェック**:
   - sibling `gen/server.gen.go` を探し `ServerInterface` メソッド名抽出
   - handler ファイルの各メソッドが `operationId`（camelCase）と一致確認
   - 余剰（handler メソッドが `ServerInterface` に無い） / 欠落（operation に handler 無い）を `violation` 報告
3. **pure template heuristic**（一律適用、annotation escape なし）:
   - 関数 body は: request 型へ Bind → 1 つの usecase メソッド呼び出し → DTO を response 型変換 → return
   - 検出する anti-pattern（基本的に全 `suggestion`、hard rule は除く）:
     - 複数 `usecase.<X>` 呼び出し → 「orchestration は usecase 側に集約推奨」
     - 単純な `if err != nil` を超える条件分岐 → 確認推奨
     - domain entity の直接構築 → 確認推奨
     - `<handler_pkg>` / generated gen / errorhandler 以外の package 呼び出し → 確認推奨
   - 関数長 30 行超 → `suggestion`

   Hard rule（`violation`）: 禁止 import（Repository / infra） — step 1 でカバー済み。
4. **middleware / error mapping override**: handler ファイルがカスタム middleware / error mapping を定義していたら `suggestion`「endpoint 固有の特殊扱いは README に意図記載を」

## Step 4. TODO hand-off コメント挿入 (opt)

scope 確認時に「TODO 追加」を opt-in した場合（既定 ON）、Step 3 の `suggestion` レベル findings 各々について:

1. ソース内の逸脱位置を特定（handler メソッド行）
2. 逸脱位置の直上 3 行以内に既存コメントブロックがあれば **skip**
3. なければ `// TODO:` コメントを handler メソッド直前に挿入。何が検出されたかと人間向け解決選択肢を記述。標準 `// TODO:` 接頭辞のみ（AI 識別子なし）

コメントは **人間に判断を委ねる hand-off baton** であって AI の判定ではない。AI は「multi-usecase orchestration が意図的か refactor すべきか」を決めず、人間が解決する。

例:

```go
// TODO: handler が複数 usecase（ListUsers, CountUsers）を呼び出している。
// orchestration を usecase 側に集約するか、本コメントを WHY 説明に置き換えてください。
func (h *Handler) GetUsers(...) {
```

`violation` レベル（禁止 import、operationId mismatch）には TODO を付けない — 修正一択。

## Step 5. レポート（日本語）

```text
arch-check-controller 結果（スコープ: <scope>）

[lint] N 件
  - <file:line>: <linter>: <message>

[operationId ↔ handler method] K 件
  internal/controller/handler/v1/users/v1_users_handler.go
    violation: ServerInterface に PutUsers あるが handler メソッド未実装
    source: internal/controller/handler/v1/users/gen/server.gen.go

[pure template] M 件
  internal/controller/handler/v1/orders/orders_handler.go:67
    suggestion: 関数 52 行 + 2 usecase 呼び出し
    source: internal/controller/handler/README.md "handler は pure template"
    remediation: orchestration を usecase 側に集約推奨

総計: violations <N+K+M>, suggestions <L>
TODO hand-off: 追加 <P> 件, スキップ <Q> 件（既存コメント）
```

空:

```text
arch-check-controller 結果（スコープ: <scope>）
controller 層の違反は検出されませんでした（lean A 規約遵守）。
```

## Step 6. クロージング

- 単独: exit 0
- chain: 違反数 + TODO 追加数返却
- 自動修正なし（TODO hand-off コメントのみ追加）

## AI 修正スコープ

**ほぼ read-only**、ただし 1 つだけ narrow な write scope あり:

- 読み込み: `CLAUDE.md` / READMEs / OpenAPI gen / in-scope Go ファイル。`make lint` 実行
- 書き込み（scope 確認時の user opt 時のみ）: `internal/controller/**/*.go` の handler メソッド逸脱位置への `// TODO:` hand-off コメント追加。**TODO 追加のみ**
- 逸脱位置の直上 3 行以内に既存コメントブロックがあれば skip
- user が「TODO 追加なし」を opt した場合は完全 read-only

## 制約事項

- ❌ controller ルールをハードコード（必ず README から読む）
- ❌ 実装コードに arch-check 専用 annotation を導入（pure-template 一律適用、不確実は suggestion 止まり）
- ❌ ソースコード変更（逸脱位置への TODO hand-off コメントのみ user opt 時に許可）
- ❌ 単独実行時の scope 確認をスキップ
- ❌ `violation` レベル（禁止 import / operationId mismatch）に TODO 付与 — 修正一択
- ❌ multi-usecase / 非 template body が意図的か AI が判定（TODO で人間に hand-off）
- ❌ TODO に AI 識別 prefix（`// TODO:` のみ）
- ❌ 既存コメントを上書き
- ✅ 日本語出力
- ✅ source-of-truth 引用 + 行
- ✅ lean A 規約強制（operationId ↔ method、pure template、import 禁止）
- ✅ suggestion 検出位置に TODO hand-off コメント追加（opt-in 既定 ON）、既存コメントあれば skip
- ✅ README + gen を毎回再読

## チェックリスト

- [ ] scope + TODO opt 確認 or 受領
- [ ] CLAUDE.md / controller README / handler README / gen/server.gen.go を読み込み
- [ ] `make lint` controller 絞り込み
- [ ] operationId ↔ handler method 1:1 チェック
- [ ] pure-template heuristic（一律適用、不確実は suggestion）
- [ ] import 禁止リストチェック
- [ ] findings に source-of-truth 引用
- [ ] レポート日本語
- [ ] suggestion 検出位置に TODO hand-off コメント挿入（opt-in 時）、既存コメントあれば skip
- [ ] TODO 追加以外のコード変更なし
