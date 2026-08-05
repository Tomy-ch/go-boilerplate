---
name: comment-sweep
description: Sweep the existing stock of source-code comments to decide whether their content belongs at the declaration or in its governing document, preserving operative code residue and links when relocating rationale. Use for bloated or essay-like comments, doc comments grown into design arguments, periodic hygiene of a package/layer/repository, and before a fork or template cut; Japanese triggers include 「コメントが長すぎる」「コメントを整理して」「この Why はコードに置くべきか」「コメントを ADR に移したい」. Do not use for comments on a change just written, which `impl-review` / `comment-reviewer` own as diff scope; README or docs prose quality, which `doc-reviewer` owns; or README-to-code structural drift, which `back-prop` / `sync-readme` own.
---

# Comment Sweep

Treat this as a relocation workflow, not a comment-deletion workflow. Do not re-argue comment
volume: a non-obvious, verifiable Why has a concrete case for remaining. Instead, ask the
jurisdiction question from the runtime authority: if this decision were reversed, which document
would require an update? If none would, the code is the jurisdiction and the full comment remains.

All user-visible output, including findings, choices, proposed prose, and summaries, must be in
Japanese.

## Runtime authorities

Before inspecting the selected scope, read each source below. Do not hardcode or substitute a
remembered policy. `docs/rules.md` wins if it conflicts with this skill.

| Question | Runtime source |
| --- | --- |
| Allowed comment content and jurisdiction | *Comment Rules* in `docs/rules.md` |
| Destination type | *What belongs here* in `docs/adr/README.md` |
| Subsystem design references | `docs/design/README.md` |
| Package scope | The package's nearest ancestor `README.md` |

Read [`references/audit-prompt.md`](references/audit-prompt.md) before auditing or dispatching an
auditor. It is the single shared instruction file for all auditors.

## 1. Confirm the scope

Before reading target files, present these numbered choices in Japanese and wait for the user's
choice:

1. 指定パス配下（パッケージまたはディレクトリ）
2. ベースブランチとの差分
3. レイヤ全体（`internal/domain`、`internal/usecase`、`internal/controller`、`internal/infrastructure`、`pkg`）
4. キャンセル

Stock scope is this skill's purpose. If the user selects diff scope, state in one Japanese line
that `comment-reviewer` is normally the better diff-scoped tool, then continue. On cancel, stop
without reading the target scope.

## 2. Resolve and rank targets

Resolve all source files under the selected scope. Exclude `*.gen.go`, `*.sql.go`, `*_mock.go`, and
`*_test.go`. Include non-Go sources such as shell, Dockerfile, Makefile, SQL, YAML, and `.mjs` when
they are in scope; their prose follows the same standard and is not covered by `revive`.

Count comment lines, rank files from highest volume to lowest, and show that ranking in Japanese.
Group files by package directory. A package is the audit unit because its nearest README can be the
jurisdictional destination.

## 3. Audit read-only

When the runtime supports parallel inspection, audit independent package groups concurrently;
otherwise audit them sequentially. Each auditor must:

- Be strictly read-only: surface evidence, verdicts, and landing forms only; never ask the user or write.
- Read and follow `references/audit-prompt.md` verbatim rather than a paraphrased spawn prompt.
- Read the runtime authorities and the nearest README for its package.

The orchestrator alone obtains approval and performs every write, single-threaded. This prevents
parallel auditors from contending over one destination document.

## 4. Aggregate before decisions

Show the entire audit surface before proposing a write. Group by package and show `維持 / 短縮 /
削除 / 移設` counts. Show individual findings only for contradictions and non-`維持` verdicts, with
location, verbatim comment, reasoning, and the complete landing form. Show destination totals and a
grand total. `維持` is a count only, except a code contradiction is always an individual finding.

If no action is needed, say so in Japanese and stop.

## 5. Approve and apply one finding at a time

Process non-`維持` findings in descending impact order. For each one, present the evidence and a
before/after comment diff. For `移設`, also present the exact destination prose. Ask for approval for
that one finding; never batch-apply close calls.

After approval, write the destination document first and the code second. This order is mandatory:
an interruption must never leave the rationale without its new home.
When the auditor reported `追記なし`, do not write the destination document: land the finding as a
`短縮` and change only the code.

For an ADR destination, do not choose its record shape. Ask the user to choose one of: rewrite the
existing ADR-NNNN, create a new ADR, use `docs/design` or a README instead, or do not relocate now.
If approved, update the English canonical document, its `docs/ja/` mirror, and the English/Japanese
ADR log tables together. If a package README addition materially changes its claims, mention
`back-prop` as the appropriate follow-up.

## 6. Verify

Run `make fix`, then `make lint` over touched packages. Run `make md-lint` whenever a Markdown
destination was written. Finally, reread every edited comment against the residue test: it must stand
alone for a reader who does not follow its link. Report the result in Japanese; do not stage, commit,
or push.

## Reviewer boundary

| Tool | Scope | Verdicts | Owns |
| --- | --- | --- | --- |
| `comment-reviewer` via `impl-review` | diff | 削除 / 書換 / 加筆 | Generation-time inflow gate |
| `doc-reviewer` | `README*` / `docs/**` | Content findings | Docs prose quality |
| `comment-sweep` | stock | 維持 / 短縮 / 削除 / 移設 | Content jurisdiction |
