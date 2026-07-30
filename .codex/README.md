# `.codex/` — Codex configuration for this repository

This directory contains the Codex configuration distributed with the repository: project defaults,
reusable skills, read-only agent roles, and the spec templates used by scaffold and validation
workflows. A trusted clone receives the same repository-level development workflow without receiving
any personal credentials or account settings.

For the Japanese reference translation, see [README.ja.md](README.ja.md).

## Relationship to `AGENTS.md`

[`AGENTS.md`](../AGENTS.md) is the human-maintained operational contract. It defines what an agent may
change, protected targets, architecture boundaries, Git rules, and validation expectations. Read it
before using or modifying this directory.

`config.toml` records this scaffold's intended Codex posture; it is not a loaded repository-level
configuration. As measured with `codex-cli 0.145.0`, Codex does not auto-load project-level
configuration. Apply its values with `-c key=value`, layer them through `--profile`, or reflect them
in `~/.codex/config.toml`. Nothing is presently mis-set because the effective Codex defaults already
coincide with this intended posture. The fine-grained safety rules remain canonical in `AGENTS.md`:
generated-file protection, prohibited Git operations, and repository modification scope are not
duplicated here.

## Layout

| Path | Purpose |
| --- | --- |
| `config.toml` | Recorded Codex posture: sandbox policy, approval policy, and enabled capabilities. Codex does not auto-load it; apply it with `-c`, `--profile`, or `~/.codex/config.toml`. |
| `skills/<name>/` | Reusable workflows. Each skill has an English-canonical `SKILL.md`; `agents/openai.yaml` provides Codex UI metadata. Some skills bundle `scripts/` or `prompts/`. |
| `agents/` | Role definitions for read-only parallel work, such as layer auditors and review verifiers. Integrator skills use them when delegation is available and otherwise execute the same role instructions inline. |
| `scaffold-spec/` | Runtime spec-format definitions used by `new-spec-*`, `verify-spec`, and `scaffold-*` skills. Updating these files changes the shared format without duplicating it into skills. |

## First-time setup

Install and authenticate personal tools outside this repository. Do not commit credentials, tokens,
or user-specific MCP settings here.

```bash
codex login
gh auth login       # required by GitHub-aware skills
mise install        # installs the repository-pinned toolchain
```

Docker is required for workflows that run the repository's containerized tools or database. Follow
the repository root documentation and `AGENTS.md` for project verification commands.

MCP servers, personal model preferences, and account-specific plugins belong in the user's Codex
configuration. Add a project-level integration only when every trusted clone needs it and its setup
contains no secret or machine-specific value.

### Recommended external skills

This repo officially recommends `graphify` (`/graphify`); see
[`.claude/README.md`](../.claude/README.md) for what it does. It installs into user scope
(`~/.codex/skills/graphify/`), so a trusted clone does not carry it.

One bootstrap covers both Codex and Claude Code, and reads the pinned version from `mise.toml`:

```bash
bash .claude/scripts/bootstrap-external-skills.sh
```

The script lives under `.claude/` only because that is where this repo keeps agent-tooling scripts;
it installs for every supported assistant. Operational notes — what the graph excludes, which
commands stay local versus call an LLM API, and the uninstall caveat — are in
[`.claude/README.md`](../.claude/README.md); the version-bump procedure is in
[`docs/design/security.md`](../docs/design/security.md).

## Conventions

- **English is canonical.** Skill and configuration bodies are written in English. User-visible
  runtime output follows `AGENTS.md` and is Japanese by default.
- **Use `$manage-skill` to author or revise a repository skill.** Keep the skill concise and place
  reusable deterministic work in bundled scripts rather than duplicating it in prompts.
- **Keep agent roles narrow and read-only.** The integrating skill owns user confirmation and any
  writes, so delegated work can safely run in parallel.
- **Discover the available workflow set with `$tool-map`.** It inventories repository-local skills,
  agent roles, and their cross-references.
- **Do not put personal configuration here.** Use `~/.codex/` for model preferences, credentials,
  personal MCP servers, and other machine-specific defaults.
