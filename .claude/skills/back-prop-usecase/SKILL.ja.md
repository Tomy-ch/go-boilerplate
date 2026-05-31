> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# Back-Prop — Usecase

usecase README + Implementation Example、usecase 実装、usecase 関連 skill body の drift を検出。

## 使うとき

- `internal/usecase/` 編集後 commit 前に drift 検出
- 定期 hygiene check
- usecase 規約のリファクタ時
- `back-prop` 統合から chain

以下の用途には使いません: 実装コード修正（surface のみ） / 単一ファイルアーキ準拠（`arch-check-usecase`） / 新規 usecase 生成（`scaffold-usecase`）。

## 読み書き範囲

**読み込み**:

- `internal/usecase/README.md` — canonical（末尾 Implementation Example 含む）
- `internal/usecase/boundary/README.md` — boundary 規約
- `internal/usecase/**/*.go` — 実装（`*.gen.go` / `_mock.go` / `*_test.go` 除く）
- `.claude/skills/arch-check-usecase/SKILL.md` — rule enumerate を README と突き合わせ
- `.claude/skills/scaffold-usecase/SKILL.md` — 生成規約 vs README
- `.claude/skills/new-spec-usecase/SKILL.md` + `.claude/skills/verify-spec-usecase/SKILL.md` — secondary

**書き込み（user 承認 + 理由明示後のみ）**:

- `internal/usecase/README.md` + `boundary/README.md`
- usecase 関連 skill SKILL.md

**触らない**: 実装コード、他 layer、生成物。

## 最初のステップ: スコープ + 検出種別確認

`AskUserQuestion` 2 質問 batched:

1. 「back-prop-usecase のスコープを選んでください」 / 「変更ファイルのみ」 / 「internal/usecase/ 全体」 / 「キャンセル」
2. 「検出する drift 種別を選んでください（multi-select、既定 3 種類すべて）」 / 「(A) README → Code drift」 / 「(B) Code → README undocumented pattern」 / 「(C) Skill ↔ README duplication」

`back-prop` 統合から chain 時はスキップ。

## Step 1. 入力読み込み

1. `internal/usecase/README.md`（rules + Implementation Example）+ `boundary/README.md` 読み込み
2. in-scope `.go` ファイル parse（struct field / method / import / tracer span / boundary 呼び出し / DTO 規約）
3. usecase 関連 skill SKILL.md 読み込み

## Step 2. 検出 (A) README → Code drift

usecase 固有の典型 drift check:

- `usecase` struct 名（README "fixed"） → 全 package で使用確認
- 全 public method 冒頭で `tracer.Start(ctx); defer endSpan()`（Observability 節）
- `time.Now()` 直接禁止、`math/rand` 直接禁止（Time Handling / Boundary Concept）
- `internal/infrastructure/**` import 禁止（Forbidden dependencies）
- HTTP/echo import 禁止
- 複数書き込み workflow は `tx.Manager.Do(...)` で tx 境界

各違反: `(rule, file:line, reasoning, user 選択肢: コード修正 / README 緩和 / 例外)`。

## Step 3. 検出 (B) Code → README Undocumented Pattern

recurring pattern scan（≥3 ファイル）:

- DTO 命名規約（`<X>MutableFields` / `<X>ParamsDTO` 等）
- 特定 error wrap pattern
- private method / helper pattern
- constructor 引数順

各 finding: `(pattern, 出現数, sample ファイル, reasoning, user 選択肢: README 追記 / 無視 / refactor)`。

## Step 4. 検出 (C) Skill ↔ README Duplication

`arch-check-usecase/SKILL.md`（他 usecase skill）の enumerate rule:

- 同 rule が `internal/usecase/README.md` にあるか確認
- 重複あれば `(rule, skill 位置, README 位置, reasoning, user 選択肢: skill slim / 両方維持)`

## Step 5. 集約レポート

```text
back-prop-usecase 結果（scope: <X>, 種別: A/B/C）

[A] README → Code drift  N 件
[B] undocumented pattern  M 件
[C] Skill duplication  K 件

総計 N+M+K 件。per-item 承認 / 棄却。
```

## Step 6. per-item user 判断

各 finding `AskUserQuestion`、選択肢から判断。doc / skill 変更承認 → AI が **理由明示** + draft 提示 → user 最終確認 → 書き込み。

## Step 7. クロージング

```text
back-prop-usecase 完了。
  処理 finding: N
    README 更新: <X> 件
    Skill 簡略化: <Y> 件
    コード修正委任: <Z> 件
    無視 / 棄却: <W> 件
  最終 make md-lint OK
```

## AI 修正スコープ

書き込み: `internal/usecase/README.md`、`boundary/README.md`、usecase 関連 skill（承認 + 理由明示後）。
触らない: 実装コード、他 layer、生成物。

## 制約事項

- ❌ 実装コードへの書き込み
- ❌ user 承認なしの自動更新
- ❌ 理由を述べずに draft 実行
- ❌ scope + 種別 `AskUserQuestion` スキップ
- ❌ recurring threshold 3 未満を surface
- ✅ 日本語出力、各 finding に reasoning、per-item 承認制
- ✅ README が canonical の前提を貫く

## チェックリスト

- [ ] scope + 種別確認 / 受領
- [ ] README + boundary README + 実装 + skill 読み込み
- [ ] (A)(B)(C) 検出（選択種別のみ）
- [ ] 各 finding に reasoning 明示
- [ ] per-item 承認 → 理由提示 → draft → 最終確認 → 書き込み
- [ ] 実装コード触らない
- [ ] サマリ surface
