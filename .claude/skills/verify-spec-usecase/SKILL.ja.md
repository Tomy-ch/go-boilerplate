> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# Verify Spec — Usecase

`docs/spec/<feature>/usecase.md` を format + cross-spec 参照 + 命名規約で検証するスキル。OpenAPI ↔ usecase mapping は `scaffold-controller` が scaffold 時に検証する（usecase は HTTP/OpenAPI を知らない — 依存方向）。

## 使うとき

- `scaffold-usecase` 起動前の spec 不整合検知
- `usecase.md` 編集後の単独確認
- `verify-spec` 統合から chain

以下の用途には使いません:

- 実装 ↔ spec drift — `arch-check-usecase` 担当
- 不整合の修正 — read-only、レポートのみ

## source of truth（毎回読む）

| ソース | 用途 |
| --- | --- |
| `.claude/scaffold-spec/usecase-spec.md` | usecase.md の必須 H2 節 + YAML schema |
| `.claude/scaffold-spec/verify-rules.md` | 検証スコープ（format + cross-spec + 命名規約） |
| `docs/spec/<feature>/usecase.md` | 検証対象 spec ファイル |
| `docs/spec/<feature>/domain.md` | cross-spec `calls:` 解決用 |
| `internal/usecase/README.md` | 命名規約（動詞接頭辞、Usecase interface 命名） |
| `internal/usecase/<sibling>/*.go` | README に規約記述がない場合の fallback — 既存パターンを観察 |

## 最初のステップ: 対象 feature 確認

単独実行時 `AskUserQuestion`:

- 質問: 「検証対象の feature 名を選んでください」
- 選択肢: `docs/spec/` 直下のサブディレクトリを列挙 + 規約外パス用のフリーテキスト

`verify-spec` 統合から chain 時は feature 名提供済みのためスキップ。

`docs/spec/<feature>/usecase.md` 欠落時は明確メッセージで中断。

## Step 1. format 検査

1. `.claude/scaffold-spec/usecase-spec.md` から必須 H2 節リスト取得
2. `usecase.md` に全必須 H2 節が存在するか確認
3. 全 fenced YAML コードブロックをパース、エラーは `violation`
4. Interface method YAML について必須キー (`name`, `signature`) 確認
5. Workflow entry YAML について必須キー (`tx_required`, `steps`, `calls`, `errors`) 確認
6. Dependency YAML について boundary or Repository 参照として認識可能か確認

## Step 2. cross-spec 参照チェック

`docs/spec/<feature>/domain.md` を読んで inventory 構築:

| inventory | source |
| --- | --- |
| `domain.repository_methods` | `domain.md` Repository Methods (name 一覧) |
| `domain.behavior_methods` | `domain.md` Behavior Methods (name 一覧) |
| `domain.factory` | `domain.md` Entity struct 名 + Value Objects factory 名 |
| `usecase.dependencies` | `usecase.md` Dependencies (boundary 名) |

`usecase.md` Workflow の各エントリの `calls:` リストを分類:

- `<aggregate>.Repository.<Method>` → `domain.repository_methods` に存在 → 不在は `violation`
- `<aggregate>.<BehaviorMethod>` or `<aggregate>.New` → `domain.behavior_methods` or `domain.factory` に存在 → 不在は `violation`
- `<boundary>.<Method>`（例: `clock.Now`, `tx.Do`） → boundary が `usecase.dependencies` に存在 → 不在は `violation`（method 自体は compile-time）

## Step 3. 命名規約チェック (lean A)

なぜこの step があるか: scaffold-controller は scaffold 時に OpenAPI operationId → usecase method mapping を機械的に導出する。その導出が決定論的に成立するために、usecase method は一貫した命名規約に従う必要がある。本 check は spec を **OpenAPI を参照せず** に検証する（依存方向逆転を避ける — usecase は HTTP を知らない）。

source of truth（順に読む）:

1. `internal/usecase/README.md` — 命名規約が明示されていればそこから
2. 既存 `internal/usecase/<sibling>/*.go` — fallback として現行コードベースの観察パターン（使われている action verb）

`usecase.md` Interface について:

- **Usecase interface 名**: プロジェクト規約に従う（典型的には package ごとに `Usecase`、または README 記載通り）
- **method 名**: 認識される action verb 接頭辞（例: `List`, `Create`, `Get`, `Update`, `Delete`, `Register`, `Activate`, `Deactivate` 等 — README / sibling から導出、ここでハードコードしない）
- **HTTP terminology の混入を避ける**（例: `Post`, `Put`, `Patch` → domain action verb への rename 推奨）

各 finding → `suggestion`（規約への準拠は推奨、blocker ではない。命名規約違反は scaffold-controller 側で mapping 失敗として最終的に surface される）。

## Step 4. Workflow 内部整合性

- `tx_required: true` の Workflow entry が `tx.Manager` boundary を `calls` に含むか確認
- `errors` リストが domain で定義された error 型を参照しているか（部分一致でいい、命名規則チェック）

## Step 5. レポート（日本語）

```text
verify-spec-usecase 結果（feature: <feature>）

[format] N 件
  - usecase.md: 必須節 "Workflow" が見つからない
  - usecase.md L42 YAML パースエラー: ...

[cross-spec] M 件
  - usecase.md CreateUser calls 'user.Repository.Save' が domain.md Repository Methods に存在しない
  - usecase.md ActivateUser calls 'clock.Now' だが Dependencies に clock 無し

[naming convention] K 件（suggestion）
  - usecase Interface method `PostUser` は HTTP verb 由来命名
    source: internal/usecase/README.md / 既存 sibling pkg のパターン
    remediation: `CreateUser` 等の action verb prefix に rename 推奨

[internal] L 件
  - Workflow `Register` の tx_required:true だが calls に tx.Manager 無し

総計: violations <N+M>, suggestions <K+L>
```

空:

```text
verify-spec-usecase 結果（feature: <feature>）
usecase.md の違反は検出されませんでした（suggestions: 0）。
```

## Step 6. クロージング

- 単独実行: exit 0（情報的）
- `verify-spec` から chain: 違反数 + suggestion 数を caller に返す
- 自動修正しない

## AI 修正スコープ

完全 read-only。ファイル変更なし。

## 制約事項

- ❌ 違反の自動修正
- ❌ spec / source ファイル変更
- ❌ 節リストハードコード（必ず `.claude/scaffold-spec/usecase-spec.md` から読む）
- ❌ 単独実行時の対象確認 `AskUserQuestion` をスキップ
- ❌ OpenAPI operationId カバレッジを検査 — 依存方向逆転（usecase は HTTP/OpenAPI を知らない）。OpenAPI ↔ usecase mapping は `scaffold-controller` の責務
- ❌ 命名規約違反を hard violation 扱い（必ず `suggestion`）
- ✅ 日本語出力
- ✅ source-of-truth 文書 + 行を引用
- ✅ 全チェックを 1 パスで実施

## チェックリスト

- [ ] 対象 feature を確認 or 受領
- [ ] `.claude/scaffold-spec/usecase-spec.md` を毎回読み込み
- [ ] `usecase.md` format チェック（節 + YAML）
- [ ] cross-spec inventory 構築用に `domain.md` 読み込み
- [ ] cross-spec `calls:` 参照を検証
- [ ] 命名規約チェック（`internal/usecase/README.md` + sibling pkg パターン）
- [ ] Workflow 内部整合性確認
- [ ] findings に source-of-truth 引用
- [ ] レポート日本語
- [ ] ファイル変更なし
