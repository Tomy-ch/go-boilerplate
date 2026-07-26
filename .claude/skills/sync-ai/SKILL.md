---
name: sync-ai
description: >-
  Reconcile a changed skill between this repository's Claude Code (`.claude/skills/`) and Codex (`.codex/skills/`) environments without blindly copying platform-specific instructions. Use after creating or materially updating a skill and when asked to synchronize, migrate, port, or compare a skill across Claude and Codex. Select one authoritative source direction, inspect the changed skill unit and its counterpart, then have the receiving environment's native `manage-skill` workflow adapt the intent, resources, and trigger description. Do not use for ordinary documentation translation, implementation code synchronization, or automatic two-way merges.
argument-hint: '[skill-name] [--from=claude|codex] [--to=claude|codex]'
allowed-tools: Bash, Read, Write, Edit, Glob, Grep, AskUserQuestion, Skill
---

# Sync AI Skills (Claude)

Synchronize a skill as a **one-way semantic port**, never as a raw directory copy. The source skill
is authoritative only for this run; the receiving skill remains native to its AI environment.

`manage-skill` invokes this after every new or material Claude skill change unless the skill is
explicitly Claude-only. For a Codex-originated change, use the same flow with Codex as the source.

## 1. Resolve direction and unit

Require exactly one skill name and one direction:

```text
source: .claude/skills/<name>/  → target: .codex/skills/<name>/
source: .codex/skills/<name>/   → target: .claude/skills/<name>/
```

Infer the source only when the user clearly names it. Do not infer direction from file mtimes.
If both copies changed, stop and ask which copy is authoritative; never merge two edited skills
automatically.

Inspect the complete source unit and the target unit if it exists. Treat `SKILL.md`,
`SKILL.ja.md`, UI metadata, scripts, references, and assets as one unit. Record only a temporary
transfer note under `tmp/sync-ai/`; do not create a durable third copy or a synchronization manifest.

## 2. Build the transfer contract

Extract purpose, trigger and non-trigger cases, inputs, approvals, side effects, validation, and
reusable resources. Classify each item as **port**, **adapt**, or **omit**. Translate platform
mechanisms instead of copying them: Codex delegation and tool instructions must not become Claude
syntax, and Claude's `AskUserQuestion` / `Agent` instructions must not become Codex syntax unchanged.

## 3. Receive through the native skill workflow

For a target under `.claude/skills/`, invoke `/manage-skill` with the transfer contract. It creates
or updates the target in place, preserves Claude-native metadata, applies the skill-creator flow,
and synchronizes `SKILL.ja.md` from canonical English `SKILL.md`.

For a target under `.codex/skills/`, hand the same transfer contract to Codex `manage-skill`; do not
attempt to write Codex-specific configuration from Claude without that workflow.

Do not delete the source skill, commit, push, publish, or change unrelated skills.

## 4. Verify and report

Run the receiving environment's structural validation and inspect the target diff. Confirm that the
target preserves the source behavior while using only target-supported tools and configuration.
Report the direction, source commit/diff basis, port/adapt/omit decisions, files changed, and any
intent the target cannot express.

## Guardrails

- The source-side `manage-skill` starts synchronization; the receiver-side `manage-skill` is a
  child operation and must not start a second outbound synchronization.
- Keep platform-only skills platform-only and report why.
- Never use bidirectional automatic sync, timestamp conflict resolution, or wholesale overwrite.
- Keep transfer artifacts under ignored `tmp/`; maintain only the two native skill directories.
