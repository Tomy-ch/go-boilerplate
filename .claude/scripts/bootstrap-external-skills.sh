#!/usr/bin/env bash
# Install the external skills this repository officially recommends, for every
# AI assistant the repo supports (Claude Code and Codex CLI).
#
# An external skill is a third-party skill that is NOT a Claude Code marketplace
# plugin, so `bootstrap-plugins.sh` cannot install it. The tool ships the skill
# and writes it into the assistant's USER-scope config dir (`~/.claude/skills/`,
# `~/.codex/skills/`) — unlike the plugins, which this repo declares at project
# scope. A fresh machine therefore needs this script even after a trusted clone.
#
# The version is pinned in `python/graphify.in`, and the resolved tree with its
# sha256 hashes in `python/graphify.txt`; this script never chooses one.
# Idempotent and non-interactive: safe to re-run.
#
# Usage: bash .claude/scripts/bootstrap-external-skills.sh
set -euo pipefail

# Platforms to install for, as named by `graphify install --platform`.
PLATFORMS=(claude codex)
# Landing path per platform, used to verify the install actually resolved.
SKILL_PATHS=("$HOME/.claude/skills/graphify/SKILL.md" "$HOME/.codex/skills/graphify/SKILL.md")

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
# user-scope install, and a path under the checkout would be one more thing for
# every worktree to ignore and clean up.
VENV="${XDG_CACHE_HOME:-$HOME/.cache}/go-boilerplate/graphify"

# `mise exec` rather than the shims: this script runs on a fresh machine where
# mise may not be activated in the shell yet.
mise install uv
mise exec uv -- uv venv --python "$(mise config get tools.python)" "$VENV"
# `--require-hashes` refuses any entry without a version and a hash, so a
# tampered lockfile fails here instead of installing.
mise exec uv -- uv pip install \
  --python "$VENV/bin/python" \
  --require-hashes \
  --no-cache \
  -r "$LOCKFILE"

# Install unconditionally rather than skipping on the `.graphify_version` marker:
# a successful install stamps that marker for EVERY platform already on disk
# (`_refresh_all_version_stamps`), so a marker match does not prove the skill
# itself was refreshed. Copying is cheap and idempotent; a stale skill is not.
#
# Use only `install --platform <name>`. The similarly named `<name> install`
# subcommands (`graphify codex install`, `graphify claude install`, …) write
# project-scope `AGENTS.md` / `CLAUDE.md` and assistant hooks — files AGENTS.md
# keeps hard-protected here.
for platform in "${PLATFORMS[@]}"; do
  echo "→ installing skill ($platform)"
  "$VENV/bin/graphify" install --platform "$platform"
done

# Verify every platform resolved on disk.
status=0
for i in "${!PLATFORMS[@]}"; do
  path="${SKILL_PATHS[$i]}"
  if [[ -s "$path" ]]; then
    echo "✔ resolved: ${PLATFORMS[$i]} ($path)"
  else
    echo "error: skill for ${PLATFORMS[$i]} not found at $path after install" >&2
    status=1
  fi
done
[ "$status" -eq 0 ] || exit "$status"

echo "done. Restart the assistant session to load the skill, then run '/graphify .'."
