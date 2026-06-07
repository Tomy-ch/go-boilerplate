> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# Back-Prop — Domain

`internal/domain/README.md`（canonical）と domain 実装、domain 関連 skill body の drift を検出するスキル。

## 使うとき

- `internal/domain/` 編集後 `/commit` 前、drift 検出に
- 定期的な hygiene check（未文書化規約 / skill 肥大 / README drift を catch）
- domain 規約のリファクタ時
- `back-prop` 統合から chain

以下の用途には使いません:

- 実装コード修正 — surface のみ、修正は user
- 単一ファイルアーキ準拠 — `arch-check-domain`（TODO hand-off 付き）
- 新規 domain コード生成 — `scaffold-domain`

## 読み書き範囲

**読み込み（常時）**:

- `internal/domain/README.md` — canonical な convention 源（Implementation notes / Aggregate Design / Testing strategy / Do / Don't 等）
- `internal/domain/**/*.go` — 実装（`*.gen.go` / `_mock.go` / `*_test.go` 除く）
- `.claude/skills/arch-check-domain/SKILL.md` — rule enumeration を README と突き合わせ
- `.claude/skills/scaffold-domain/SKILL.md` — 生成規約 vs README
- `.claude/skills/new-spec-domain/SKILL.md` + `.claude/skills/verify-spec-domain/SKILL.md` — secondary

**書き込み（user 承認 + 理由明示後のみ、per-item）**:

- `internal/domain/README.md` — doc 更新承認時
- `.claude/skills/arch-check-domain/SKILL.md`（他 domain skill）— skill slim-down 承認時

**触らない**:

- 実装コード（`internal/domain/**/*.go`） — コード修正は user
- 他 layer の README / skill
- 生成物

## 最初のステップ: スコープ + 検出種別確認

`AskUserQuestion` を 2 質問 batched:

1. 質問: 「back-prop-domain のスコープを選んでください」
   - 選択肢: 「変更ファイルのみ (git diff)」 / 「internal/domain/ 全体」 / 「キャンセル」

2. 質問: 「検出する drift 種別を選んでください（multi-select、既定 3 種類すべて）」
   - 選択肢:
     - 「(A) README → Code drift」
     - 「(B) Code → README undocumented pattern」
     - 「(C) Skill ↔ README duplication」

`back-prop` 統合から chain 時は両方供与済み、質問スキップ。

## Step 1. 入力読み込み

1. `internal/domain/README.md` を全文読み:
   - 明示 rule（Implementation notes / Do/Don't / Invariants 等）
   - 各節内のコード例 / pattern 描写
2. in-scope `.go` ファイル列挙、top-level 構造 parse（struct field / method / import）
3. 各 domain skill SKILL.md 読み込み、enumerate された rule / 生成 pattern 抽出

## Step 2. 検出 (A) README → Code drift

`internal/domain/README.md` の明示 rule 各々:

- in-scope code で遵守確認
- finding: `(rule, 違反ファイル一覧, README 行番号)`
- threshold: 1+ ファイルで違反あれば surface

例:

```text
[A] README → Code drift
  rule: "全フィールドは unexport、getter 経由でのみ公開" (README L113)
  違反ファイル: internal/domain/foo/foo_domain.go (FirstName, LastName が export)
  reasoning: README が明示的に unexport を要求しているが、当該ファイルが export field を持つ
  user 判断:
    1. コード修正（field を unexport に + getter 追加）
    2. README 緩和（export を許可するケースを明記）
    3. 例外扱い（特殊事情、コード修正せず）
```

## Step 3. 検出 (B) Code → README Undocumented Pattern

in-scope code で recurring pattern を scan:

- threshold: pattern X が **3+ ファイル** で繰り返し → 「規約候補」
- 例: 全 aggregate が `xerrors.Wrap(apperror.ErrValidation, ...)` で error chain → README に記載なければ surface

各 finding:

- 検出 pattern（具体例）
- 出現ファイル数 + 代表ファイル名
- reasoning: 「N ファイルで同一 pattern。事実上の規約と推測、README 未記載」
- user 判断:
  1. README に追記（AI が draft + 理由提示後に書き込み）
  2. 偶然の重複として無視
  3. リファクタで消す（規約として確立せず、code 側削減）

## Step 4. 検出 (C) Skill ↔ README Duplication

`arch-check-domain/SKILL.md`（他 domain skill）の enumerate rule 各々:

- 同じ rule が `internal/domain/README.md` にも記載されているか確認
- 重複あれば「skill duplication 候補」として surface

例:

```text
[C] Skill ↔ README duplication
  rule: "entity フィールドは unexport"
  duplicated in:
    - arch-check-domain/SKILL.md L82
    - internal/domain/README.md L113 (Naming/Structure)
  reasoning: 同じルールが skill 内 enumerate + README で記述。skill は README 参照のみで slim 化可能
  user 判断:
    1. skill 内記述を削除、README 参照のみに簡略化（AI diff draft + 理由提示）
    2. skill 内記述を維持（README が薄い / skill 独自表現が価値ある場合）
```

## Step 5. 集約レポート

日本語、種別ごとにグルーピング:

```text
back-prop-domain 結果（scope: <X>, 種別: A/B/C）

[A] README → Code drift  N 件
  ...

[B] undocumented pattern  M 件
  ...

[C] Skill duplication  K 件
  ...

総計 N+M+K 件。
各 finding について選択肢を提示します。1 件ずつ承認 / 棄却。
```

## Step 6. per-item user 判断

各 finding について `AskUserQuestion`、finding 内の選択肢を提示。doc / skill 変更承認時:

1. AI が **理由を明示してから** draft 提示:

   ```text
   理由: <なぜこの変更が必要か>
   draft 内容 (diff 形式):
     <変更前 / 変更後>
   ```

2. user 最終確認後、書き込み (`Edit` / `Write`)

全 finding ループで処理、途中 abort 可。

## Step 7. クロージング

```text
back-prop-domain 完了。
  処理 finding: N 件
    README 更新: <X> 件
    Skill 簡略化: <Y> 件
    コード修正委任: <Z> 件（user 作業）
    無視 / 棄却: <W> 件
  README / Skill 書き込み: <count> 箇所
  最終 make md-lint OK
```

実装コードは触っていない（surface のみ）。コード修正は user 作業。

## AI 修正スコープ

- 書き込み: `internal/domain/README.md` + 関連 skill SKILL.md（user 承認 + 理由明示後のみ）
- 触らない: 実装コード（`internal/domain/**/*.go`）、他 layer README / skill、生成物

## 制約事項

- ❌ 実装コードへの書き込み（surface のみ、修正は user）
- ❌ user 承認なしの README / skill 自動更新
- ❌ 理由を述べずに draft 実行
- ❌ scope + 種別 `AskUserQuestion` をスキップ
- ❌ recurring threshold 3 未満を「規約候補」 surface（noise）
- ✅ 日本語出力
- ✅ 各 finding に reasoning 明示
- ✅ per-item 承認制
- ✅ README が canonical の前提を貫く

## チェックリスト

- [ ] scope + 種別を `AskUserQuestion` 確認 or 受領
- [ ] `internal/domain/README.md` 読み込み
- [ ] in-scope 実装ファイル parse
- [ ] 関連 skill SKILL.md 読み込み
- [ ] (A) drift / (B) undocumented / (C) duplication 検出（選択種別のみ）
- [ ] 各 finding に reasoning 明示
- [ ] per-item user 承認 → 理由明示 → draft → 最終確認 → 書き込み
- [ ] 実装コードへの書き込みなし
- [ ] 最終サマリで処理件数 + 書き込み箇所 surface
