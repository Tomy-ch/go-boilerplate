> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# New Spec — Usecase

1 feature の usecase 層 spec テンプレートを作成するスキル。

## 使うとき

- 新規 feature 開始で usecase 層 spec テンプレが欲しい
- 既存 feature ディレクトリに usecase spec だけ追加

以下の用途には使いません:

- 既存 `usecase.md` の編集 — エディタで直接
- spec から Go コード生成 — `scaffold-usecase`
- spec 整合性検証 — `verify-spec`
- 2 層 (domain + usecase) を一括作成 — 統合 `new-spec`（lean A: controller / infra は OpenAPI gen + sqlc gen から導出されるため spec 不要）

## 読み書き範囲

**読み込み（常時）**:

- `.claude/scaffold-spec/usecase-spec.md` — usecase 層の canonical 節リスト
- `docs/spec/<feature>/` — `usecase.md` 既存確認

**書き込み（承認後）**:

- `docs/spec/<feature>/usecase.md` — テンプレートファイル

**触らない**:

- 既存 `usecase.md`（あれば中断）
- 他層の spec ファイル

## 最初のステップ: identity 確認

`AskUserQuestion` 起動直後（`new-spec` 統合から context 提供時は除く）:

1. **feature 名** — フリーテキスト、kebab-case
2. **usecase パッケージ名** — フリーテキスト、lowercase（例: `user`, `order`）
3. **Usecase interface 名** — フリーテキスト、既定 `Usecase`（PascalCase）

`docs/spec/<feature>/usecase.md` 既存なら中断。

## Step 1. 節定義の読み込み

`.claude/scaffold-spec/usecase-spec.md` から canonical 節リスト:

1. Overview
2. Interface
3. DTOs
4. Dependencies
5. Workflow

ハードコードしない — 実行時に spec format ファイル再読込。

## Step 2. テンプレ生成

H1 `<FeatureName Display> — Usecase Spec`、続けて各節:

- Overview: `TODO:` 1 行
- Interface: YAML（`package` / `name` / `methods` プレースホルダ 1 件）
- DTOs: YAML プレースホルダ DTO 構造体
- Dependencies: YAML プレースホルダ boundary リスト
- Workflow: メソッドごと H3 + YAML（`tx_required` / `steps` / `calls` / `errors`）

`.claude/scaffold-spec/usecase-spec.md` の出力例を template として使用。

## Step 3. 承認と書き込み

提案パス + テンプレ冒頭 20 行を表示:

- 「以下の内容で `docs/spec/<feature>/usecase.md` を作成しますか？」
- 選択肢: 「作成する」 / 「キャンセル」

## Step 4. クロージング

```text
docs/spec/<feature>/usecase.md を作成しました。次は editor で TODO を埋めてください。
domain spec も必要なら new-spec-domain または統合 new-spec を使ってください
（lean A 構成: controller / infra spec は不要、OpenAPI gen + sqlc gen から導出されます）。
```

## AI 修正スコープ

- 書き込み: `docs/spec/<feature>/` 配下の新規ファイルのみ
- `usecase.md` 既存時は中断

## 制約事項

- ❌ 既存 `usecase.md` の上書き
- ❌ 業務内容（メソッド / DTO / workflow）を発明
- ❌ 節リストをハードコード
- ❌ identity `AskUserQuestion` をスキップ
- ❌ `usecase.md` 以外の layer に触る
- ✅ ユーザー向け出力は日本語
- ✅ feature 名 kebab-case + パッケージ名 lowercase を検証

## チェックリスト

- [ ] feature + パッケージ + interface 名を `AskUserQuestion` で確認
- [ ] `.claude/scaffold-spec/usecase-spec.md` を読んで現行節リスト取得
- [ ] `usecase.md` 未存在（あれば中断）
- [ ] H2 節 + YAML + TODO でテンプレを書き出し
- [ ] 最終サマリは日本語
- [ ] `docs/spec/<feature>/usecase.md` のみ書き込み
