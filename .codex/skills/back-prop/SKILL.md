---
name: back-prop
description: Detect and reconcile drift between implementation, canonical README files, repository skills, and the DDD pattern ledger. Use after a multi-layer refactor, before a major review, or for documentation hygiene. Audit changed files by default, classify README-to-code drift, undocumented recurring code patterns, skill-to-README duplication, and DDD-ledger-to-ADR/README-corpus drift, then propose each documentation change for approval. Never change implementation code with this skill.
---

# Documentation Drift Review

The authority order is **README > code > skill**. This skill discovers drift; it does not use implementation behavior to silently overwrite a documented decision.

## Scope and categories

Default to changed production Go files against the PR base or default branch. Allow a requested full scan or named layers. Exclude generated files, mocks, and tests.

Audit only the requested categories; default to all four:

- **A — README → code:** implementation violates or no longer matches a documented convention.
- **B — code → README:** a recurring, intentional-looking implementation pattern lacks documentation. Require evidence from at least three independent occurrences before reporting it.
- **C — skill ↔ README:** a skill duplicates stale rules or contradicts the canonical README.
- **D — DDD ledger ↔ ADR/README corpus:** the DDD pattern ledger's pointers or stated coverage no
  longer match the canonical corpus. This is bookkeeping drift, not an assessment of fidelity to
  Evans; `ddd-audit` owns that external comparison.

Map Go files to `domain`, `usecase`, `controller`, `infrastructure`, or `pkg`. For category D, read
the `corpus` globs from `.agents/ddd-audit/pattern-ledger.yaml` at runtime (never hardcode them) and
intersect them with the changed files; inspect the complete corpus for a full scan. Run
`.codex/agents/drift-detector-ddd.toml` whenever D has an in-scope corpus, alongside layer detectors.
Report changed files outside these layers and the DDD corpus as not audited. If no in-scope files
exist, stop cleanly.

## Read-only detection

1. Read `AGENTS.md`, then the relevant layer README and nearest subpackage README. Read the matching Codex skill only for category C. For D, read the ledger and only its runtime-resolved corpus.
2. Inspect the scoped code and collect concrete file-and-line evidence. Use sibling code as supporting evidence, never as a substitute for an explicit rule.
3. Compare each finding against the authority order:

   - For A, recommend a code correction or an explicit documented exception; do not edit code.
   - For B, offer a README addition only when the three-occurrence threshold and intent evidence are met. Otherwise report nothing.
   - For C, remove duplicated procedural rules from the skill or update it to point at the README; do not copy README rules into the skill.
   - For D, surface only D1 pointer rot, D2 semantic rot, or D3 uncaptured interpretation. Do not
     rewrite an ADR or README to resolve it; the ledger is the only possible approved write target.
4. When the runtime supports parallel inspection, inspect independent layers and the corpus-driven
   DDD detector concurrently. Otherwise inspect sequentially. Detection is always read-only.

## Report before writes

Return a Japanese, layer-grouped report before proposing edits:

```text
back-prop 検出結果（スコープ: <scope>、種別: <A/B/C/D>）

[<layer>]
- A|B|C: <file:line or README section>
  判断: <why this is drift>
  根拠: <canonical README / repeated code evidence>
  選択肢: <code fix | README update | skill simplification | documented exception | ignore>

[ddd]
- D1|D2|D3: <ledger entry and corpus file:line>
  判断: <why the ledger and corpus drift>
  根拠: <runtime-resolved canonical corpus>
  選択肢: <ledger update | ddd-audit follow-up | ignore>

総件数: <n>
```

Do not write anything until the user selects a resolution for each finding. If the report has no findings, say which layers and categories were checked.

## Applying approved documentation changes

Only apply a README or Codex-skill change after the user explicitly approves that individual finding. Before writing, present the intended diff and explain why it preserves the authority order.

- Never edit implementation code, generated files, or `AGENTS.md`.
- Keep edits limited to the affected canonical English README or `.codex/skills/<name>/SKILL.md`;
  for an individually approved D finding, `.agents/ddd-audit/pattern-ledger.yaml` is also allowed.
- Never resolve a D finding by rewriting an ADR or README. Surface that desired decision-record
  change as follow-up for the user.
- For a proposed code correction, report the task instead of implementing it.
- After documentation edits, run `make md-lint`; if needed, run `make md-fix` only on the approved Markdown changes, then rerun the lint.

## Completion

Report each finding as approved, rejected, deferred to code work, or ignored. State all files changed and the Markdown validation result. Do not stage, commit, or push.
