#!/usr/bin/env sh

set -eu

repository_root=$(git rev-parse --show-toplevel 2>/dev/null || exit 0)
instructions_path="${repository_root}/CODEX.md"

[ -f "${instructions_path}" ] || exit 0

printf '%s\n\n' '以下はこの checkout に必須の Codex 運用指示です。作業を始める前に従ってください。'
cat "${instructions_path}"
