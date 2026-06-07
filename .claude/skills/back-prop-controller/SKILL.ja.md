> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# Back-Prop — Controller

controller READMEs（reference snippet 込み）、controller 実装、controller 関連 skill body の drift を検出。

## 使うとき

- `internal/controller/` 編集後 commit 前
- 定期 hygiene check
- `back-prop` 統合から chain

以下の用途には使いません: 実装コード修正 / 単一ファイルアーキ準拠（`arch-check-controller`） / 生成（`scaffold-controller`）。

## 読み書き範囲

**読み込み**:

- `internal/controller/README.md` + `internal/controller/handler/README.md` — canonical（handler README に reference snippet）
- `internal/controller/**/*.go` — 実装（`*.gen.go` / `_mock.go` / `*_test.go` 除く）
- `internal/controller/handler/**/gen/server.gen.go` — OpenAPI gen（operationId ↔ handler method 突き合わせ用）
- `.claude/skills/arch-check-controller/SKILL.md` + `.claude/skills/scaffold-controller/SKILL.md`

**書き込み（user 承認 + 理由明示後のみ）**:

- controller READMEs
- controller 関連 skill SKILL.md

**触らない**: 実装コード、他 layer、生成 `gen/`。

## 最初のステップ: スコープ + 検出種別確認

`AskUserQuestion` 2 質問 batched:

1. 「back-prop-controller のスコープを選んでください」 / 「変更ファイルのみ」 / 「internal/controller/ 全体」 / 「キャンセル」
2. 「検出する drift 種別を選んでください（multi-select、既定 3 種類すべて）」 / (A) / (B) / (C)

Chained: スキップ。

## Step 1. 入力読み込み

1. controller README + handler README（rules + reference snippet）読み込み
2. in-scope `.go` ファイル parse（handler struct、BindHandler / New、`gen.NewStrictHandler` 利用、tracer span、ctxhelper 利用）
3. controller 関連 skill SKILL.md 読み込み

## Step 2. 検出 (A) README → Code drift

controller 固有の典型 drift check（README 準拠）:

- handler struct 名 `server`（小文字、reference snippet 準拠）
- Constructor `BindHandler(echo, tf, uc)`（`New` ではない、reference snippet 準拠）
- `BindHandler` 内: `gen.RegisterHandlers(e, gen.NewStrictHandler(&server{...}, nil))` パターン
- 全 handler method 冒頭で `s.tracer.Start(ctx); defer endSpan()`
- handler body = pure template（業務ロジックなし）
- DI: `fx.Invoke(<pkg>.BindHandler)`（`fx.Provide` ではない）
- operationId ↔ handler method 1:1（camelCase）
- Repository / infra import 禁止

各違反: `(rule, file:line, reasoning, 選択肢: コード修正 / README 緩和 / 例外)`。

## Step 3. 検出 (B) Code → README Undocumented Pattern

recurring pattern（≥3 ファイル）:

- multi-usecase orchestration pattern
- response 構築 pattern（paging utility 利用 等）
- auth context 取り扱い pattern

各 finding: `(pattern, 出現数, sample, reasoning, 選択肢: README 追記 / 無視 / refactor)`。

## Step 4. 検出 (C) Skill ↔ README Duplication

controller 関連 skill 内 enumerate rule:

- 同 rule が `internal/controller/README.md` or `handler/README.md` にあれば surface
- `(rule, skill 位置, README 位置, reasoning, 選択肢: skill slim / 両方維持)`

## Step 5. 集約レポート（日本語）

```text
back-prop-controller 結果（scope: <X>, 種別: A/B/C）

[A] README → Code drift  N 件
[B] undocumented pattern  M 件
[C] Skill duplication  K 件

総計 N+M+K 件。per-item 承認 / 棄却。
```

## Step 6. per-item user 判断

各 finding `AskUserQuestion`。doc / skill 変更承認 → AI **理由明示** + draft → user 最終確認 → 書き込み。

## Step 7. クロージング

```text
back-prop-controller 完了。
  処理 finding: N
    README 更新: <X> / Skill 簡略化: <Y> / コード修正委任: <Z> / 無視: <W>
  最終 make md-lint OK
```

## AI 修正スコープ

書き込み: controller READMEs + controller 関連 skill（承認 + 理由明示後）。
触らない: 実装コード、他 layer、生成 `gen/`。

## 制約事項

- ❌ 実装コードへの書き込み
- ❌ user 承認なしの自動更新
- ❌ 理由を述べずに draft 実行
- ❌ scope + 種別 `AskUserQuestion` スキップ
- ❌ recurring threshold 3 未満を surface
- ✅ 日本語出力、各 finding に reasoning、per-item 承認制
- ✅ README が canonical

## チェックリスト

- [ ] scope + 種別確認 / 受領
- [ ] READMEs + 実装 + skill 読み込み
- [ ] (A)(B)(C) 検出
- [ ] 各 finding に reasoning 明示
- [ ] per-item 承認 → 理由提示 → draft → 最終確認 → 書き込み
- [ ] 実装コード触らない
- [ ] サマリ surface
