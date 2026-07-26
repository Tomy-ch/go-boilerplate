---
name: back-prop
description: Detect and reconcile drift between implementation, canonical README files, and repository skills. Use after a multi-layer refactor, before a major review, or for documentation hygiene. Audit changed files by default, classify README-to-code drift, undocumented recurring code patterns, and skill-to-README duplication, then propose each documentation change for approval. Never change implementation code with this skill.
---

# Documentation Drift Review

The authority order is **README > code > skill**. This skill discovers drift; it does not use implementation behavior to silently overwrite a documented decision.

## Scope and categories

Default to changed production Go files against the PR base or default branch. Allow a requested full scan or named layers. Exclude generated files, mocks, and tests.

Audit only the requested categories; default to all three:

- **A — README → code:** implementation violates or no longer matches a documented convention.
- **B — code → README:** a recurring, intentional-looking implementation pattern lacks documentation. Require evidence from at least three independent occurrences before reporting it.
- **C — skill ↔ README:** a skill duplicates stale rules or contradicts the canonical README.

Map files to `domain`, `usecase`, `controller`, `infrastructure`, or `pkg`. Report changed files outside these layers as not audited. If no in-scope files exist, stop cleanly.

## Read-only detection

1. Read `AGENTS.md`, then the relevant layer README and nearest subpackage README. Read the matching Codex skill only for category C.
2. Inspect the scoped code and collect concrete file-and-line evidence. Use sibling code as supporting evidence, never as a substitute for an explicit rule.
3. Compare each finding against the authority order:

   - For A, recommend a code correction or an explicit documented exception; do not edit code.
   - For B, offer a README addition only when the three-occurrence threshold and intent evidence are met. Otherwise report nothing.
   - For C, remove duplicated procedural rules from the skill or update it to point at the README; do not copy README rules into the skill.
4. When the runtime supports parallel inspection, inspect independent layers concurrently. Otherwise inspect sequentially. Detection is always read-only.

## Report before writes

Return a Japanese, layer-grouped report before proposing edits:

```text
back-prop 検出結果（スコープ: <scope>、種別: <A/B/C>）

[<layer>]
- A|B|C: <file:line or README section>
  判断: <why this is drift>
  根拠: <canonical README / repeated code evidence>
  選択肢: <code fix | README update | skill simplification | documented exception | ignore>

総件数: <n>
```

Do not write anything until the user selects a resolution for each finding. If the report has no findings, say which layers and categories were checked.

## Applying approved documentation changes

Only apply a README or Codex-skill change after the user explicitly approves that individual finding. Before writing, present the intended diff and explain why it preserves the authority order.

- Never edit implementation code, generated files, or `AGENTS.md`.
- Keep edits limited to the affected canonical English README or `.codex/skills/<name>/SKILL.md`.
- For a proposed code correction, report the task instead of implementing it.
- After documentation edits, run `make md-lint`; if needed, run `make md-fix` only on the approved Markdown changes, then rerun the lint.

## Completion

Report each finding as approved, rejected, deferred to code work, or ignored. State all files changed and the Markdown validation result. Do not stage, commit, or push.
