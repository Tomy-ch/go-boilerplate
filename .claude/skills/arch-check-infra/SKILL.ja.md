> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# Arch Check — Infra

`internal/infrastructure/**` を infrastructure 層アーキ規則 + **lean A 規約**で監査するスキル（Repository body は pure template であるべき、infra 層は spec 駆動ではなく domain IF + sqlc gen から導出される）。

## 使うとき

- `internal/infrastructure/` 変更を commit 直前に layer + lean A チェック
- 新規 Repository 実装 PR レビュー
- 単独 or `arch-check` 統合 chain

## source of truth（毎回読む）

| ソース | 用途 |
| --- | --- |
| `CLAUDE.md` (Layer Rules) | 上位制約 |
| `internal/infrastructure/README.md` | infra 層規則 |
| `internal/infrastructure/rdb/README.md` | RDB 層固有規約 |
| `internal/infrastructure/rdb/pgerror/README.md` | エラー正規化規約 |
| `internal/infrastructure/rdb/sqlc/gen/*.gen.{sql.go,go}` | sqlc 生成関数一覧（`*.gen.sql.go` が query files、`*.gen.go` が db / models）— 両方を Repository ↔ sqlc gen チェックで利用 |
| `internal/domain/<aggregate>/<aggregate>_repository.go` | domain Repository IF（impl カバレッジ確認） |
| `.golangci.yaml` `depguard:` | 既に強制済み |

## lean A 規約（本 skill が強制）

| 規約 | 理由 |
| --- | --- |
| Repository メソッドが 1+ の sqlc gen 関数を呼ぶ（多重 OK、switch dispatch OK、JOIN 用複数 query OK） | scaffold-infra-db は 1:1 mechanical case のみ自動導出、それ以外は hand-write |
| Repository body = データ orchestration のみ（sqlc 呼び出し + pgerror + 行→entity 変換）。**業務ロジック厳禁** | 業務ロジックは domain entity / usecase の責務 |
| `pgerror.NormalizeError` が全 sqlc return で呼ばれる | DB エラーの統一正規化 |
| 全 Repository が対応する domain Repository IF を実装 | layer 境界 |

**実装コードに arch-check 専用 annotation を導入しない方針**: 多重 sqlc 呼び出し / switch dispatch / JOIN は body 構造から自動推論。escape hatch が必要なら Go 標準慣習（`//nolint:<linter>` 等）に揃える検討は将来余地として残すが、独自 prefix の annotation 系は導入しない。実装コードはあくまで実装の主体であり、補助ツールに引きずられない。

## 最初のステップ: スコープ確認 + TODO opt

単独実行時 `AskUserQuestion` を 2 問:

1. 質問: 「infrastructure 層のどのスコープで検査しますか？」 / 選択肢: 「変更ファイルのみ」 / 「internal/infrastructure/ 全体」 / 「キャンセル」
2. 質問: 「suggestion 検出箇所に TODO hand-off コメントを追加しますか？」 / 選択肢: 「追加する（既定）」 / 「追加しない（read-only）」

`arch-check` 統合から chain 時は scope + TODO opt 提供済みのためスキップ。

## Step 1. ファイルスコープ解決

```sh
git diff --name-only "origin/${BASE}...HEAD" -- 'internal/infrastructure/**/*.go' | grep -vE '\.gen\.go$|_mock\.go$|_test\.go$'
git ls-files 'internal/infrastructure/**/*.go' | grep -vE '\.gen\.go$|_mock\.go$|_test\.go$'
```

空 → exit。

## Step 2. lint baseline（infra スコープ）

```sh
make lint 2>&1 | tee /tmp/arch-check-infra-lint.out
```

`internal/infrastructure/` パスに絞り込み。

## Step 3. semantic check（lean A 強制）

各 repository ファイル（`internal/infrastructure/rdb/repository/**/*_repository.go`）について:

1. **import 検査**:
   - `database/sql` 直接 import → suggestion（sqlc gen wrapper を使う）
   - Repository メソッドが存在するのに `internal/infrastructure/rdb/pgerror` import 無し → violation 可能性
   - DB 以外の framework 直接 import → violation
2. **domain IF カバレッジ**:
   - 対応する `internal/domain/<aggregate>/<aggregate>_repository.go` を探す
   - Repository struct が全 IF メソッドを実装しているか確認（compile-time でも強制、skill は IF メソッド名 ↔ struct メソッド名を二重チェック）
3. **Repository メソッド ↔ sqlc gen チェック（soft）**:
   - 各 Repository メソッドが `internal/infrastructure/rdb/sqlc/gen/*.gen.{sql.go,go}` の関数を 1+ 呼び出していることを body 構造から確認
   - **正当な multi-query パターンを自動許容**（annotation 不要）:
     - 複数 sqlc gen 結合（JOIN、N+1 解決、集計+詳細） — body 内の sqlc.* 呼び出し回数で検出
     - パラメータ駆動の switch dispatch（例: `FindByActive`） — switch 構文内の sqlc 呼び出しを検出
   - **`suggestion`（`violation` ではない）として報告**:
     - Repository メソッドが `sqlc/gen/` の関数を呼んでいない → 「sqlc gen 呼び出しが見当たりません。実装見直しを推奨」
     - メソッド名が sqlc gen 関数名と stem マッチしない → 確認推奨

4. **body composition heuristic**:
   - 期待形: tracer span → 1+ sqlc gen 呼び出し → 行→entity 変換 → return（`pgerror.NormalizeError` 経由）
   - **`violation`（hard rule）**:
     - body に業務ロジック検出 — 例: entity invariant 検証、domain レベルの条件分岐、フィールドコピー / nil-coalesce / 型 cast 超えるデータ加工 → 「業務ロジックは domain / usecase の責務」
     - sqlc 呼び出し戻り値 error を `pgerror.NormalizeError` 経由せず raw で返す
   - **`suggestion`**:
     - 関数長 50 行超（switch dispatch 除く） → 複雑度確認
     - tracer span 未配線
