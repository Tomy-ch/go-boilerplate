# .agents

English | [日本語](README.ja.md)

`.agents/` holds **artifacts produced and consumed by skills, for machines rather than for people**.

The human-facing counterpart of anything in here is the report a skill prints, or the canonical
documentation under `docs/` and the per-package `README.md`. Nothing in this directory is meant to be
read straight through; it is state that a skill writes down so the next run does not have to
re-derive it.

## Why not under `.claude/`

Every other AI-tool directory in this repository holds **configuration for one assistant** —
`.claude/` for Claude Code, `.codex/skills/` for the OpenAI Codex CLI, `.cursor/` for Cursor,
`.gemini/` for Gemini. Those are correctly vendor-scoped, because the instructions inside them are
written against one tool's contract.

An artifact is not. The same skill may be run from Claude, from Codex, or from Cursor, and its output
is the same data in every case. Filing that data under one assistant's directory would make the other
assistants either ignore it or duplicate it, and a duplicated ledger is a ledger that disagrees with
itself. So artifacts live here, one level up from any single tool.

## What belongs here

| Belongs | Does not belong |
| --- | --- |
| Machine-readable state a skill writes and reads back (ledgers, indexes, resolved caches) | Instructions to an assistant — those are tool configuration (`.claude/`, `.codex/skills/`, …) |
| Committed, shared machine-readable state whose audience is the next run of a skill | Per-run resume state — keep it in the owning skill's ignored artifact under `tmp/skills/<skill-name>/` |
| Output that is the same regardless of which assistant produced it | Anything tied to one vendor's contract |

Lockfiles for pinned toolchains (`.github/actions-pin.toml`, `docker/images-pin.toml`) are the same
kind of thing conceptually, but they stay next to the manifests they lock, because that is where the
tooling that maintains them looks.

## Layout

One subdirectory per artifact domain, named after the skill family that owns it.

- `doc-router/` — which documents govern an edit, keyed by where the edit lands. Read by a
  `PreToolUse` hook so the answer arrives at the moment of writing rather than being looked up
  again each time. Deliberately incomplete: a path with no entry falls back to the protocol.
- `ddd-audit/` — the DDD pattern ledger: which Evans pattern this repository has interpreted, and
  where. Owned by `.claude/skills/ddd-audit/SKILL.md`.
- `glossary-drift/` — the exclusions the glossary-drift detector honors: where a business term
  appearing outside `docs/spec/` is knowingly not a finding yet. Owned by
  `.claude/agents/drift-detector-glossary.md`.
- `closed-loop/` — the configuration the AI-feedback closed loop reads: per-skill usage class and
  opportunity predicate, and which comment authors are machines. The loop's *data* is not here — it
  lives in the issue tracker, per [ADR-0009](../docs/adr/0009-long-running-agent-state.md).
- `private/` — **the one machine-local subtree, and the only one that is gitignored.** It holds
  caches that can be regenerated from GitHub, such as the closed loop's branch-to-work-item index.
  Nothing here is shared, so nothing here may be the source of truth for anything: a second machine
  and a colleague picking up the work both see an empty directory.

A domain here may have a finite life. `comment-remediation/` recorded progress through a migration
to the comment policy and stopped meaning anything once the tree was fully swept, so it was deleted
along with its hook entry — that was the intended end state, not neglect. Judge each domain by
whether its question is still open, not by whether the directory is still there.

Each file documents its own schema in a header comment. The schema is not repeated here: a reader who
needs it is already opening the file, and a second copy would drift.

## Editing

Edit these files **through the skill that owns them**, not by hand. They record the result of an
analysis, so a hand-edited entry claims an analysis that never ran — and the next audit silently
inherits the claim. Where a skill offers a per-item approval loop, that loop is the intended way to
change the artifact.

Hand-editing is reasonable for one case: repairing a file that a failed run left malformed. Re-run
the owning skill afterwards so the content matches what the tooling would produce.

The distinction between durable knowledge and per-run state is decided in
[ADR-0009 (long-running-agent-state)](../docs/adr/0009-long-running-agent-state.md). A finding belongs in its owning canonical
document only when it changes that document's described relationship; do not turn activity logs into
another durable artifact domain.

## Scope for AI agents

`AGENTS.md` lists the directories an agent may modify. This one is not in that list, and the same
rule applies: **do not create, modify, or delete anything under `.agents/` unless the user asks, or
unless a skill whose defined procedure covers it is running.** Invoking a skill counts as that
instruction, for the paths that skill owns and for its duration only.
