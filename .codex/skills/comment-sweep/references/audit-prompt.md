# Comment Sweep Auditor Instructions

Audit existing comments in the assigned package only. You are strictly read-only: never write,
never ask the user, and return your final Japanese report as data for the orchestrator.

## Read at runtime

Read these before examining the assigned files. Apply them as written; `docs/rules.md` wins over this
instruction if there is a conflict.

1. *Comment Rules* in `docs/rules.md`, including Jurisdiction.
2. *What belongs here* in `docs/adr/README.md`.
3. `docs/design/README.md`.
4. The assigned package's nearest ancestor `README.md`.

Judge all resolved files, not only changed lines. For every prose comment, answer the jurisdiction
question: if the decision were reversed, which document would someone be obliged to update? Do not
replace it with “is this Why non-obvious?”

## Verdicts

Give exactly one verdict per finding.

- **維持**: Content belongs at this declaration, including an ordinary call-site constraint or a
  library/API-specific behavior that can change with dependency upgrades. Count only; do not list it
  individually, except for a contradiction with code.
- **短縮**: Content belongs here but exceeds the fact it delivers. Provide compressed wording; never
  propose deletion under this verdict.
- **削除**: Content carries nothing: a restatement, tautology, resolved TODO, or narration already
  evident from code. Quote the code that makes it redundant.
- **移設**: Content is meaningful but belongs in a governing document. Supply every landing-form item
  below.

Report a comment that contradicts code first and individually regardless of verdict. A false doc
comment outranks jurisdiction.

For an exported Go declaration, `revive exported` requires a leading-identifier doc comment such as
`// Foo は …`. `削除` is unavailable: mark it explicitly and propose only `短縮` or `移設` whose residue
retains that form.

## Complete relocation landing form

Every `移設` finding must include all of the following:

1. A concrete destination file and section. Read the destination before proposing an addition. If it
   already states the content, report `追記なし`: there is nothing to relocate, so land the finding
   as **短縮** to the residue plus a link. Duplicating into the destination corrupts the document
   this skill is meant to keep authoritative. For an ADR, name an existing candidate by number and
   title when applicable; otherwise state that none exists, never invent a number.
2. The exact Japanese prose to add, in that document's voice; write `追記なし` when the destination
   already states the content.
3. The exact code residue: one or two operative sentences and its link.
4. A residue test confirming that the residue makes sense without following the link.

Relocation is not dumping. Refuse an ADR for a library/API property: it remains `維持` at the call
site. Business/domain knowledge belongs in `docs/spec/**`, never an ADR. Send an ADR only a lasting
choice among alternatives or a deliberate exclusion. If no destination admits the content, that is
evidence for `維持`.

## Exclude entirely

Do not flag or side-note any of the following:

- One-line `// Name は、〜です。` field comments; this repository deliberately preserves their visual
  uniformity.
- Generated files, mocks, and tests (`*.gen.go`, `*.sql.go`, `*_mock.go`, `*_test.go`).
- Functional/directive comments: `go:generate`, `nolint`, `go:build`, `go:embed`, generated-code
  markers, shebangs, and SQL/YAML tool directives.
- Unresolved TODO/FIXME markers.
- **Package overviews** — a `// Package …` comment, **wherever it lives**. This repository has no
  `doc.go` at all: every package overview sits at the top of an ordinary source file, so matching on
  the filename excludes nothing and flags every overview in the repo. Usage and How belong in an
  overview; `docs/rules.md` exempts them.
- Existing documentation prose quality; this audit may propose an addition but never audits docs.

## Output in Japanese

Report only code-backed evidence. If the package has no action, say so plainly. Use this exact shape;
the outer four-backtick fence intentionally permits the inner fenced diffs.

````text
## comment-sweep 監査結果: <パッケージパス>

対象 <n> ファイル / 判定内訳: 維持 <a> / 短縮 <b> / 削除 <c> / 移設 <d>

### [判定] <短いタイトル>
- 場所: <path:line>
- 対象コメント: `<コメント全文>`
- 判定: 維持 / 短縮 / 削除 / 移設
- 根拠: <コード根拠と、移設なら更新義務のある文書>
- 移設先: <具体的なファイルと節>
  - 追記する文面: <実際の文面>
- 着地形（変更前 → 変更後）:
  ```go
  // 変更前の全文
  ```

  ```go
  // 変更後の残滓とリンク
  ```
- 残滓テスト: <リンクなしでも成立する確認>
- ※ export 宣言: 削除不可（該当時のみ）
- 確度: high / medium / low
````

`維持` はヘッダーの件数だけに含め、個別ブロックに出さない。ただし、コメントがコードと矛盾する場合は、
判定に関わらず必ず個別に報告する。
