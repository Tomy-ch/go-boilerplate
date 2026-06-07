> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# Arch Check — Domain

`internal/domain/**` を domain 層アーキ規則で監査するスキル。

## 使うとき

- `internal/domain/` 変更を commit 直前に layer 固有のチェックをかけたい
- domain entity / Repository IF を触ったリファクタのレビュー
- 単独 or `arch-check` 統合から chain

以下の用途には使いません:

- 他層の監査 — 対応する `arch-check-<layer>` を使う
- formatting / style — `make fix` / `make lint`

## source of truth（毎回読む）

| ソース | 用途 |
| --- | --- |
| `CLAUDE.md`（Layer Rules / Forbidden Shortcuts） | 上位制約 |
| `internal/domain/README.md` | domain 層規則（純粋性、許可 import、value vs interface 規約） |
| `internal/domain/<aggregate>/*.go`（既存 aggregate package） | sibling code からのパターン参照（aggregate ごとの README は意図的に作らない — 原則は top-level domain README に集約） |
| `database/migrations/*.sql` | SQL `CREATE TABLE` で entity ↔ カラムチェック（lean A） |
| `.golangci.yaml` `depguard:` | 既に強制済み — 重複しない |

## 最初のステップ: スコープ確認 + TODO opt

単独実行時 `AskUserQuestion` を 2 問:

1. 質問: 「domain 層のどのスコープで検査しますか？」 / 選択肢: 「変更ファイルのみ」 / 「internal/domain/ 全体」 / 「キャンセル」
2. 質問: 「suggestion 検出箇所に TODO hand-off コメントを追加しますか？」 / 選択肢: 「追加する（既定）」 / 「追加しない（read-only）」

`arch-check` 統合から chain 時は scope + TODO opt 提供済みのためスキップ。

## Step 1. ファイルスコープ解決

```sh
# 変更ファイルのみ (domain に絞り込み)
BASE=$(gh pr view --json baseRefName -q '.baseRefName' 2>/dev/null || gh repo view --json defaultBranchRef -q '.defaultBranchRef.name')
git diff --name-only "origin/${BASE}...HEAD" -- 'internal/domain/**/*.go' | grep -vE '\.gen\.go$|_mock\.go$|_test\.go$' || true

# 全体
git ls-files 'internal/domain/**/*.go' | grep -vE '\.gen\.go$|_mock\.go$|_test\.go$'
```

空 → 報告して exit。

## Step 2. lint baseline（domain スコープ）

```sh
make lint 2>&1 | tee /tmp/arch-check-domain-lint.out
```

出力を `internal/domain/` パスに絞り込む。depguard / forbidigo / gosec を `violation` で記録。

lint 自体が無関係な理由で失敗時は verbatim 出力で中断。

## Step 3. semantic check

各 domain Go ファイルについて:

1. `import (...)` ブロック抽出
2. 各 import を `internal/domain/README.md` の許可 / 禁止リストと突き合わせ。よくある禁止パターン:
   - `go.uber.org/zap`, `go.uber.org/fx` — logging / DI framework
   - `github.com/labstack/echo`, `database/sql`, `gorm` 等 — framework / infra
   - `time.Now()` / `uuid.New()` 呼び出し — `internal/domain/README.md` "Handling time and ID" 節により time / ID 生成は Domain 外（`time.Time` / `uuid.UUID` を値型として使うのは可）
   - `context.Context`: **Repository interface のメソッドシグネチャでのみ許可**（README の Repository 例による）。それ以外の利用（domain entity behavior method、VO factory 等）は `suggestion`。CLAUDE.md は domain で context.Context を一括禁止としているが、README が Repository IF の例外を明示しているため、README > Code > SKILL の優先度で README に従う
3. entity ファイル（Aggregate Root の struct 定義）について SQL migration と **soft 突き合わせ** — **実装コードに arch-check 専用 annotation を導入しない**。正当な逸脱はコード構造から自動推論する:
   - `database/migrations/*.sql` から `CREATE TABLE <aggregate_plural>` を探す（テーブル定義 migration が複数あれば最新で再構成）
   - `snake_case` カラム ↔ `camelCase` struct フィールドをマッピング
   - **自動認識される正当な逸脱**（findings に含めない）:
     - メソッド形式の計算値（例: `func (u User) FullName() string`） — field でないので自然に検査対象外
     - 複数 SQL カラムをラップする VO 型フィールド（例: `Money` フィールドの型が `Currency` + `Amount` を包含） — VO 型を解決して包含カラムを「対応済み」扱い
     - フィールド無しの method-only struct
   - **`suggestion`（`violation` ではない）として報告**:
     - SQL カラムに対応 struct field なし（VO ラップも未検出） → 「永続化されないカラムです。entity 追加か migration 削除を検討」
     - struct field に対応カラムなし + VO 解決もできない → 「SQL に対応カラムなし。計算値ならメソッド形式に書き換え、不要なら削除を検討」
     - 型不一致（例: `VARCHAR` vs `int`） → 確認推奨

   **規約（annotation ではない）**: 計算値は **メソッド形式** で書くのが既存規約。これにより struct field 検査の対象外となり、annotation 不要で正当な逸脱を表現できる。

   理由: 1:1 strict は理想モデルで、現実には計算値 / VO 群化 / 入力分解 / audit 由来値等の正当な逸脱がある。検出は人間レビューの起点とし、blocker にしない。実装コードに arch-check 専用 annotation を入れない（補助ツールはコードを読んで自動推論する設計）。

