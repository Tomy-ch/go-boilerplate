> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# Back-Prop — Pkg

pkg READMEs（layer + sub-package）、pkg 実装、pkg 関連 skill body の drift を検出。

## 使うとき

- `pkg/` 編集後 commit 前
- 定期 hygiene check
- `back-prop` 統合から chain

以下の用途には使いません: 実装コード修正 / 単一ファイルアーキ準拠（`arch-check-pkg`）。

## 読み書き範囲

**読み込み**:

- `pkg/README.md` — layer canonical（Policy / Package List / Package Details / Checklist for Adding a New Package）
- `pkg/<name>/README.md` — sub-package READMEs（`Public API` 必須、`Wraps` は third-party ラップ時のみ、`Notes` は注記がある場合のみ）。pkg 規約上、全 sub-package に README は存在
- `pkg/**/*.go` — 実装（`*.gen.go` / `*_test.go` 除く）
- `.claude/skills/arch-check-pkg/SKILL.md`

**書き込み（user 承認 + 理由明示後のみ）**:

- `pkg/README.md` + `pkg/<name>/README.md`
- `arch-check-pkg/SKILL.md`

**触らない**: 実装コード、他 layer。

## 最初のステップ: スコープ + 検出種別確認

`AskUserQuestion` 2 質問 batched:

1. 「back-prop-pkg のスコープを選んでください」 / 「変更ファイルのみ」 / 「pkg/ 全体」 / 「キャンセル」
2. 「検出する drift 種別を選んでください（multi-select、既定 3 種類すべて）」 / (A) / (B) / (C)

Chained: スキップ。

## Step 1. 入力読み込み

1. `pkg/README.md` + 全 sub-package READMEs (`pkg/<name>/README.md`) 読み込み
2. in-scope `.go` ファイル parse（特に import、Public API、ファイル構造）
3. `arch-check-pkg/SKILL.md` 読み込み

## Step 2. 検出 (A) README → Code drift

pkg 固有の典型 drift check:

- `pkg/*` ファイルから `internal/**` import なし（top-level 制約）
- framework import なし（echo / fx / gorm 等、sub-package README 明示許可なき限り）
- feature 固有 business logic なし（Policy）
- 全 sub-package に README が存在し、`Public API` セクション必須（`Wraps` は third-party ラップ時、`Notes` は注記がある場合）
- sub-package が top-level `pkg/README.md` Package List に列挙

各違反: `(rule, file:line, reasoning, 選択肢: コード修正 / README 緩和 / 例外)`。

## Step 3. 検出 (B) Code → README Undocumented Pattern

recurring pattern（≥3 ファイル / package）:

- 共通 helper signature pattern
- 共通 test pattern（table-driven、t.Parallel 配置）
- 3rd-party library wrap 規約

各 finding: `(pattern, 出現数, sample, reasoning, 選択肢: README 追記 / 無視 / refactor)`。

## Step 4. 検出 (C) Skill ↔ README Duplication

`arch-check-pkg/SKILL.md` 内 enumerate rule:

- 同 rule が `pkg/README.md` Constraints / Policy にあるか確認
- 重複: `(rule, skill 位置, README 位置, reasoning, 選択肢: skill slim / 両方維持)`

## Step 5. 集約レポート

```text
back-prop-pkg 結果（scope: <X>, 種別: A/B/C）

[A] README → Code drift  N 件
[B] undocumented pattern  M 件
[C] Skill duplication  K 件

総計 N+M+K 件。per-item 承認 / 棄却。
```

## Step 6. per-item user 判断

各 finding `AskUserQuestion`。doc / skill 変更承認 → AI **理由明示** + draft → user 最終確認 → 書き込み。

## Step 7. クロージング

```text
back-prop-pkg 完了。
  処理 finding: N
    README 更新: <X> / Skill 簡略化: <Y> / コード修正委任: <Z> / 無視: <W>
  最終 make md-lint OK
```

## AI 修正スコープ

書き込み: `pkg/README.md` + sub-package READMEs + `arch-check-pkg/SKILL.md`（承認 + 理由明示後）。
触らない: 実装コード、他 layer。

## 制約事項

- ❌ 実装コードへの書き込み
- ❌ user 承認なしの自動更新
- ❌ 理由を述べずに draft 実行
- ❌ scope + 種別 `AskUserQuestion` スキップ
- ❌ recurring threshold 3 未満を surface
- ✅ 日本語出力、各 finding に reasoning、per-item 承認制
- ✅ READMEs が canonical（pkg は layer + sub-package READMEs の二層構造）

## チェックリスト

- [ ] scope + 種別確認 / 受領
- [ ] layer + sub-package READMEs + 実装 + skill 読み込み
- [ ] (A)(B)(C) 検出
- [ ] 各 finding に reasoning 明示
- [ ] per-item 承認 → 理由提示 → draft → 最終確認 → 書き込み
- [ ] 実装コード触らない
- [ ] サマリ surface
