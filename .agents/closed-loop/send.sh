#!/usr/bin/env sh
# Send the windows that have closed but never reached their feedback issue.
#
# This runs at SessionStart rather than SessionEnd, and the choice is deliberate. A window that
# has closed is complete and can be sent whenever; a session that is ending is the one moment the
# user wants the process gone, and a network call there hangs on the way out where nobody sees it
# fail. Deferring to the next start is also what the index already models — an entry with no
# feedback issue means "not sent yet", and the next start is the next chance. `/clear` closes a
# window and fires SessionStart in the same breath, so in practice a boundary sends almost at once.
#
# It sends only CLOSED windows. The one still open is incomplete, and half a window is worse than
# a late one.
#
# Everything here degrades to doing nothing. A missing runner, an unauthenticated `gh`, no network
# — each leaves the index untouched and the window pending, which is the state the next run reads.
# It must never block or fail a session (requirement §8), so it exits 0 unconditionally and runs
# the actual send in the background.
#
# Transcripts live under the user's home, which the tool-runner container cannot see; the path is
# therefore computed here and passed explicitly. Claude Code names a project directory after the
# checkout path with the separators replaced, so a worktree gets its own directory — which is why
# this resolves from the checkout it is run in rather than from a fixed name.

set -eu

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH='' cd -- "${SCRIPT_DIR}/../.." && pwd)
RUNNER="${REPO_ROOT}/scripts/node_modules/.bin/tsx"
ENTRY="${REPO_ROOT}/scripts/closed-loop/send"

usage() {
  cat <<'USAGE'
Usage:
  send.sh            Send closed, unsent windows (blocking; prints the result)
  send.sh --hook     Same, detached and silent (SessionStart)
  send.sh --dry-run  Print what would be sent, send nothing

Requires scripts/node_modules/.bin/tsx and an authenticated gh. Missing either is not an error:
the windows stay pending and the next run picks them up.
USAGE
}

# Claude Code's project directory for this checkout. `/Users/x/repo` -> `-Users-x-repo`.
transcripts_dir() {
  printf '%s/.claude/projects/%s' "${HOME}" "$(printf '%s' "${REPO_ROOT}" | tr '/' '-')"
}

run() {
  dir=$(transcripts_dir)
  if [ -d "${dir}" ]; then
    "${RUNNER}" "${ENTRY}" --transcripts "${dir}" "$@"
  else
    "${RUNNER}" "${ENTRY}" "$@"
  fi
}

case "${1:-}" in
  -h | --help)
    usage
    ;;
  --hook)
    # Detached: SessionStart must not wait on the network. Failures are invisible by design —
    # the index keeps the window pending, and the next start tries again.
    [ -x "${RUNNER}" ] || exit 0
    command -v gh >/dev/null 2>&1 || exit 0
    (run >/dev/null 2>&1 &) || :
    exit 0
    ;;
  --dry-run)
    [ -x "${RUNNER}" ] || { echo "tsx が見つかりません: ${RUNNER}" >&2; exit 1; }
    run --dry-run
    ;;
  '')
    [ -x "${RUNNER}" ] || { echo "tsx が見つかりません: ${RUNNER}" >&2; exit 1; }
    run
    ;;
  *)
    usage >&2
    exit 2
    ;;
esac
