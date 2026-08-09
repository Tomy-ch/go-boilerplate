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
| Data whose audience is the next run of a skill | Prose meant to be read by a person — that is `docs/` or a package `README.md` |
| Output that is the same regardless of which assistant produced it | Anything tied to one vendor's contract |

Lockfiles for pinned toolchains (`.github/actions-pin.toml`, `docker/images-pin.toml`) are the same
kind of thing conceptually, but they stay next to the manifests they lock, because that is where the
tooling that maintains them looks.

## Layout

One subdirectory per artifact domain, named after the skill family that owns it.

```txt
.agents/
├── comment-remediation/
│   ├── comment-remediated.toml   # files whose comment stock has been swept
│   ├── comment-remediated.sh     # the lookup, run by a PreToolUse hook before an edit
│   └── …sweep_on_touch.prompt    # what to do about a miss, read on demand
└── ddd-audit/
    └── pattern-ledger.yaml       # DDD pattern ledger — see .claude/skills/ddd-audit/SKILL.md
```

`comment-remediation/` is the one domain here with a finite life. It records progress through a
migration to the comment policy, so it stops meaning anything once the tree is fully swept; deleting
the directory and the hook entry in `.claude/settings.json` is then the intended end state, not
neglect.

Each file documents its own schema in a header comment. The schema is not repeated here: a reader who
needs it is already opening the file, and a second copy would drift.

## Editing

Edit these files **through the skill that owns them**, not by hand. They record the result of an
analysis, so a hand-edited entry claims an analysis that never ran — and the next audit silently
inherits the claim. Where a skill offers a per-item approval loop, that loop is the intended way to
change the artifact.

Hand-editing is reasonable for one case: repairing a file that a failed run left malformed. Re-run
the owning skill afterwards so the content matches what the tooling would produce.

## Scope for AI agents

`AGENTS.md` lists the directories an agent may modify. This one is not in that list, and the same
rule applies: **do not create, modify, or delete anything under `.agents/` unless the user asks, or
unless a skill whose defined procedure covers it is running.** Invoking a skill counts as that
instruction, for the paths that skill owns and for its duration only.
