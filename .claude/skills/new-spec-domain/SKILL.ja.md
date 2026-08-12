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
- `docs/spec/glossary.md` — 業務語彙の統括 spec。集約名を突き合わせる

**書き込み（承認後）**:

- `docs/spec/<feature>/domain.md` — テンプレートファイル
- `docs/spec/glossary.md` — 集約 1 行のみ。ユーザーが承認したときだけ

**触らない**:

- 既存 `domain.md`（あれば中断）
- 他層の spec ファイル
- 追加する 1 行以外の用語表の行

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

## Step 3.5. 集約を用語表へ登録する

**この時点で用語なのは集約名だけである。** このスキルが書くのは TODO のテンプレートであり、
フィールド・値オブジェクト・振る舞いはまだ決まっていない。登録できるものが他に無い。1 行、
あるいはゼロ行である。

`docs/spec/glossary.md` を読み、集約名を既存の行と突き合わせる。

- **別の所有 feature で既出** — 同音異義である。両方の行を並べて報告し、そこで止まる。同じ語を
  2 つの feature が所有している状態こそがこのページの捕まえる findings であり、どちらの名前が勝つかは
  業務がこれからどう話すかについての決定である。**ここで解決しないこと。**
- **同じ所有 feature で既出** — することは無い。その旨を述べる
- **未登録** — 1 行を提案し、`AskUserQuestion` で確認する
  （「用語表へ追加する」/「今回は追加しない」）

定義文は feature 名と集約名から草案を起こし、**草案であることを明言する**。誰も編集していない定義は
誰も合意していない定義であり、もっともらしく誤っている行より空の行のほうが価値がある。

feature がサンプル由来なら行を `sample-api:begin` / `sample-api:end` の内側へ、そうでなければ外側へ
置く。マーカーの反対側に置かれた行は、サンプルと共に消えるか、サンプルより長生きするかのどちらかになる。

## Step 4. クロージング

```text
docs/spec/<feature>/domain.md を作成しました。次は editor で TODO を埋めてください。
用語表には集約名だけを登録しました。TODO を埋めると値オブジェクトや状態の語が現れるので、
そのときは /glossary で用語表へ反映してください（新出用語・orphan・同音異義を出します）。
usecase spec も必要なら new-spec-usecase または統合 new-spec を使ってください
（lean A 構成: controller / infra spec は不要、OpenAPI gen + sqlc gen から導出されます）。
```

**`new-spec-usecase` に同じステップは無い。意図的である。** usecase spec が宣言するのは Interface・
DTO・Workflow であり、`CreatePurchase` や `PurchaseView` のようなアプリケーション層の名前は
用語表の基準（業務を知っている人がこれは業務の語だと認めるか）を通らない。ドメイン層が用語を導入し、
usecase 層はそれを使う。

これは**導入する側のドメイン層が在るあいだ成り立つ**。投影だけの feature にはその層が無く、
その場合は `/glossary` が読み取り側から拾う。このスキルの担当ではないのは、このスキルがまさに
集約を作るときに走るものだからである。

## AI 修正スコープ

- 書き込み: `docs/spec/<feature>/` 配下の新規ファイルと、`docs/spec/glossary.md` への 1 行追記
- `domain.md` 既存時は中断

## 制約事項

- ❌ 既存 `domain.md` の上書き
- ❌ 業務内容（フィールド / メソッド）を発明
- ❌ 節リストをハードコード（必ず `.claude/scaffold-spec/domain-spec.md` から読む）
- ❌ identity `AskUserQuestion` をスキップ
- ❌ `domain.md` 以外の layer に触る
- ❌ 用語表の同音異義を解決する / どちらの名前が勝つかを決める
- ❌ 集約名以外を登録する（他はまだ TODO）
- ❌ 草案であると述べずに定義文を書く
- ✅ ユーザー向け出力は日本語
- ✅ feature 名 kebab-case + aggregate 名 PascalCase を検証

## チェックリスト

- [ ] feature 名 + aggregate 名を `AskUserQuestion` で確認
- [ ] `.claude/scaffold-spec/domain-spec.md` を読んで現行節リスト取得
- [ ] `domain.md` 未存在（あれば中断）
- [ ] H2 節 + YAML コードブロック + TODO でテンプレを書き出し
- [ ] 集約名を `docs/spec/glossary.md` と突き合わせ（同音異義なら報告して停止）
- [ ] 用語表への追記は `AskUserQuestion` の後、サンプルマーカーの正しい側へ
- [ ] 最終サマリは日本語で、残りの用語を `/glossary` へ引き継ぐ
- [ ] `docs/spec/<feature>/domain.md` と用語表 1 行のみ書き込み
