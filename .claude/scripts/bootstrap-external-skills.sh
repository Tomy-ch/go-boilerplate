#!/usr/bin/env bash
# Install the external skills this repository officially recommends, for every
# AI assistant the repo supports (Claude Code and Codex CLI).
#
# An external skill is a third-party skill that is NOT a Claude Code marketplace
# plugin, so `bootstrap-plugins.sh` cannot install it. The tool ships the skill;
# this script lands it inside the checkout (`.claude/skills/`, `.codex/skills/`),
# the same way `vendor/` and `node_modules/` hold a dependency's own files: built
# from a pin, ignored by git, re-made per machine rather than reviewed.
#
# The version is pinned in `python/graphify.in`, and the resolved tree with its
# sha256 hashes in `python/graphify.txt`; this script never chooses one.
# Idempotent and non-interactive: safe to re-run.
#
# Usage: bash .claude/scripts/bootstrap-external-skills.sh
set -euo pipefail

# Platforms to install for, as named by `graphify install --platform`, paired
# with the directory each one's skill lands in inside the checkout.
PLATFORMS=(claude codex)
SKILL_DIRS=(.claude/skills/graphify .codex/skills/graphify)

if ! command -v mise >/dev/null 2>&1; then
  echo "error: 'mise' not found on PATH; install it first (see README)" >&2
  exit 1
fi

REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
cd "$REPO_ROOT"

LOCKFILE=python/graphify.txt
if [[ ! -s "$LOCKFILE" ]]; then
  echo "error: lockfile '$LOCKFILE' not found; run 'make py-lock' first" >&2
  exit 1
fi
echo "→ pinned graphify version ($LOCKFILE): $(sed -n 's/^graphifyy==\([^ ]*\).*/\1/p' "$LOCKFILE")"

# The venv lives outside the repository: it is a machine-local artifact of a
# tool install, and a path under the checkout would be one more thing for every
# worktree to ignore and clean up.
VENV="${XDG_CACHE_HOME:-$HOME/.cache}/go-boilerplate/graphify"

# `mise exec` rather than the shims: this script runs on a fresh machine where
# mise may not be activated in the shell yet.
mise install uv
# `--clear` rather than reusing an existing venv: `uv pip install` only adds, so a package dropped
# from the lockfile since the last run would survive in a reused environment. Without the flag
# `uv venv` refuses outright, which is what made a second run of this script fail.
mise exec uv -- uv venv --clear --python "$(mise config get tools.python)" "$VENV"
# `--require-hashes` refuses any entry without a version and a hash, so a
# tampered lockfile fails here instead of installing.
mise exec uv -- uv pip install \
  --python "$VENV/bin/python" \
  --require-hashes \
  --no-cache \
  -r "$LOCKFILE"

# Every install mode the tool offers writes something this repository keeps
# hard-protected: `--project` appends a `## graphify` section to `CLAUDE.md` —
# which is a symlink to `AGENTS.md` here — and registers PreToolUse hooks in
# `.claude/settings.json`, while the default user-scope mode edits
# `~/.claude/CLAUDE.md` and leaves the skill on the machine rather than in the
# checkout. So the install is run against a throwaway HOME and only the skill
# directory is taken from it. Nothing outside the checkout is touched, and the
# copy is the whole of what this script installs.
STAGE=$(mktemp -d)
trap 'rm -rf "$STAGE"' EXIT

for i in "${!PLATFORMS[@]}"; do
  platform="${PLATFORMS[$i]}"
  dst="${SKILL_DIRS[$i]}"
  echo "→ installing skill ($platform) -> $dst"

  HOME="$STAGE" "$VENV/bin/graphify" install --platform "$platform" >/dev/null

  # `graphify install` names the landing directory after the platform's own
  # config dir, which is what the staged HOME now contains.
  staged="$STAGE/.${platform}/skills/graphify"
  if [[ ! -s "$staged/SKILL.md" ]]; then
    echo "error: skill for $platform not found at $staged after install" >&2
    exit 1
  fi

  # Replace rather than merge: a file upstream dropped between versions would
  # otherwise survive as a stale reference the skill still resolves.
  rm -rf "${dst:?}"
  mkdir -p "$(dirname "$dst")"
  cp -R "$staged" "$dst"
  echo "✔ resolved: $platform ($dst)"
done

echo "done. Restart the assistant session to load the skill, then run '/graphify .'."
