> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# New Spec — Domain

1 feature の domain 層 spec テンプレートを作成するスキル。

## 使うとき

- 新規 feature 開始で domain 層 spec テンプレが欲しい
- 既存 feature ディレクトリに domain spec だけ追加（他層は既存）

以下の用途には使いません:

- 既存 `domain.md` の編集 — エディタで直接
- spec から Go コード生成 — `scaffold-domain`
- spec 整合性検証 — `verify-spec`
- 2 層 (domain + usecase) を一括作成 — 統合 `new-spec`（lean A: controller / infra は OpenAPI gen + sqlc gen から導出されるため spec 不要）

## 読み書き範囲

**読み込み（常時）**:

- `.claude/scaffold-spec/domain-spec.md` — domain 層の canonical 節リスト
- `docs/spec/<feature>/` — `domain.md` 既存確認

**書き込み（承認後）**:

- `docs/spec/<feature>/domain.md` — テンプレートファイル

**触らない**:

- 既存 `domain.md`（あれば中断）
- 他層の spec ファイル

## 最初のステップ: identity 確認

`AskUserQuestion` 起動直後（`new-spec` 統合から context 提供時は除く）:

1. **feature 名** — フリーテキスト、kebab-case。`^[a-z][a-z0-9-]*$` 検証
2. **aggregate 名** — フリーテキスト、PascalCase（例: `User`, `Order`）。Go struct 名になる

`docs/spec/<feature>/domain.md` 既存なら中断。未存在なら `docs/spec/<feature>/` を作成。

## Step 1. 節定義の読み込み

`.claude/scaffold-spec/domain-spec.md` から canonical 節リストを抽出。ハードコードしない — このファイルが source of truth。

現行節リスト（実行時にミラー）:

1. Overview
2. Entity
3. Cross-field Invariants
4. Behavior Methods
5. Value Objects
6. Repository Methods

## Step 2. テンプレ生成

H1 `<FeatureName Display> — Domain Spec`、続けて各節:

- Overview: `TODO:` 1 行
- Entity: YAML（`package` / `struct` / `fields` プレースホルダ 1 件）
- Cross-field Invariants: bullet + `TODO:`
- Behavior Methods: YAML プレースホルダ
- Value Objects: YAML + 注「利用しない場合は節ごと削除」
- Repository Methods: YAML プレースホルダ

`.claude/scaffold-spec/domain-spec.md` の出力例を template として使用。

## Step 3. 承認と書き込み

提案パス + テンプレ冒頭 20 行を表示:

- 「以下の内容で `docs/spec/<feature>/domain.md` を作成しますか？」
- 選択肢: 「作成する」 / 「キャンセル」

承認時は `mkdir -p docs/spec/<feature>` + `Write`。

## Step 4. クロージング

```text
docs/spec/<feature>/domain.md を作成しました。次は editor で TODO を埋めてください。
usecase spec も必要なら new-spec-usecase または統合 new-spec を使ってください
（lean A 構成: controller / infra spec は不要、OpenAPI gen + sqlc gen から導出されます）。
```

## AI 修正スコープ

- 書き込み: `docs/spec/<feature>/` 配下の新規ファイルのみ
- `domain.md` 既存時は中断

## 制約事項

- ❌ 既存 `domain.md` の上書き
- ❌ 業務内容（フィールド / メソッド）を発明
- ❌ 節リストをハードコード（必ず `.claude/scaffold-spec/domain-spec.md` から読む）
- ❌ identity `AskUserQuestion` をスキップ
- ❌ `domain.md` 以外の layer に触る
- ✅ ユーザー向け出力は日本語
- ✅ feature 名 kebab-case + aggregate 名 PascalCase を検証

## チェックリスト

- [ ] feature 名 + aggregate 名を `AskUserQuestion` で確認
- [ ] `.claude/scaffold-spec/domain-spec.md` を読んで現行節リスト取得
- [ ] `domain.md` 未存在（あれば中断）
- [ ] H2 節 + YAML コードブロック + TODO でテンプレを書き出し
- [ ] 最終サマリは日本語
- [ ] `docs/spec/<feature>/domain.md` のみ書き込み
