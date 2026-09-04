> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# New Spec

1 feature の 2 layer spec テンプレートセット（lean A: domain spec 1 本 + usecase spec 1 本）を作成する統合スキル。

## 使うとき

- 新規 feature 開始で 2 layer 分の spec テンプレを 1 つの chain flow で作成
- 2 本のうち片方の spec が既にある feature で、欠落 layer を埋める

以下の用途には使いません:

- 単一 layer テンプレ作成 — 該当 per-layer skill を直接呼ぶ:
  - `new-spec-domain` / `new-spec-usecase`
- 既存 spec ファイル編集 — エディタで直接
- spec から Go コード生成 — `scaffold-endpoint`（または per-layer `scaffold-<layer>`）
- spec 整合性検証 — `verify-spec`

## なぜ 2 spec か（lean A）

controller / infra layer は spec 駆動ではなく、OpenAPI gen と sqlc gen から scaffold 時に導出される。規約自体は `arch-check` で強制される。詳細は `.codex/scaffold-spec/lifecycle.md` を参照。

## 読み書き範囲

本 skill 自体は spec ファイルを **直接読み書きしない**。全構造作業は per-layer skill に委譲し、各 per-layer skill が:

- 実行時に対応する `.codex/scaffold-spec/<layer>-spec.md` を読んで節リスト取得
- 自身の `ask the user explicitly` で layer 固有 identity 収集
- `docs/spec/domain/` または `docs/spec/usecase/` 配下に Markdown ファイル 1 つを書き込み

本 skill は orchestration のみ: feature 名 + layer 選択 + layer ごとのパッケージパスを確認し、依存順で per-layer skill を起動。

**feature は束ねかたであってディレクトリではない。** spec のツリーはパッケージパスで引かれる
（`docs/spec/<layer>/<rest>.md` ⇔ `internal/<layer>/<rest>`）ので、1 feature の 2 spec は別々の
ツリーに置かれ、パスが一致する必要はない — `internal/usecase/user/search` は `internal/domain/user`
と同じ feature である。ここで集める feature 名は束ねのラベルであり、用語表の所有列を埋めるもので、
ファイルの置き場所は決めない。

## 最初のステップ: feature と layer 選択の確認

まず `ask the user explicitly` を 1 回の batched call で 2 質問:

### Question 1: feature 名

- フリーテキスト: 「feature 名（kebab-case）。例: `user-management`, `order-fulfillment`」
- `^[a-z][a-z0-9-]*$` を検証。無効なら再質問
- これは束ねのラベル — レポートと用語表の所有列で feature を指す名前であり、**パスではない**

### Question 2: scaffold する layer

- multi-select、2 択:
  - 「domain」
  - 「usecase」
- 既定: 両方。一部 deselect 可（例: domain spec 既存で usecase のみ）

### 続けて: 選択した layer ごとのパッケージパス

2 回目の batched `ask the user explicitly`。選択した layer ごとにフリーテキスト 1 問。skip 判定が必要とするのは
パスであり、feature 名からは導出できないため:

- domain: 「domain パッケージパス（`internal/domain/` 以下）。例: `cart`, `product/category`」
- usecase: 「usecase パッケージパス（`internal/usecase/` 以下）。例: `address`, `product/ranking`, `user/search`」

いずれも `^[a-z][a-z0-9]*(/[a-z][a-z0-9]*)*$` を検証 — Go のパッケージ名にハイフンは無いので、
kebab-case の feature 名はたいてい妥当なパッケージパス**ではない**。feature 名の `-` を `/` に
置き換えたものを既定値として提示し、ユーザーに直させる。

回答後:

1. 作業計画作成: feature 名 + scaffold 対象の (layer, パッケージパス) 順序リスト（依存順 `domain` → `usecase`）
2. 各選択 layer について `docs/spec/<layer>/<pkgpath>.md` 既存確認。存在すれば chain 失敗ではなく **skip** マーク
3. 計画を日本語で skip マーク付きで表示、`ask the user explicitly` で確認:
   - 質問: 「以下の順番で per-layer skill を chain します。進めますか？」
   - 選択肢: 「進める」 / 「キャンセル」

skip 適用後に実行対象が空なら報告して停止:

```text
全 layer の spec ファイルが既に存在します。実行対象がありません。
```

## Step 1. 依存順で per-layer skill を chain

計画の各 layer について（依存順 `domain` → `usecase`）:

1. `Skill` tool で `new-spec-<layer>` を起動
2. chain context として feature 名**とその layer のパッケージパス**を渡し、child のパス `ask the user explicitly` をスキップ可能にする — layer 固有 identity（aggregate 名 / interface 名）のみ child が質問
3. child が自身の確認を実施しファイルを書き込む
4. child が失敗報告（user キャンセル等）したら chain 停止 + ステータス surface、書き込み済み layer の自動 rollback はしない

各 per-layer skill は独立 — feature 名と layer 実行順序のみ共有。パッケージパスは layer ごとに別の回答であり、aggregate / interface 名も skill 間で受け渡さない。

## Step 2. クロージングレポート

全選択 layer 処理後（chain 完了 / 中途停止のいずれも）日本語サマリ:

```text
new-spec 完了（feature: <feature>）。
  ✓ domain   : 作成済み (docs/spec/domain/<domainPkgPath>.md)
  ✓ usecase  : 作成済み (docs/spec/usecase/<usecasePkgPath>.md)

次のアクション:
  - editor で TODO を埋める
  - 2 spec 揃ったら verify-spec で format / 派生元 / cross-layer 参照を検証
  - 検証通過後に scaffold-endpoint で実装一括生成
    (controller / infra は OpenAPI / sqlc gen から自動導出される)
```

マーク:

- ✓ = 新規作成
- `-` = スキップ（ファイル既存）
- ✗ = 失敗 / キャンセル

commit しない。scaffold skill 自動起動しない。

## AI 修正スコープ

本 skill 自体はファイル書き込みなし。全 write スコープは per-layer skill に委譲、各 `docs/spec/<layer>/<pkgpath>.md` にスコープ。

## 制約事項

- ❌ spec ファイル直接書き込み（必ず per-layer skill 委譲）
- ❌ 節リストハードコード（per-layer skill が `.codex/scaffold-spec/<layer>-spec.md` を実行時読み込み）
- ❌ 1 layer の既存ファイル理由で chain 全体中断 — skip マークして継続
- ❌ 後段失敗時に earlier layer 自動 rollback
- ❌ feature 確認 `ask the user explicitly` をスキップ
- ❌ layer のパッケージパスを尋ねずに feature 名から導出（kebab-case はパッケージパスではなく、2 layer のパスは食い違う）
- ❌ per-layer skill を依存順以外で起動
- ❌ controller / infra を spec 選択肢として提示（lean A: 両層は導出、spec 駆動ではない）
- ✅ ユーザー向け出力は日本語
- ✅ multi-select layer 選択（domain / usecase）、既定は両方
- ✅ 依存順 `domain` → `usecase` で chain
- ✅ 最終レポートで layer ごとに skip / fail ステータス surface

## チェックリスト

- [ ] feature 名 + layer 選択 + layer ごとのパッケージパスを `ask the user explicitly` で確認
- [ ] 既存 layer ファイルを skip マーク（失敗扱いにしない）
- [ ] per-layer skill を依存順で起動
- [ ] 各 per-layer skill が自身の identity `ask the user explicitly` を実施
- [ ] per-layer 失敗時は chain 停止、自動 rollback なし
- [ ] 最終日本語サマリで ✓ / - / ✗ マーク使用
- [ ] 本 skill 自身による直接 file 書き込みなし
- [ ] commit / scaffold 自動起動なし
