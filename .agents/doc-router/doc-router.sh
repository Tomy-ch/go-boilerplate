#!/usr/bin/env sh
# Name the documents that govern an edit, at the moment the edit is about to happen.
#
# AGENTS.md's Task Execution Protocol already says to read the owning README and to open the
# design / ADR indexes. This script does not restate that instruction — it answers the part the
# instruction cannot: for THIS path, which documents are they. Two answers, from two sources:
#
#   nearest README   Derived by walking up. No table, so it cannot go stale.
#   routed documents Read from routes.conf, for cases where the owning document's name does not
#                    contain the words anyone would search for.
#
# It always exits 0, and stays silent when it has nothing to add. The verdict is advice to
# whoever is about to write; a non-zero exit would turn advice into a blocked edit, and a
# missing entry would then read as a prohibition instead of as "no answer here, go look".
#
# Output is English regardless of the file's own language: a model reads it on every edit, and
# English is the cheaper encoding.

set -eu

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
ROUTES="${SCRIPT_DIR}/routes.conf"
REPO_ROOT=$(CDPATH='' cd -- "${SCRIPT_DIR}/../.." && pwd)

# The object store REPO_ROOT shares with its worktrees, and the sole test for whether a path
# belongs to this repository. Empty when git cannot answer, which narrows recognition to paths
# under REPO_ROOT.
REPO_COMMON=$(git -C "${REPO_ROOT}" rev-parse --path-format=absolute --git-common-dir 2>/dev/null || :)

usage() {
  cat <<'USAGE'
Usage:
  doc-router.sh <path>...   Print the governing documents for each path
  doc-router.sh --hook      Read a PreToolUse payload on stdin, emit hook JSON
  doc-router.sh --routes    Print the parsed routes as `<glob>\t<docs>\t<why>`

Silence means no routed document applies. The nearest ancestor README is always reported when
one exists.
USAGE
}

# The checkout holding an absolute path: REPO_ROOT itself, or one of its worktrees, which lives
# beside REPO_ROOT rather than under it. Empty for a path in another repository or in none.
checkout_root() {
  target=$1
  [ -n "${REPO_COMMON}" ] || return 0

  dir=$(dirname -- "${target}")
  while [ ! -d "${dir}" ]; do
    parent=$(dirname -- "${dir}")
    [ "${parent}" != "${dir}" ] || return 0
    dir=${parent}
  done

  info=$(git -C "${dir}" rev-parse --path-format=absolute --show-toplevel --git-common-dir 2>/dev/null) || return 0
  [ "$(printf '%s\n' "${info}" | sed -n 2p)" = "${REPO_COMMON}" ] || return 0
  printf '%s\n' "${info}" | sed -n 1p | tr -d '\n'
}

