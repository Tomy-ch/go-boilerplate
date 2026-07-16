# `.claude/` — Agent configuration for this repository

This directory holds the Claude Code configuration that ships **with the repo**: reusable skills,
subagents, the spec templates the scaffolding reads, helper scripts, and project-scoped permissions /
plugin declarations. Anyone who clones and trusts the repo inherits the same agent behavior.

For a Japanese reference translation, see [`README.ja.md`](README.ja.md).

## Relationship to `AGENTS.md`

`AGENTS.md` (repo root) is the human-maintained **operational contract** — what an agent may touch and
how it must behave. It is the source of truth; this README does not restate its rules. In particular,
`AGENTS.md` treats `.claude/**` as **out of scope for AI edits** unless the user explicitly requests a
change or a running skill relaxes the scope for its own duration. Read `AGENTS.md` first.

## Layout

| Path | What it is |
| --- | --- |
| `settings.json` | Project-scoped permissions (`allow` / `ask` / `deny`), enabled plugins, and known marketplaces. Shared with everyone who trusts the repo. |
| `skills/<name>/` | Reusable skills. Each has an English-canonical `SKILL.md` (+ a `SKILL.ja.md` reference translation) and optional bundled `scripts/` / `references/`. Invoked as `/<name>`. |
| `agents/` | Subagent definitions used by the skills (e.g. the read-only `arch-auditor-*` / `drift-detector-*` per-layer workers that integrator skills fan out in parallel). |
| `scaffold-spec/` | Spec-format definitions (`domain-spec.md`, `usecase-spec.md`, `verify-rules.md`, …) read **at runtime** by the `scaffold-*` / `verify-spec` / `new-spec-*` skills, so format changes propagate without editing the skills. |
| `scripts/` | Repo-level agent-tooling scripts (not project build tooling — that lives in the root `scripts/`). |

## First-time setup: official plugins

This repo depends on Anthropic's official plugins, declared at **project scope** in `settings.json`
(`enabledPlugins` + `extraKnownMarketplaces`):

- `skill-creator` — wrapped by the local `manage-skill` skill for authoring/updating skills.
- `feature-dev` — guided feature-development workflow (`/feature-dev`) with `code-explorer` /
  `code-architect` / `code-reviewer` agents.

A trusted clone already has them. If they are missing (or you added a new one to the list), run the
idempotent bootstrap:

```bash
bash .claude/scripts/bootstrap-plugins.sh
```

Newly enabled plugins load on the **next** Claude Code session.

## Conventions

- **English is canonical.** Skill/README bodies are written in imperative English; the paired
  `*.ja.md` is a human reference translation kept in sync via the `canonicalize-doc` skill. Runtime
  output to the user still follows `CLAUDE.md` (Japanese).
- **Authoring skills.** Use `/manage-skill` to create or update a skill — it wraps `skill-creator` and
  applies this repo's placement, frontmatter, translation-pair, and eval-artifact conventions.
- **Discovering what exists.** Run `/tool-map` for a full inventory of skills / agents / commands and a
  dependency map, instead of maintaining a hand-written list here.
