> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# New Spec — Usecase

1 feature の usecase 層 spec テンプレートを作成するスキル。

## 使うとき

- 新規 feature 開始で usecase 層 spec テンプレが欲しい
- domain spec が既にある feature に usecase spec だけ追加

以下の用途には使いません:

- 既存 usecase spec の編集 — エディタで直接
- spec から Go コード生成 — `scaffold-usecase`
- spec 整合性検証 — `verify-spec`
- 2 層 (domain + usecase) を一括作成 — 統合 `new-spec`（lean A: controller / infra は OpenAPI gen + sqlc gen から導出されるため spec 不要）

## 読み書き範囲

**読み込み（常時）**:

- `.claude/scaffold-spec/usecase-spec.md` — usecase 層の canonical 節リスト
- `docs/spec/usecase/<pkgpath>.md` — 既存確認

**書き込み（承認後）**:

- `docs/spec/usecase/<pkgpath>.md` — テンプレートファイル

**触らない**:

- 当該パスの既存 usecase spec（あれば中断）
- 他層の spec ファイル

## 最初のステップ: identity 確認

spec の同一性は**パッケージパス**である: `docs/spec/usecase/<rest>.md` は `internal/usecase/<rest>` に対応し、Interface YAML の `package:` は同じパスを宣言する。パッケージパスは旧来の feature 名 + パッケージ名の 2 つを置き換える 1 つの識別子であり、ファイルの置き場所を決めるのはこれである。

`AskUserQuestion` 起動直後（`new-spec` 統合から context 提供時は除く）:

1. **usecase パッケージパス** — フリーテキスト、`internal/usecase/` 以下のパス（例: `address`, `product/ranking`, `user/search`）。`^[a-z][a-z0-9]*(/[a-z][a-z0-9]*)*$` 検証 — Go のパッケージ名にハイフンは無いので `user-search` はパッケージパスではなく、入れ子の `user/search` がそれにあたる
2. **Usecase interface 名** — フリーテキスト、既定 `Usecase`（PascalCase）

`docs/spec/usecase/<pkgpath>.md` 既存なら中断。未存在なら親ディレクトリを作成。

## Step 1. 節定義の読み込み

`.claude/scaffold-spec/usecase-spec.md` から canonical 節リスト:

1. Overview
2. Interface
3. DTOs
4. Dependencies
5. Workflow

ハードコードしない — 実行時に spec format ファイル再読込。

## Step 2. テンプレ生成

H1 `<Package Display> — Usecase Spec`（パッケージパスを Title Case にしたもの。`user/search` → `User Search`）、続けて各節:

- Overview: `TODO:` 1 行
- Interface: YAML（`package` / `name` / `methods` プレースホルダ 1 件）。`package:` は必ず `internal/usecase/<pkgpath>` — ファイルが置かれるパスと同じで、`verify-spec` がこれを assert する
- DTOs: YAML プレースホルダ DTO 構造体
- Dependencies: YAML プレースホルダ boundary リスト。この節は突き合わせ先の domain spec を解決する場所でもある — Repository の項目は `internal/domain/<X>` を名指しし、`verify-spec` はそこから `docs/spec/domain/<X>.md` を読む。このファイル自身のパスからは導出しない
- Workflow: メソッドごと H3 + YAML（`tx_required` / `steps` / `calls` / `errors`）

`.claude/scaffold-spec/usecase-spec.md` の出力例を template として使用。

## Step 3. 承認と書き込み

提案パス + テンプレ冒頭 20 行を表示:

- 「以下の内容で `docs/spec/usecase/<pkgpath>.md` を作成しますか？」
- 選択肢: 「作成する」 / 「キャンセル」

承認時はファイルの親ディレクトリ（`docs/spec/usecase/<pkgpath>` から末尾セグメントを除いたもの）を `mkdir -p` して `Write`。

## Step 4. クロージング

```text
docs/spec/usecase/<pkgpath>.md を作成しました。次は editor で TODO を埋めてください。
domain spec も必要なら new-spec-domain または統合 new-spec を使ってください
（lean A 構成: controller / infra spec は不要、OpenAPI gen + sqlc gen から導出されます）。
```

## AI 修正スコープ

- 書き込み: `docs/spec/usecase/` 配下の新規ファイルのみ
- 当該パスの usecase spec 既存時は中断

## 制約事項

- ❌ 既存 usecase spec の上書き
- ❌ 業務内容（メソッド / DTO / workflow）を発明
- ❌ 節リストをハードコード
- ❌ identity `AskUserQuestion` をスキップ
- ❌ `docs/spec/usecase/` 以外のツリーに触る
- ✅ ユーザー向け出力は日本語
- ✅ パッケージパス（小文字・`/` 区切り・ハイフン無し）を検証

## チェックリスト

- [ ] usecase パッケージパス + interface 名を `AskUserQuestion` で確認
- [ ] `.claude/scaffold-spec/usecase-spec.md` を読んで現行節リスト取得
- [ ] 当該パスの usecase spec 未存在（あれば中断）
- [ ] Interface YAML の `package:` がファイルのパスと一致
- [ ] H2 節 + YAML + TODO でテンプレを書き出し
- [ ] 最終サマリは日本語
- [ ] `docs/spec/usecase/<pkgpath>.md` のみ書き込み
