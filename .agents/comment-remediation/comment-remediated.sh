#!/usr/bin/env sh
# Answer whether a file's comments have already been swept, by looking it up in
# comment-remediated.toml.
#
# Two modes. With paths as arguments it prints one verdict per path. With --hook it reads a
# Claude Code or Codex PreToolUse payload on stdin and emits the additionalContext envelope.
#
# It always exits 0. The verdict is advice to whoever is about to edit the file, and a
# non-zero exit from the hook path would turn that advice into a blocked edit — which would
# make every one-line fix drag a whole-file sweep behind it.
#
# Output is English regardless of the file's own language: it is read by a model on every
# edit, and English is the cheaper encoding.

set -eu

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
LEDGER="${SCRIPT_DIR}/comment-remediated.toml"
REPO_ROOT=$(CDPATH='' cd -- "${SCRIPT_DIR}/../.." && pwd)
PROMPT_REL=".agents/comment-remediation/feedback_comment_remediation_sweep_on_touch.prompt"

CLEAR='no action needed'
REQUIRED='comment remediation required'

usage() {
  cat <<'USAGE'
Usage:
  comment-remediated.sh <path>...   Print a verdict per path
  comment-remediated.sh --hook      Read a PreToolUse payload on stdin, emit hook JSON

Verdicts:
  no action needed               In the ledger, or holding no comment stock to sweep —
                                 the parenthetical says which.
  comment remediation required   Read feedback_comment_remediation_sweep_on_touch.prompt
                                 in this directory and follow it.
USAGE
}

# Repository-relative, so an absolute path from a hook and a relative one from a shell
# reach the same entry. Paths outside the repository are left as they are and will simply
# miss.
to_relative() {
  case "$1" in
    "${REPO_ROOT}/"*) printf '%s' "${1#"${REPO_ROOT}"/}" ;;
    /*) printf '%s' "$1" ;;
    ./*) printf '%s' "${1#./}" ;;
    *) printf '%s' "$1" ;;
  esac
}

# Nothing outside the sweep's reach: generated output is rewritten by its generator, and
# vendored code is not ours to restyle. A path that does not exist is a file being created,
# which has no comment stock to sweep. Prints the reason, empty when the file is in scope.
out_of_scope_reason() {
  case "$1" in
    vendor/* | */node_modules/* | node_modules/*)
      printf 'vendored'
      return
      ;;
    *.gen.go | *.sql.go | *_mock.go | *.gen.yaml | *.gen.sql)
      printf 'generated'
      return
      ;;
    docs/portal/* | docs/coverage/* | docs/db-schema/* | docs/godoc/* | docs/openapi/*)
      printf 'generated'
      return
      ;;
    *.lock | *lock.json | *lock.yaml | *.sum)
      printf 'lockfile'
      return
      ;;
  esac

  [ -f "${REPO_ROOT}/$1" ] || printf 'file does not exist'
}

# The lookup is a literal match on the quoted key rather than a TOML parse, so the script
# stays dependency-free. It holds because every key is written quoted, which the header of
# the ledger states as its schema.
is_listed() {
  grep -qF "\"$1\" = " "${LEDGER}"
}

verdict() {
  rel=$(to_relative "$1")
  reason=$(out_of_scope_reason "${rel}")

  if [ -n "${reason}" ]; then
    printf '%s (%s)' "${CLEAR}" "${reason}"
  elif is_listed "${rel}"; then
    printf '%s (in ledger)' "${CLEAR}"
  else
    printf '%s' "${REQUIRED}"
  fi
}

run_hook() {
  # Without jq there is no way to read the payload. Staying silent is the right failure:
  # the alternative is an error notice on every single edit.
  command -v jq >/dev/null 2>&1 || exit 0

  payload=$(cat) || exit 0
  paths=$(printf '%s' "${payload}" | jq -r '.tool_input.file_path // empty' 2>/dev/null) || exit 0
  if [ -z "${paths}" ]; then
    paths=$(printf '%s' "${payload}" | jq -r '.tool_input.command // empty' 2>/dev/null \
      | awk '/^\*\*\* (Add|Update|Delete) File: / { sub(/^\*\*\* (Add|Update|Delete) File: /, ""); print }') || exit 0
  fi
  [ -n "${paths}" ] || exit 0

  required=''
  while IFS= read -r path; do
    [ -n "${path}" ] || continue
    rel=$(to_relative "${path}")
    [ -z "$(out_of_scope_reason "${rel}")" ] || continue
    is_listed "${rel}" && continue
    required="${required}${required:+, }${rel}"
  done <<EOF
${paths}
EOF
  [ -n "${required}" ] || exit 0

  # Deliberately a pointer, not the instruction. This is emitted on every edit to every
  # unswept file, so its cost is paid whether or not the sweep happens; the prompt behind it
  # is read once, and only when it will be acted on.
  jq -n --arg paths "${required}" --arg prompt "${PROMPT_REL}" '{
    hookSpecificOutput: {
      hookEventName: "PreToolUse",
      additionalContext: (
        $paths + ": comments predate the current policy. Make your edit, then before "
        + "finishing the task read " + $prompt + " and follow it."
      )
    }
  }'
}

if [ ! -f "${LEDGER}" ]; then
  echo "ledger not found: ${LEDGER}" >&2
  exit 0
fi

case "${1:-}" in
  --hook)
    run_hook
    ;;
  -h | --help | '')
    usage
    ;;
  *)
    for target in "$@"; do
      printf '%s\t%s\n' "$(verdict "${target}")" "$(to_relative "${target}")"
    done
    ;;
esac
