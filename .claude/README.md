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

## First-time setup: recommended external skills

An **external skill** is a third-party skill that is not a marketplace plugin, so the plugin
bootstrap cannot install it. This repo officially recommends one:

- `graphify` (`/graphify`) — parses the repo with tree-sitter into a queryable knowledge graph
  under `graphify-out/`, then answers structural questions (`query` / `affected` / `god-nodes`)
  from the graph instead of from repeated greps.

Install it for every assistant this repo supports (Claude Code + Codex CLI) with the idempotent
bootstrap:

```bash
bash .claude/scripts/bootstrap-external-skills.sh
```

Three properties differ from the plugins above and are worth knowing before running it:

- **User scope, not project scope.** The skill is written to `~/.claude/skills/graphify/` (and
  `~/.codex/skills/graphify/`), so a trusted clone does *not* carry it — every machine runs the
  bootstrap once. The version is pinned at project scope in `mise.toml`, and the script reads that
  pin rather than choosing one.
- **The installer also writes `~/.claude/CLAUDE.md`** (user-global memory) to register the
  `/graphify` trigger. It does not touch this repository's `CLAUDE.md`.
- **`graphify uninstall` leaves the Codex copy behind** — remove
  `~/.codex/skills/graphify/` by hand. `--purge` additionally deletes `graphify-out/`.

Install only through `install --platform <name>`, which is what the bootstrap runs. The
similarly named `<name> install` subcommands are a different thing: `graphify claude install`
writes this repository's `CLAUDE.md`, `graphify codex install` (also `opencode` / `aider` /
`kilo`) writes `AGENTS.md`, and `graphify hook install` adds git hooks and a merge driver — all
project scope, and `AGENTS.md` / `CLAUDE.md` are hard-protected by `AGENTS.md`. Those forms are
in `settings.json`'s `deny` list so an agent reading `graphify --help` cannot reach them.

The graph is a derived artifact and is gitignored, so build it locally. `update` and the query
commands are AST-only and need no API key; the docs / PDF / image extraction, `--mode deep`
inference, `--wiki`, and community *naming* call an LLM API and send content off the machine, so
keep those opt-in.

```bash
mise exec "pipx:graphifyy[sql]" -- graphify update . --no-cluster
```

What the graph excludes (committed generated artifacts, Japanese mirrors, vendored code) is
declared in `.graphifyignore`. Changing it requires a full re-extraction — an incremental `update`
is fail-closed and keeps the now-excluded nodes.

## Conventions

- **English is canonical.** Skill/README bodies are written in imperative English; the paired
  `*.ja.md` is a human reference translation kept in sync via the `canonicalize-doc` skill. Runtime
  output to the user still follows `CLAUDE.md` (Japanese).
- **Authoring skills.** Use `/manage-skill` to create or update a skill — it wraps `skill-creator` and
  applies this repo's placement, frontmatter, translation-pair, and eval-artifact conventions.
- **Discovering what exists.** Run `/tool-map` for a full inventory of skills / agents / commands and a
  dependency map, instead of maintaining a hand-written list here.
- **Definitions are linted against reality.** `make md-skill-lint` (part of `make md-lint`, so it runs
  on the `pre-commit` hook) checks frontmatter, translation-pair heading structure, and the existence
  of every `make` target and path this directory's prose references. A skill is an instruction sheet —
  a reference that has rotted makes an agent execute the wrong step. Scope and the
  `<!-- skill-lint-ignore -->` directive are documented in [`scripts/README.md`](../scripts/README.md).
