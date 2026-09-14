---
name: settle-comments
description: >-
  Settle source-code comments by deciding whether a mechanism already guards their facts (不要), their content belongs at the declaration or in a governing document (移設), or repeated content should be consolidated at one owning declaration (集約). Use unconditionally as the last step of an implementation over touched declarations, and for bloated or essay-like comments, repeated or fragmented Why across files or packages, doc comments grown into design arguments, periodic hygiene of a package/layer/repository, or a template cut; supports 確認して適用 (default), 自動適用 (`--apply`), and 報告のみ (`--report-only`). Japanese triggers include 「コメントが長すぎる」「コメントを整理して」「テストと同じことをコメントが書いている」「この Why はコードに置くべきか」「コメントを ADR に移したい」. Do not use for README or docs prose quality, which `doc-reviewer` owns, or README-to-code structural drift, which `back-prop` / `sync-readme` own.
---

# Settle Comments

A Japanese reference translation of this skill is available at `SKILL.ja.md` in the same directory
(not loaded as a skill; for human reference only).

Treat this as a guard, relocation, and consolidation workflow, not a comment-deletion workflow.
Before jurisdiction, ask: **if this sentence turned false, would anything fail?** If a compile error,
test, lint or architecture rule, or regeneration check already keeps it true, the comment is an
unchecked second copy: verdict `不要`. If no mechanism exists, ask whether one can be built instead.
A comment's proper subject is what no mechanism can check.

This first question is bounded by the canonical direction declared in `docs/rules.md`: a doc comment
publishes the caller-facing contract. A test or `docs/spec/**` section repeating observable behavior
(what a declaration does, input/output/error meaning, or observable condition order) does not make
the comment the copy. Reasons not observable at the call site belong to `docs/spec/**`. For anything
that survives the first question, ask which document must change if the decision reverses. If none,
the declaration is the jurisdiction and the comment remains.

All user-visible output, including findings, choices, proposed prose, and summaries, must be in
Japanese.

## Runtime authorities

Before inspecting the selected scope, read each source below. Do not hardcode or substitute a
remembered policy. `docs/rules.md` wins if it conflicts with this skill.

| Question | Runtime source |
| --- | --- |
| Guard question, canonical direction, allowed content, and jurisdiction | *Comment Rules* in `docs/rules.md` |
| Destination type | *What belongs here* in `docs/adr/README.md` |
| Subsystem design references | `docs/design/README.md` |
| Package scope | The package's nearest ancestor `README.md` |

Read [`references/audit-prompt.md`](references/audit-prompt.md) before auditing or dispatching an
auditor. It is the single shared instruction file for all auditors.

## 1. Confirm the scope and apply mode

Before reading target files, present two numbered questions in Japanese and wait for the user's
choices. Skip a question when a flag or caller already fixed it; skip both when both are fixed.

Scope:

1. 指定パス配下（パッケージまたはディレクトリ）
2. ベースブランチとの差分
3. レイヤ全体（`internal/domain`、`internal/usecase`、`internal/controller`、`internal/infrastructure`、`pkg`）
4. キャンセル

Apply mode:

1. 確認して適用（既定。1 件ずつ確認してから書き換える）
2. 自動適用（確認なしで適用可能な 短縮 / 削除 / verified 不要 / high-confidence 集約を書き換える）
3. 報告のみ（書き込まず、全 findings を報告する）

`--apply` fixes the mode to 自動適用. `--report-only` fixes it to 報告のみ. If both flags are
present, state in Japanese that they contradict each other and fall back to asking the mode question;
do not choose precedence silently.

Stock scope is this skill's purpose. If the user selects diff scope, state in one Japanese line
that `comment-reviewer` is normally the better diff-scoped tool and that this sweep judges the whole
comment stock of the files touched, not changed lines alone, then continue. On cancel, stop without
reading the target scope.

## 2. Resolve and rank targets

Resolve all source files under the selected scope. Exclude `*.gen.go`, `*.sql.go`, `*_mock.go`, and
`*_test.go`. Include non-Go sources such as shell, Dockerfile, Makefile, SQL, YAML, and `.mjs` when
they are in scope; their prose follows the same standard and is not covered by `revive`.

Count comment lines, rank files from highest volume to lowest, and show that ranking in Japanese.
Group files by package directory. A package is the audit unit because its nearest README can be the
jurisdictional destination.

## 3. Audit read-only in three passes

When the runtime supports parallel inspection, spawn one `comment-reviewer` per independent package
group using Codex's delegation surface (`agentType: "comment-reviewer"`) and run them concurrently;
otherwise audit them sequentially inline. Each auditor must:

- Be strictly read-only: surface evidence, verdicts, and landing forms only; never ask the user or write.
- Read and follow `references/audit-prompt.md` verbatim rather than a paraphrased dispatch prompt.
  State in the dispatch that this reference overrides the agent's own diff-scoped verdict vocabulary
  and output format; its criteria for what makes a comment bad still apply.
- Read the runtime authorities and the nearest README for its package.
- Run Pass 0 and Pass 1 over every individual comment, then Pass 2 over the whole assigned package's
  comments as one body across all files.

Pass 0 asks whether a mechanism already guards each fact and requires its `file:line` for `不要`.
Pass 1 asks the jurisdiction question. Pass 2 asks which single declaration owns a Why written at
several sites and detects three shapes: 重複, 分散, and 総量過多. Pass 2 returns the set-valued 集約
verdict defined in the audit prompt.

The orchestrator alone obtains approval and performs every write, single-threaded. This prevents
parallel auditors from contending over one destination document.

## 4. Scan mechanically for repetition across packages

After dispatching the package audits and before aggregating their results, run this scan over every
resolved file. Only the orchestrator sees the full file set, while auditors report 維持 as a count,
so repeated content cannot be compared from their reports.