# Repository-relative, so an absolute path from a hook and a relative one from a shell reach the
# same entry, whichever checkout the file lives in.
to_relative() {
  target=$1
  case "${target}" in
    "${REPO_ROOT}/"*) printf '%s' "${target#"${REPO_ROOT}"/}" ;;
    /*)
      root=$(checkout_root "${target}")
      if [ -n "${root}" ] && [ "${target}" != "${target#"${root}"/}" ]; then
        printf '%s' "${target#"${root}"/}"
      else
        printf '%s' "${target}"
      fi
      ;;
    ./*) printf '%s' "${target#./}" ;;
    *) printf '%s' "${target}" ;;
  esac
}

# Where the path is on disk, which for a worktree file is not under REPO_ROOT.
checkout_for() {
  target=$1
  case "${target}" in
    /*) checkout_root "${target}" || : ;;
    *) printf '%s' "${REPO_ROOT}" ;;
  esac
}

# The nearest ancestor README of a repository-relative path, searched in the checkout the file
# actually lives in. A file directly under the repository root has none worth naming: the root
# README is an entry point, not a package contract.
nearest_readme() {
  rel=$1
  root=$2
  [ -n "${root}" ] || root=${REPO_ROOT}

  dir=$(dirname -- "${rel}")
  while [ "${dir}" != "." ] && [ "${dir}" != "/" ]; do
    if [ -f "${root}/${dir}/README.md" ] && [ "${dir}/README.md" != "${rel}" ]; then
      printf '%s' "${dir}/README.md"
      return 0
    fi
    dir=$(dirname -- "${dir}")
  done
}

# Parsed routes, tab-separated. Comments and blank lines are dropped here so every caller sees
# the same view of the file.
routes() {
  [ -f "${ROUTES}" ] || return 0
  awk -F' = ' '
    /^[[:space:]]*#/ { next }
    /^[[:space:]]*$/ { next }
    NF < 2 { next }
    {
      glob = $1
      rest = $2
      why = ""
      hash = index(rest, "#")
      if (hash > 0) {
        why = substr(rest, hash + 1)
        rest = substr(rest, 1, hash - 1)
      }
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", glob)
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", rest)
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", why)
      print glob "\t" rest "\t" why
    }
  ' "${ROUTES}"
}

# The first matching route for a repository-relative path, or nothing. First match wins, so a
# specific path can be routed differently from the subtree around it.
match_route() {
  rel=$1
  routes | while IFS="$(printf '\t')" read -r glob docs why; do
    # shellcheck disable=SC2254 # the glob is the pattern, expanding it is the point
    case "${rel}" in
      ${glob})
        printf '%s\t%s' "${docs}" "${why}"
        break
        ;;
      *) ;;
    esac
  done
}

# One line of advice for one path, or nothing when there is nothing to add.
verdict_line() {
  target=$1
  rel=$(to_relative "${target}")
  root=$(checkout_for "${target}")

  readme=$(nearest_readme "${rel}" "${root}")
  matched=$(match_route "${rel}")
  docs=$(printf '%s' "${matched}" | cut -f1)
  why=$(printf '%s' "${matched}" | cut -f2)

  [ -n "${readme}" ] || [ -n "${docs}" ] || return 0

  line="${rel}:"
  [ -z "${readme}" ] || line="${line} governed by ${readme}"
  if [ -n "${docs}" ]; then
    [ -z "${readme}" ] || line="${line};"
    line="${line} ${docs}"
    [ -z "${why}" ] || line="${line} — ${why}"
  fi
  printf '%s' "${line}"
}

run_hook() {
  # Without jq there is no way to read the payload. Staying silent is the right failure: the
  # alternative is an error notice on every single edit.
  command -v jq >/dev/null 2>&1 || exit 0

  payload=$(cat) || exit 0
  paths=$(printf '%s' "${payload}" | jq -r '.tool_input.file_path // empty' 2>/dev/null) || exit 0
  if [ -z "${paths}" ]; then
    paths=$(printf '%s' "${payload}" | jq -r '.tool_input.command // empty' 2>/dev/null \
      | awk '/^\*\*\* (Add|Update|Delete) File: / { sub(/^\*\*\* (Add|Update|Delete) File: /, ""); print }') || exit 0
  fi
  [ -n "${paths}" ] || exit 0

  context=''
  while IFS= read -r target; do
    [ -n "${target}" ] || continue
    line=$(verdict_line "${target}") || continue
    [ -n "${line}" ] || continue
    context="${context}${context:+
}${line}"
  done <<EOF
${paths}
EOF
  [ -n "${context}" ] || exit 0

  # Deliberately pointers, not the procedure. AGENTS.md's Task Execution Protocol owns the
  # instruction to read these; repeating it here would put one rule in two places, and the copy
  # would be the one that drifts.
  jq -n --arg context "${context}" '{
    hookSpecificOutput: {
      hookEventName: "PreToolUse",
      additionalContext: ("Read before editing — " + $context)
    }
  }'
}

case "${1:-}" in
  --hook)
    run_hook
    ;;
  --routes)
    routes
    ;;
  -h | --help | '')
    usage
    ;;
  *)
    for target in "$@"; do
      line=$(verdict_line "${target}")
      if [ -n "${line}" ]; then
        printf '%s\n' "${line}"
      fi
    done
    ;;
esac