## Step 4. TODO hand-off コメント挿入 (opt)

scope 確認時に「TODO 追加」を opt-in した場合（既定 ON）、Step 3 の `suggestion` レベル findings 各々について:

1. ソース内の逸脱位置を特定（struct field 行 / ファイル）
2. 逸脱位置の直上 3 行以内に既存コメントブロックがあれば **skip**（重複防止）
3. なければ `// TODO:` コメントを逸脱位置直前に挿入:
   - 何が検出されたか記述（例: 「struct field has no matching SQL column」）
   - 人間向けの解決選択肢を列挙（migration / メソッド形式 / in-memory + WHY 等）
   - 標準 `// TODO:` 接頭辞のみ使用（`// TODO(arch-check):` 等の AI 識別子付けない）

コメントは **人間に判断を委ねる hand-off baton** であって、AI の判定ではない。AI は「逸脱が意図的か bug か」を決めず、人間がコード修正 or TODO を WHY コメントに置き換えて解決する。

例:

```go
// TODO: User 構造体に phoneNumber フィールドあり、SQL カラム未定義。
// 永続化が必要なら migration 追加、計算値ならメソッド形式に書き換え、
// in-memory 保持なら本コメントを WHY 説明に置き換えてください。
phoneNumber string
```

`violation` レベルの finding には TODO コメントを付けない（修正一択、defer すべきでない）。

## Step 5. レポート（日本語）

```text
arch-check-domain 結果（スコープ: <scope>）

[lint] N 件
  - <file:line>: <linter>: <message>

[semantic] M 件
  internal/domain/foo/foo.go:12
    violation: "go.uber.org/zap" を直接 import している
    source: internal/domain/README.md L42 "domain 層は logging framework を直接利用しない"

[entity ↔ SQL] K 件（suggestion only）
  internal/domain/user/user.go vs database/migrations/0003_users.sql
    suggestion: User 構造体に `phoneNumber` フィールドあり、SQL カラム未定義
    remediation: 計算値ならメソッド形式に書き換え、永続化必要なら migration 追加、VO ラップで複数カラム束ねるなら型変更を検討

総計: violations <N+M>, suggestions <K+...>
TODO hand-off: 追加 <P> 件, スキップ <Q> 件（既存コメント）
```

空の結果:

```text
arch-check-domain 結果（スコープ: <scope>）
domain 層の違反は検出されませんでした。
```

## Step 6. クロージング

- 単独実行: レポート出力して exit。violations があっても exit 0（情報的）
- `arch-check` から chain: 違反数 + TODO 追加数を caller に返す（integrator が集約報告）
- 自動修正しない（コード本体は触らず、TODO hand-off コメントのみ追加）

## AI 修正スコープ

**ほぼ read-only**、ただし 1 つだけ narrow な write scope あり:

- 読み込み: `CLAUDE.md` / READMEs / `database/migrations/*.sql` / in-scope Go ファイル。`make lint` 実行（`/tmp/*.out` 書き込み）
- 書き込み（scope 確認時の user opt 時のみ）: `internal/domain/**/*.go` の逸脱位置への `// TODO:` hand-off コメント追加。**TODO 追加のみ** — コード変更なし、`violation` auto-fix なし、既存コメント修正なし
- 逸脱位置の直上 3 行以内に既存コメントブロックがあれば skip（重複防止）
- user が「TODO 追加なし」を opt した場合は完全 read-only

## 制約事項

- ❌ domain ルールをハードコード（必ず `internal/domain/README.md` から読む）
- ❌ depguard と重複したチェック
- ❌ ソースコード変更（逸脱位置への TODO hand-off コメントのみ user opt 時に許可）
- ❌ 単独実行時の scope 確認をスキップ
- ❌ `violation` レベルの finding に TODO コメント付与（修正一択、defer すべきでない）
- ❌ 逸脱が意図的か bug かを AI が判定（「AI が解決済み判定しない」 — TODO で人間に hand-off）
- ❌ TODO コメントに AI 識別 prefix（`// TODO:` のみ、`// TODO(arch-check):` 等は付けない）
- ❌ 既存コメントを逸脱位置で上書き / 修正
- ✅ ユーザー向け出力は日本語
- ✅ findings に source-of-truth 文書 + 行を引用
- ✅ entity ↔ SQL soft チェック（lean A; suggestion only、メソッド形式 / VO ラップを自動推論で対象外化）
- ✅ suggestion 検出位置に TODO hand-off コメント追加（opt-in 既定 ON）、既存コメントあれば skip
- ✅ READMEs を毎回再読

## チェックリスト

- [ ] scope + TODO opt を確認（単独） or 受領（chained）
- [ ] `CLAUDE.md` + `internal/domain/README.md` を毎回読み込み
- [ ] `make lint` 実行、出力を domain に絞り込み
- [ ] generated / test ファイル除外
- [ ] entity ファイルについて entity ↔ SQL soft チェック実施（suggestion only、escape hatch を respect）
- [ ] suggestion 検出位置に TODO hand-off コメント挿入（opt-in 時）、既存コメントあれば skip
- [ ] findings に source-of-truth 文書引用
- [ ] レポート日本語
- [ ] TODO 追加以外のコード変更なし