```sh
for f in <resolved files>; do
  grep -hE '^[[:space:]]*(//|#|--)' "$f" \
    | sed -E 's@^[[:space:]]*(//|#|--)[[:space:]]?@@' \
    | awk -v f="$f" 'length($0) > 30 { print f "\t" $0 }'
done | sort -t$'\t' -k2 \
  | awk -F'\t' '{ n[$2]++; src[$2] = src[$2] "\n    " $1 }
                 END { for (k in n) if (n[k] > 1) print "[" n[k] "] " k src[k] }'
```

The `grep` is load-bearing. Without it, the pipeline clusters code and blank lines and reports one
enormous meaningless cluster; do not simplify it away. The `length($0) > 30` floor is tunable for
the selected scope.

Resolve every cluster by jurisdiction first:

- When the repeated content belongs in a document, return 移設 at every site, or 短縮 when that
  document already states it. No declaration owns a concept spanning packages.
- Only when one declaration genuinely owns the concept, return a 集約 whose members span files. Each
  pointer must name the owning declaration because readers in another package cannot find it by
  proximity.

Hand clusters confined to one package to that package's auditor. The orchestrator carries clusters
that cross packages. A cluster is a finding even when every member is individually correct.

## 5. Aggregate before decisions

Show the entire audit surface before proposing a write. Group by package and show `維持 / 短縮 /
削除 / 不要 / 移設 / 集約` counts. Show individual findings only for contradictions and non-`維持` verdicts,
with location, verbatim comment, reasoning, and the complete landing form. Show destination totals
and these aggregate lines exactly:

```text
[<package>] 維持 <a> / 短縮 <b> / 削除 <c> / 不要 <g> / 移設 <d> / 集約 <e>
集約: <e> 件（対象コメント計 <t> 箇所 / 内訳 重複 <u> ・分散 <v> ・総量過多 <w>）
パッケージ横断の重複: <x> クラスタ（対象コメント計 <y> 箇所 / <z> パッケージにまたがる）
```

Count a 集約 once, not once per member, and report its member count beside it. `維持` is a count only,
except a code contradiction is always an individual finding. If no action is needed, say so in
Japanese and stop.

In 報告のみ mode, render every non-`維持` finding in full, including every 集約 member and every
移設 finding's proposed destination prose, then stop without writing. State that rerunning in
自動適用 can apply eligible 短縮 / 削除 / 不要 / 集約 findings, while 確認して適用 can also decide 移設.

## 6. Apply according to the selected mode

Do not enter this section in 報告のみ mode.

### 確認して適用

Process non-`維持` findings in descending impact order. For each one, present the evidence and a
before/after comment diff. For `移設`, also present the exact destination prose. Present numbered
choices in Japanese and wait for the user's choice for that one finding; never batch-apply close
calls.

For `不要`, show the named guard and what it says. Offer `不要`, `維持（現状のまま）`, and `判断を保留`;
the user cannot judge whether a second copy exists from the comment alone.

A 集約 is one indivisible question for its entire set, never one question per member. Show every
member and offer numbered choices for 集約, 維持（現状のまま）, 別の site を本体にする, and
判断を保留. Splitting approval could discard the Why or leave duplication unchanged.

After approval, write the destination document first and the code second. This order is mandatory:
an interruption must never leave the rationale without its new home. When the auditor reported
`追記なし`, do not write the destination document: land the finding as a `短縮` and change only the
code.

For an ADR destination, do not choose its record shape. Present numbered choices in Japanese for:
rewrite the existing ADR-NNNN, create a new ADR, use `docs/design` or a README instead, or do not
relocate now. If approved, update the English canonical document, its `.ja.md` translation, and the
English/Japanese ADR log tables together. If a package README addition materially changes its claims,
mention `back-prop` as the appropriate follow-up.

### 自動適用

Write eligible 短縮, 削除, 不要, and 集約 findings without per-item questions. Exclude and report instead:

- Any finding whose comment contradicts the code.
- A `不要` until the orchestrator opens its named `file:line` and confirms that it actually guards
  the fact. If the source does not carry the fact, report the finding instead; without a source it is
  not a valid `不要` finding at all.
- A `追記なし` 移設 until the orchestrator opens the destination and confirms the content is there;
  if confirmed, apply it as a code-only 短縮, otherwise report it.
- Any 集約 below `確度: high`.
- Any 集約 whose members span files.
- Any 移設 that would write to a destination document. ADR immutability makes new-record-versus-
  rewrite a repository-policy decision that a no-question mode cannot make.

A same-file, high-confidence 集約 writes no document and is eligible. Automatic application removes
the question, not any other guard: exported Go declarations retain valid doc comments, directives
remain untouched, and excluded files remain out of scope.

## 7. Verify

Run this section only when something was written. Run `make go-fix`, then `make go-lint` over touched
packages. Run `make md-lint` whenever a Markdown destination was written. Finally, reread every
edited comment against the residue test: it must stand alone for a reader who does not follow its
link.

After a 集約, read every file it touched top to bottom rather than checking edited sites alone. Verify
that the owning site carries every distinct fact and every pointer is navigable. When members spanned
packages, read the pointer from the other package's side.

Report the result in Japanese; do not stage, commit, or push.

## Reviewer boundary

| Tool | Scope | Verdicts | Owns |
| --- | --- | --- | --- |
| `doc-reviewer` | `README*` / `docs/**` | Content findings | Docs prose quality |
| `settle-comments` | touched declarations or stock | 維持 / 短縮 / 削除 / 不要 / 移設 / 集約 | Guard ownership, content jurisdiction, and consolidation |

No review skill carries a comment lens. This skill owns both comments added by a change and the
existing comments on the declarations that change touched; it judges them together as one stock.
