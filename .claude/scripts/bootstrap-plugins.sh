#!/usr/bin/env bash
# Ensure the official Claude Code plugins this repository relies on are present.
#
# Declares them at PROJECT scope, so the marketplace + plugin enablement live in
# the repo's own `.claude/settings.json` and anyone who trusts this repo gets
# them — no per-developer setup. Idempotent and non-interactive: safe to re-run.
#
# Usage: bash .claude/scripts/bootstrap-plugins.sh
set -euo pipefail

MARKETPLACE="claude-plugins-official"
MARKETPLACE_SOURCE="anthropics/claude-plugins-official"
SCOPE="project"
# Official plugins this repo depends on. Add new ones here.
PLUGINS=(skill-creator feature-dev)

REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
PROJECT_SETTINGS="$REPO_ROOT/.claude/settings.json"

if ! command -v claude >/dev/null 2>&1; then
  echo "error: 'claude' CLI not found on PATH" >&2
  exit 1
fi

# 1. Ensure the official marketplace is declared at project scope (writes
#    extraKnownMarketplaces into .claude/settings.json). No-op if already there.
if grep -q "\"$MARKETPLACE\"" "$PROJECT_SETTINGS" 2>/dev/null; then
  echo "✔ marketplace already declared (project): $MARKETPLACE"
else
  echo "→ declaring marketplace (project scope): $MARKETPLACE_SOURCE"
  claude plugin marketplace add "$MARKETPLACE_SOURCE" --scope "$SCOPE"
fi

# 2. Ensure each plugin is enabled at project scope (writes enabledPlugins).
for p in "${PLUGINS[@]}"; do
  ref="${p}@${MARKETPLACE}"
  if grep -q "\"${ref}\"" "$PROJECT_SETTINGS" 2>/dev/null; then
    echo "✔ plugin already enabled (project): $ref"
  else
    echo "→ installing plugin (project scope): $ref"
    claude plugin install "$ref" --scope "$SCOPE"
  fi
done

# 3. Verify each plugin's files resolved on disk.
for p in "${PLUGINS[@]}"; do
  hit=$(ls -d ~/.claude/plugins/marketplaces/*/plugins/"$p" 2>/dev/null | head -1 || true)
  if [ -n "$hit" ]; then
    echo "✔ resolved: $p ($hit)"
  else
    echo "error: $p not found on disk after install" >&2
    exit 1
  fi
done

echo "done. Restart the Claude Code session to load newly enabled plugins."
