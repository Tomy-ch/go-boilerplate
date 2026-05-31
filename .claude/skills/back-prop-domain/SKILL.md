---
name: back-prop-domain
description: Drift detection skill for the domain layer. Reads `internal/domain/README.md` (canonical) + `internal/domain/**/*.go` (implementation) + `arch-check-domain` / `scaffold-domain` / `new-spec-domain` / `verify-spec-domain` skill bodies, then surfaces three categories of drift: (A) README → Code drift — README rules that code violates; (B) Code → README undocumented pattern — recurring patterns (3+ files) that aren't in README; (C) Skill ↔ README duplication — rules duplicated in skill bodies that already live in README (candidates for skill slim-down). Each finding is reported with **explicit reasoning** + the user decides per-item (fix code / update README / relax rule / slim down skill / ignore). When the user approves a README or Skill change, AI drafts the diff with reasoning shown and writes only after final confirmation. Does NOT modify implementation code — code fixes are the user's responsibility (this skill surfaces, doesn't auto-resolve). Recommended trigger: after editing files in `internal/domain/`, before commit, to confirm doc + skill remain in sync with reality (and vice versa). Standalone-callable; chained from `back-prop` integrator skips scope question.
---

# Back-Prop — Domain

Detect drift between `internal/domain/README.md` (canonical), domain implementation, and the domain-related skill bodies.

A Japanese reference translation of this skill is available at `SKILL.ja.md` in the same directory (not loaded as a skill; for human reference only).

## When to Use

- After editing files in `internal/domain/` and before `/commit`, to detect drift.
- Periodically as a hygiene check (catch undocumented conventions / skill bloat / README drift).
- When refactoring domain conventions.
- Chained from the `back-prop` integrator.

Do NOT use for:

- Fixing implementation code — surface only, user resolves.
- Single-file architecture compliance — that's `arch-check-domain` (with TODO hand-off at each finding).
- Generating new domain code — that's `scaffold-domain`.

## What This Skill Reads / Writes

**Reads (always)**:

- `internal/domain/README.md` — canonical convention source (Implementation notes / Aggregate Design / Testing strategy / Do / Don't sections)
- `internal/domain/**/*.go` — implementation (excluding `*.gen.go`, `_mock.go`, `*_test.go`)
- `.claude/skills/arch-check-domain/SKILL.md` — rule enumeration to cross-check against README
- `.claude/skills/scaffold-domain/SKILL.md` — generation conventions vs README
- `.claude/skills/new-spec-domain/SKILL.md` + `.claude/skills/verify-spec-domain/SKILL.md` — secondary

**Writes (only with explicit per-item user approval + visible reasoning)**:

- `internal/domain/README.md` — when user approves a documentation update
- `.claude/skills/arch-check-domain/SKILL.md` (and other domain skills) — when user approves a skill slim-down

**Never touches**:

- Implementation code (`internal/domain/**/*.go`) — code fixes are user-owned
- Other layer READMEs / skills
- Generated files

## First Step: Confirm Scope + Detection Categories

`AskUserQuestion` with two questions (batched):

1. 質問: 「back-prop-domain のスコープを選んでください」
   - 選択肢: 「変更ファイルのみ (git diff)」 / 「internal/domain/ 全体」 / 「キャンセル」

2. 質問: 「検出する drift 種別を選んでください（multi-select、既定 3 種類すべて）」
   - 選択肢:
     - 「(A) README → Code drift」
     - 「(B) Code → README undocumented pattern」
     - 「(C) Skill ↔ README duplication」

When chained from `back-prop` integrator, both supplied; skip the question.

## Step 1. Load Inputs

1. Read `internal/domain/README.md` in full; extract:
   - Stated rules (Implementation notes, Do/Don't, Invariants, etc.)
   - Reference implementation patterns (from sections with code blocks at the bottom)
2. Enumerate in-scope `.go` files; for each, parse top-level structure (struct fields, methods, imports).
3. Read each domain-related skill SKILL.md; extract enumerated rules / generation patterns.

## Step 2. Detection (A) README → Code Drift

For each rule explicitly stated in `internal/domain/README.md`:

- Scan in-scope code for compliance
- Record findings as: `(rule, violating-files-list, README-line)`
- Threshold: surface only when ≥ 1 file violates

Example finding:

```text
[A] README → Code drift
  rule: "全フィールドは unexport、getter 経由でのみ公開" (README L113)
  violating files: internal/domain/foo/foo_domain.go (FirstName, LastName が export)
  reasoning: README が明示的に unexport を要求しているが、当該ファイルが export field を持つ
  user 判断:
    1. コード修正（field を unexport に + getter 追加）
    2. README 緩和（export を許可するケースを明記）
    3. 例外扱い（特殊事情、コード修正せず）
```

## Step 3. Detection (B) Code → README Undocumented Pattern

Scan in-scope code for recurring patterns:

- threshold: パターン X が **3+ ファイル** で繰り返されたら「規約候補」
- 例: 全 aggregate が `xerrors.Wrap(apperror.ErrValidation, ...)` で error chain → README に "全 domain error は apperror root に wrap" の記載が無ければ surface

Each finding includes:

- 検出された pattern（具体例）
- 出現ファイル数 + 代表ファイル名
- reasoning: 「N ファイルで同一 pattern が見られ、事実上の規約と推測される。README 未記載」
- user 判断:
  1. README に追記（AI が draft 案 + 理由を提示してから書き込み）
  2. 偶然の重複として無視
  3. リファクタで消す（規約として確立せず、コード側で減らす）

## Step 4. Detection (C) Skill ↔ README Duplication

For each rule enumerated in `arch-check-domain/SKILL.md` (and other domain skills):

- Check if the same rule (verbatim or paraphrase) appears in `internal/domain/README.md`
- If yes: surface as "skill duplication candidate"

Example finding:

```text
[C] Skill ↔ README duplication
  rule: "entity フィールドは unexport"
  duplicated in:
    - arch-check-domain/SKILL.md L82 (forbidden imports/exports check)
    - internal/domain/README.md L113 (Naming/Structure)
  reasoning: 同じルールが skill 内で enumerate + README で記述。skill は README を参照するだけにすれば slim 化可能
  user 判断:
    1. skill 内記述を削除、README 参照のみに簡略化（AI が diff を draft + 理由提示）
    2. skill 内記述を維持（README が薄い場合 / skill 独自の表現が価値ある場合）
```

## Step 5. Aggregated Report

Print Japanese report grouped by detection category:

```text
back-prop-domain 結果（scope: <X>, 種別: A/B/C）

[A] README → Code drift  N 件
  ...

[B] undocumented pattern  M 件
  ...

[C] Skill duplication  K 件
  ...

総計 N+M+K 件。
各 finding について、選択肢を提示します。1 件ずつ承認 / 棄却を判断してください。
```

## Step 6. Per-Item User Decision

For each finding, `AskUserQuestion` with the choices listed in the finding. If user approves a doc / skill change:

1. AI **明示的に理由を述べてから** draft を提示:

   ```text
   理由: <なぜこの変更が必要か>
   draft 内容（diff 形式）:
     <変更前 / 変更後>
   ```

2. user 最終確認後、書き込み（`Edit` / `Write`）

ループで全 finding を処理。途中で abort 可。

## Step 7. Closing

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

## AI Modification Scope

- 書き込み: `internal/domain/README.md` + 関連 skill SKILL.md（user 承認後、理由明示後のみ）
- 触らない: 実装コード（`internal/domain/**/*.go`）、他 layer の README / skill、生成物

## Constraints

- ❌ 実装コードへの書き込み（surface のみ、修正は user）
- ❌ user 承認なしの README / skill 自動更新
- ❌ 理由を述べずに draft を実行
- ❌ scope + 検出種別 `AskUserQuestion` をスキップ
- ❌ recurring threshold 3 未満の pattern を「規約候補」として surface（noise になる）
- ✅ Japanese output
- ✅ 各 finding に reasoning 明示
- ✅ per-item 承認制
- ✅ README が canonical の前提を貫く（drift detection 結果として「README が古い」と判定した時は緩和提案、ただし最終判断は user）

## Checklist

- [ ] Scope + 検出種別を `AskUserQuestion` で確認 or 受領
- [ ] `internal/domain/README.md` 読み込み
- [ ] in-scope 実装ファイル parse
- [ ] 関連 skill SKILL.md 読み込み
- [ ] (A) drift / (B) undocumented / (C) duplication 検出（選択された種別のみ）
- [ ] 各 finding に reasoning 明示
- [ ] per-item user 承認 → 理由明示 → draft → 最終確認 → 書き込み
- [ ] 実装コードへの書き込みなし
- [ ] 最終サマリで処理件数 + 書き込み箇所を surface
