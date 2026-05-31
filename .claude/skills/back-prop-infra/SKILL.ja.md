> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# Back-Prop — Infra

infra READMEs（3-level: infra / rdb / pgerror）、infra 実装、infra 関連 skill body の drift を検出。

## 使うとき

- `internal/infrastructure/` 編集後 commit 前
- 定期 hygiene check
- `back-prop` 統合から chain

以下の用途には使いません: 実装コード修正 / 単一ファイルアーキ準拠（`arch-check-infra`） / Repository 生成（`scaffold-infra-db`）。

## 読み書き範囲

**読み込み**:

- `internal/infrastructure/README.md` + `internal/infrastructure/rdb/README.md` + `internal/infrastructure/rdb/pgerror/README.md` — canonical（principles のみ、完全 snippet 無し）
- `internal/infrastructure/**/*.go` — 実装（`*.gen.go` / `_mock.go` / `*_test.go` 除く）
- `.claude/skills/arch-check-infra/SKILL.md` + `.claude/skills/scaffold-infra-db/SKILL.md`

**書き込み（user 承認 + 理由明示後のみ）**:

- infra READMEs (3-level)
- infra 関連 skill SKILL.md

**触らない**: 実装コード、SQL ファイル、sqlc 生成物、他 layer。

## 最初のステップ: スコープ + 検出種別確認

`AskUserQuestion` 2 質問 batched:

1. 「back-prop-infra のスコープを選んでください」 / 「変更ファイルのみ」 / 「internal/infrastructure/ 全体」 / 「キャンセル」
2. 「検出する drift 種別を選んでください（multi-select、既定 3 種類すべて）」 / (A) / (B) / (C)

Chained: スキップ。

## Step 1. 入力読み込み

1. 3-level infra READMEs（infra / rdb / pgerror）読み込み
2. in-scope `.go` ファイル parse（Repository struct、constructor、sqlc 呼び出し、pgerror 利用、tracer span）
3. infra 関連 skill SKILL.md 読み込み

## Step 2. 検出 (A) README → Code drift

infra 固有の典型 drift check:

- 全 Repository method の sqlc 呼び出し result が `pgerror.NormalizeError(err)` 経由（single-normalization-point、pgerror README）
- 全 Repository method 冒頭で `tracer.Start(ctx); defer endSpan()`（infra README Observability）
- Repository constructor が domain Repository IF 返却（concrete struct ではない、rdb README）
- DI: `fx.Provide(<pkg>.New)` in `internal/di/module/infrastructure.go`
- Repository body に業務ロジックなし（Prohibited 節）
- Repository body = data orchestration のみ（sqlc + 変換 + pgerror）

各違反: `(rule, file:line, reasoning, 選択肢: コード修正 / README 緩和 / 例外)`。

## Step 3. 検出 (B) Code → README Undocumented Pattern

recurring pattern（≥3 ファイル）:

- 変換 helper pattern（多行 → slice helpers）
- `gen.New(r.db.NewLoggingDB(ctx))` 配置 pattern
- 特定 dispatch pattern（status による switch）

各 finding: `(pattern, 出現数, sample, reasoning, 選択肢: README 追記 / 無視 / refactor)`。

## Step 4. 検出 (C) Skill ↔ README Duplication

infra 関連 skill 内 enumerate rule:

- 3 つの infra READMEs のどれかに同 rule あるか確認
- 重複: `(rule, skill 位置, README 位置, reasoning, 選択肢: skill slim / 両方維持)`

## Step 5. 集約レポート

```text
back-prop-infra 結果（scope: <X>, 種別: A/B/C）

[A] README → Code drift  N 件
[B] undocumented pattern  M 件
[C] Skill duplication  K 件

総計 N+M+K 件。per-item 承認 / 棄却。
```

## Step 6. per-item user 判断

各 finding `AskUserQuestion`。doc / skill 変更承認 → AI **理由明示** + draft → user 最終確認 → 書き込み。

## Step 7. クロージング

```text
back-prop-infra 完了。
  処理 finding: N
    README 更新: <X> / Skill 簡略化: <Y> / コード修正委任: <Z> / 無視: <W>
  最終 make md-lint OK
```

## AI 修正スコープ

書き込み: infra READMEs (3-level) + infra 関連 skill（承認 + 理由明示後）。
触らない: 実装コード、SQL ファイル、sqlc 生成物、他 layer。

## 制約事項

- ❌ 実装コードへの書き込み
- ❌ user 承認なしの自動更新
- ❌ 理由を述べずに draft 実行
- ❌ scope + 種別 `AskUserQuestion` スキップ
- ❌ recurring threshold 3 未満を surface
- ✅ 日本語出力、各 finding に reasoning、per-item 承認制
- ✅ READMEs が canonical（infra は textual principles のみ、sibling コードが de facto reference）

## チェックリスト

- [ ] scope + 種別確認 / 受領
- [ ] 3-level infra READMEs + 実装 + skill 読み込み
- [ ] (A)(B)(C) 検出
- [ ] 各 finding に reasoning 明示
- [ ] per-item 承認 → 理由提示 → draft → 最終確認 → 書き込み
- [ ] 実装コード触らない
- [ ] サマリ surface