5. **observability**:
   - Repository メソッドに `tracer.Start` 無し → `suggestion`（`internal/infrastructure/README.md` observability 節準拠）

query_service / system_query ディレクトリにも同じ規則適用（pure-template + sqlc wrap 規約を共有）。

## Step 4. TODO hand-off コメント挿入 (opt)

scope 確認時に「TODO 追加」を opt-in した場合（既定 ON）、Step 3 の `suggestion` レベル findings 各々（Repository ↔ sqlc gen soft mismatch / 関数長 suggestion / tracer span 欠落）について:

1. ソース内の Repository メソッド行を特定
2. 直上 3 行以内に既存コメントブロックがあれば **skip**
3. なければ `// TODO:` コメントをメソッド直前に挿入。検出内容と人間向け解決選択肢を記述。標準 `// TODO:` 接頭辞のみ

コメントは **人間に判断を委ねる hand-off baton**。AI は「multi-query / dispatch が意図的か」を決めず、人間が解決する。

例:

```go
// TODO: FindUserWithPosts は users / posts の 2 つの sqlc gen を組み合わせている。
// 意図通りなら本コメントを WHY 説明に置き換えてください。
func (r *userRepository) FindUserWithPosts(...) {
```

`violation` レベル（body 業務ロジック / pgerror 経由なし）には TODO を付けない — 修正一択。

## Step 5. レポート（日本語）

```text
arch-check-infra 結果（スコープ: <scope>）

[lint] N 件
  - <file:line>: <linter>: <message>

[Repository ↔ sqlc gen] K 件（suggestion only）
  internal/infrastructure/rdb/repository/user/user_repository.go
    suggestion: Save メソッドが sqlc gen 関数を呼んでいません
    remediation: 対応 query 追加 or 実装の見直し

[body composition] M 件
  internal/infrastructure/rdb/repository/order/order_repository.go:42
    violation: NormalizeError 経由なし、errors.New(...) で直接返している
    source: internal/infrastructure/rdb/pgerror/README.md "全 sqlc エラーは NormalizeError 経由"

  internal/infrastructure/rdb/repository/order/order_repository.go:60
    violation: Repository body に業務ロジック検出（amount > 0 のような entity 不変条件チェック）
    source: internal/infrastructure/README.md "Repository は data orchestration のみ"
    remediation: invariant check を domain entity 側へ

総計: violations <M>, suggestions <K+...>
TODO hand-off: 追加 <P> 件, スキップ <Q> 件（既存コメント）
```

空:

```text
arch-check-infra 結果（スコープ: <scope>）
infrastructure 層の違反は検出されませんでした（lean A 規約遵守）。
```

## Step 6. クロージング

- 単独: exit 0
- chain: 違反数 + TODO 追加数返却
- 自動修正なし（TODO hand-off コメントのみ追加）

## AI 修正スコープ

**ほぼ read-only**、ただし 1 つだけ narrow な write scope あり:

- 読み込み: `CLAUDE.md` / READMEs / sqlc gen / domain Repository IF / in-scope Go ファイル。`make lint` 実行
- 書き込み（scope 確認時の user opt 時のみ）: `internal/infrastructure/**/*.go` の Repository メソッド逸脱位置への `// TODO:` hand-off コメント追加。**TODO 追加のみ**
- 逸脱位置の直上 3 行以内に既存コメントブロックがあれば skip
- user が「TODO 追加なし」を opt した場合は完全 read-only

## 制約事項

- ❌ infra ルールをハードコード
- ❌ sqlc gen 1:1 不一致を hard violation 扱い（suggestion のみ）
- ❌ 実装コードに arch-check 専用 annotation を導入（multi-query / dispatch は body 構造から自動推論）
- ❌ ソースコード変更（逸脱位置への TODO hand-off コメントのみ user opt 時に許可）
- ❌ 単独実行時の scope 確認をスキップ
- ❌ `violation` レベル（業務ロジック / pgerror 経由なし）に TODO 付与 — 修正一択
- ❌ multi-query / dispatch が意図的か AI が判定（TODO で人間に hand-off）
- ❌ TODO に AI 識別 prefix（`// TODO:` のみ）
- ❌ 既存コメントを上書き
- ✅ 日本語出力
- ✅ source-of-truth 引用 + 行
- ✅ lean A 強制（Repository ↔ sqlc gen soft、body 業務ロジック禁止 strict、pgerror strict、IF impl coverage）
- ✅ multi-query / switch dispatch / JOIN を body 構造から自動推論（実装コードに annotation 不要）
- ✅ suggestion 検出位置に TODO hand-off コメント追加（opt-in 既定 ON）、既存コメントあれば skip
- ✅ README + sqlc gen + domain IF を毎回再読

## チェックリスト

- [ ] scope + TODO opt 確認 or 受領
- [ ] CLAUDE.md / infra README / rdb README / sqlc gen / domain Repository IF 読込
- [ ] `make lint` infra 絞り込み
- [ ] Repository ↔ sqlc gen 1:1 チェック
- [ ] body composition heuristic（multi-query / dispatch を body 構造から自動推論）
- [ ] `pgerror.NormalizeError` 利用チェック
- [ ] tracer span チェック
- [ ] findings に source-of-truth 引用
- [ ] レポート日本語
- [ ] suggestion 検出位置に TODO hand-off コメント挿入（opt-in 時）、既存コメントあれば skip
- [ ] TODO 追加以外のコード変更なし
