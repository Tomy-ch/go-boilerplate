---
name: sync-ai
description: >-
  Reconcile a changed skill between this repository's Claude Code (`.claude/skills/`) and Codex (`.codex/skills/`) environments without blindly copying platform-specific instructions. Use after creating or materially updating a skill and when asked to synchronize, migrate, port, or compare a skill across Claude and Codex. Select one authoritative source direction, inspect the changed skill unit and its counterpart, then have the receiving environment's native `manage-skill` workflow adapt the intent, resources, and trigger description. Do not use for ordinary documentation translation, implementation code synchronization, or automatic two-way merges.
argument-hint: '[skill-name] [--from=claude|codex] [--to=claude|codex]'
---

# Sync AI Skills (Codex)

Synchronize a skill as a **one-way semantic port**, never as a raw directory copy. The source skill
is authoritative only for this run; the receiving skill remains native to its AI environment.

`manage-skill` invokes this after every new or material Codex skill change unless the skill is
explicitly Codex-only. For a Claude-originated change, use the same flow with Claude as the source.

## 1. Resolve direction and unit

Require exactly one skill name and one direction:

```text
source: .claude/skills/<name>/  → target: .codex/skills/<name>/
source: .codex/skills/<name>/   → target: .claude/skills/<name>/
```

Infer the source only when the user clearly names it. Do not infer direction from file mtimes.
If both copies changed, stop and ask which copy is authoritative; never merge two edited skills
automatically.

Inspect the complete source unit and the target unit if it exists:

```sh
git diff -- .claude/skills/<name> .codex/skills/<name>
find .claude/skills/<name> -type f | sort
find .codex/skills/<name> -type f | sort
```

Treat `SKILL.md`, `SKILL.ja.md`, UI metadata, scripts, references, and assets as one unit.
Record only a temporary transfer note under `tmp/sync-ai/`; do not create a durable third copy or
a synchronization manifest.

## 2. Build the transfer contract

Extract the source skill's:

- purpose and explicit trigger/non-trigger cases;
- required inputs, approvals, side effects, and validation;
- reusable scripts, references, and assets;
- platform-specific mechanisms that must be translated rather than copied.

Classify every source item as **port**, **adapt**, or **omit**. Examples: Claude's
`AskUserQuestion` / `Agent` instructions adapt to Codex's user-input / delegation mechanisms;
Claude-only settings, hooks, and permission syntax are omitted unless Codex has an equivalent.
Never carry source-specific tool names into the target without verifying the equivalent exists.

## 3. Receive through the native skill workflow

For a target under `.codex/skills/`, invoke `manage-skill` with the transfer contract. It must:

1. create or update the target skill in place;
2. preserve target-native frontmatter and `agents/openai.yaml`;
3. use the official skill-creator validation flow;
4. synchronize `SKILL.ja.md` from the English canonical `SKILL.md`.

For a target under `.claude/skills/`, hand the same transfer contract to Claude's
`/manage-skill`; do not attempt to write Claude-specific command syntax from Codex without that
workflow.

Do not delete the source skill after a successful port. Do not commit, push, publish, or change
unrelated skills.

## 4. Verify and report

Run the receiving environment's structural validation, then inspect the target diff. Confirm that
the target retains the source's behavior while using only target-supported tools and configuration.
Report the direction, source commit/diff basis, port/adapt/omit decisions, files changed, and any
intent that cannot be represented on the target.

## Guardrails

- The source-side `manage-skill` starts synchronization; the receiver-side `manage-skill` is a
  child operation and must not start a second outbound synchronization.
- A platform-only skill remains platform-only; record the reason instead of forcing a weak port.
- Never use a bidirectional automatic sync, timestamp conflict resolution, or wholesale overwrite.
- Keep transfer artifacts under ignored `tmp/`; the two native skill directories are the only
  maintained copies.
