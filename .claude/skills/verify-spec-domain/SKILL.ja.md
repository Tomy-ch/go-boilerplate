> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# Verify Spec — Domain

`docs/spec/<feature>/domain.md` を format + entity ↔ SQL 対応で検証するスキル。

## 使うとき

- `scaffold-domain` 起動前の spec 不整合検知
- `domain.md` 編集後の単独確認
- `verify-spec` 統合から chain

以下の用途には使いません:

- 実装 ↔ spec drift — `arch-check-domain` 担当
- 不整合の修正 — read-only、レポートのみ

## source of truth（毎回読む）

| ソース | 用途 |
| --- | --- |
| `.claude/scaffold-spec/domain-spec.md` | domain.md の必須 H2 節 + YAML schema |
| `.claude/scaffold-spec/verify-rules.md` | 検証スコープ（format + spec ↔ 派生元） |
| `docs/spec/<feature>/domain.md` | 検証対象 spec ファイル |
| `database/migrations/*.sql` | entity ↔ カラムチェック用 `CREATE TABLE` |

## 最初のステップ: 対象 feature 確認

単独実行時 `AskUserQuestion`:

- 質問: 「検証対象の feature 名を選んでください」
- 選択肢: `docs/spec/` 直下のサブディレクトリを列挙 + 規約外パス用のフリーテキスト

`verify-spec` 統合から chain 時は feature 名提供済みのためスキップ。

`docs/spec/<feature>/domain.md` 欠落時は明確メッセージで中断。

## Step 1. format 検査

1. `.claude/scaffold-spec/domain-spec.md` から必須 H2 節リスト取得
2. `domain.md` に全必須 H2 節が存在するか確認
3. 全 fenced YAML コードブロックをパース、エラーは `violation`
4. Entity フィールド YAML について必須キー (`name`, `type`) 確認
5. Behavior Method YAML について必須キー (`name`, `signature`) 確認
6. Repository Method YAML について必須キー (`name`, `signature`) 確認

## Step 2. entity ↔ SQL soft チェック

`database/migrations/*.sql` を読み、`domain.md` の Entity struct 名と一致する `CREATE TABLE <aggregate_plural>` を探す。続いて:

- `snake_case` カラム ↔ Entity YAML の `camelCase` field 名をマッピング
- **自動認識される正当な逸脱**（findings に含めない）:
  - Behavior Methods に宣言された メソッド形式値（Entity field でないため自然に対象外）
  - 複数 SQL カラムをラップする VO 型フィールド（Value Objects YAML から VO 解決して包含カラムを「対応済み」扱い）
- **`suggestion`（`violation` ではない）として報告**:
  - SQL カラムに対応 Entity field 無し（VO ラップ未検出） → 「永続化されないカラム」
  - Entity field に対応カラム無し + VO 解決もできない → 「SQL 対応カラムなし、計算値ならメソッド形式に書き換え推奨」
  - 型不一致 heuristic（`VARCHAR` vs `int` 等） → 確認推奨

## Step 3. 内部整合性チェック

- Behavior Methods が Entity fields を参照する場合（signature 内）、対応 field が Entity に存在するか
- Value Objects の factory が他 VO を参照する場合、その VO が定義されているか
- Cross-field Invariants で言及する field が Entity に存在するか

## Step 4. レポート（日本語）

```text
verify-spec-domain 結果（feature: <feature>）

[format] N 件
  - domain.md: 必須節 "Behavior Methods" が見つからない
  - domain.md L42 YAML パースエラー: ...

[entity ↔ SQL] K 件（suggestion only）
  - domain.md Entity に `phoneNumber` フィールドあり、SQL カラム未定義
    remediation: 計算値ならメソッド形式に書き換え、永続化必要なら migration 追加

[internal] M 件
  - Behavior Method `Deactivate` の signature が field `deactivatedAt` を参照、Entity 未定義

総計: violations <N+M>, suggestions <K>
```

空:

```text
verify-spec-domain 結果（feature: <feature>）
domain.md の違反は検出されませんでした（suggestions: 0）。
```

## Step 5. クロージング

- 単独実行: exit 0（情報的）
- `verify-spec` から chain: 違反数 + suggestion 数を caller に返す
- 自動修正しない

## AI 修正スコープ

完全 read-only。ファイル変更なし。

## 制約事項

- ❌ 違反の自動修正
- ❌ spec / source ファイル変更
- ❌ 節リストハードコード（必ず `.claude/scaffold-spec/domain-spec.md` から読む）
- ❌ 単独実行時の対象確認 `AskUserQuestion` をスキップ
- ❌ entity ↔ SQL 不一致を hard violation 扱い（必ず `suggestion`）
- ✅ 日本語出力
- ✅ source-of-truth 文書 + 行を引用
- ✅ 全チェックを 1 パスで実施

## チェックリスト

- [ ] 対象 feature を確認 or 受領
- [ ] `.claude/scaffold-spec/domain-spec.md` を毎回読み込み
- [ ] `domain.md` format チェック（節 + YAML）
- [ ] entity ↔ SQL soft チェック実施
- [ ] 内部整合性チェック実施
- [ ] findings に source-of-truth 引用
- [ ] レポート日本語
- [ ] ファイル変更なし
