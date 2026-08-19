#!/usr/bin/env sh
# Stamp the phase boundaries of a work window, so the closed loop can tell how long each phase
# actually took.
#
# A session transcript records every turn and every tool call, and it can say how long a turn
# took. What it cannot say is which phase the turn belonged to: nothing in the log distinguishes
# "still reading the code" from "writing the implementation" from "reviewing it". Those
# boundaries are known only to the workflow that crosses them, so the workflow writes them down
# here. Being a shell script rather than a client feature, it is also the one observation that
# behaves identically under every assistant.
#
# A window is one unit of work, not one session. A session runs for hours — measured median 3.1 h,
# p90 26 h — and a person clears context between pieces of work far more often than they quit.
# So `/clear` and a manual `/compact` end a window and start the next, while a resume continues
# the one already open. Marks are therefore filed per window, not per checkout: without that, two
# consecutive pieces of work in the same worktree would pool their timings into one meaningless
# average.
#
# Every mark is an EVENT STREAM: one file, one epoch per line, always appended. The reader takes
# the first line, the last, or the count, whichever the question needs — "when did implementation
# start" and "how many times was this reviewed" are then the same recording. Nothing needs to
# decide up front which marks may repeat.
#
#   separate files  A window is worked by several processes at once — subagents run in parallel —
#                   and appending to one shared file invites interleaved writes. Separate files
#                   cannot collide.
#   epoch per line  Comparable without parsing, in any language, with no timezone to get wrong.
#
# Marks live under the checkout's ignored `tmp/`. This is the same idiom as `.gobp-db-slot`,
# which `.makefiles/database/pool.mk` reads back the same way.
#
# The set of names is closed. An unknown name is refused rather than silently creating a file,
# because the failure it prevents is a typo becoming a mark nothing ever reads — invisible until
# someone asks why a phase has no data.
#
# It is also short on purpose. A moment GitHub already records is not stamped here: when a pull
# request opened and when it merged are read back from `gh` at aggregation time, and a mark for
# either would be a second copy of a fact, free to disagree with the first. Only the moments no
# other system observes get a mark.
#
# It always exits 0 when invoked as a hook. A window must open whether or not this can answer.

set -eu

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH='' cd -- "${SCRIPT_DIR}/../.." && pwd)
LOOP_DIR="${REPO_ROOT}/tmp/closed-loop"
CURRENT_FILE="${LOOP_DIR}/current"

# The phases the loop reports on. Adding one here is what makes it writable.
KNOWN_MARKS='openedAt planApprovedAt implStartedAt commitAt reviewStartedAt closedAt'

# What ends a window is a person saying "that is done": `/clear`, or a `/compact` they typed.
# `startup` and `resume` continue whatever was already open. An AUTOMATIC compact is the context
# filling up mid-task and is emphatically not a boundary — treating it as one would split a long
# piece of work in half and halve both of its phase timings. The two are distinguishable only at
# `PreCompact`, whose payload carries `trigger`; `SessionStart` reports both as `compact`, which
# is why manual compaction is caught at the earlier event rather than here.
ROTATING_SOURCES='clear'

usage() {
  cat <<'USAGE'
Usage:
  marks.sh <name>     Stamp <name> in the current window
  marks.sh --list     Print the current window's marks as `<name> <epoch>` lines
  marks.sh --window   Print the current window id (opening one if none exists)
  marks.sh --names    Print the writable mark names
  marks.sh --hook <event>
                      Called from a hook. <event> is one of:
                        session-start  rotate when `source` ends a window, else continue
                        pre-compact    rotate when `trigger` is manual, else continue
                        session-end    close the current window

Marks are written under tmp/closed-loop/marks/<window-id>/ in this checkout.
USAGE
}

is_known() {
  for known in ${KNOWN_MARKS}; do
    [ "$1" = "${known}" ] && return 0
  done
  return 1
}

# Enough entropy that two windows opened in the same second do not collide, without depending on
# a uuid binary being installed.
new_window_id() {
  suffix=$(od -An -N4 -tx1 /dev/urandom 2>/dev/null | tr -d ' \n') || suffix=""
  [ -n "${suffix}" ] || suffix=$$
  printf 'w%s-%s' "$(date +%s)" "${suffix}"
}

open_window() {
  close_window
  id=$(new_window_id)
  mkdir -p "${LOOP_DIR}"
  printf '%s\n' "${id}" >"${CURRENT_FILE}"
  mkdir -p "${LOOP_DIR}/marks/${id}"
  date +%s >>"${LOOP_DIR}/marks/${id}/openedAt"
  printf '%s' "${id}"
}

# Stamping the outgoing window is what makes a window a closed interval. Without it the last
# phase of every window would run to infinity, and no duration could be computed for it.
close_window() {
  [ -f "${CURRENT_FILE}" ] || return 0
  id=$(cat "${CURRENT_FILE}")
  [ -n "${id}" ] || return 0
  dir="${LOOP_DIR}/marks/${id}"
  [ -d "${dir}" ] || return 0
  [ -f "${dir}/closedAt" ] || date +%s >>"${dir}/closedAt"
}

current_window() {
  if [ -f "${CURRENT_FILE}" ]; then
    id=$(cat "${CURRENT_FILE}")
    [ -n "${id}" ] && { printf '%s' "${id}"; return 0; }
  fi
  open_window
}

case "${1:-}" in
  -h | --help)
    usage
    ;;
  --names)
    for known in ${KNOWN_MARKS}; do printf '%s\n' "${known}"; done
    ;;
  --window)
    current_window
    printf '\n'
    ;;
  --list)
    dir="${LOOP_DIR}/marks/$(current_window)"
    [ -d "${dir}" ] || exit 0
    for known in ${KNOWN_MARKS}; do
      file="${dir}/${known}"
      [ -f "${file}" ] || continue
      while IFS= read -r epoch; do
        [ -n "${epoch}" ] && printf '%s %s\n' "${known}" "${epoch}"
      done <"${file}"
    done
    ;;
  --hook)
    # The payload arrives on stdin. Reading it must never be able to fail the hook, so every step
    # degrades to "continue the current window" — the conservative answer, since a missed rotation
    # merges two windows while a spurious one destroys a window that was still open.
    field=""
    case "${2:-}" in
      session-start) field='.source' ;;
      pre-compact) field='.trigger' ;;
      session-end)
        close_window
        exit 0
        ;;
      *) exit 0 ;;
    esac

    value=""
    if command -v jq >/dev/null 2>&1; then
      payload=$(cat 2>/dev/null) || payload=""
      [ -n "${payload}" ] && value=$(printf '%s' "${payload}" | jq -r "${field} // empty" 2>/dev/null || printf '')
    fi

    rotate=0
    case "${2}" in
      session-start)
        for ending in ${ROTATING_SOURCES}; do
          [ "${value}" = "${ending}" ] && rotate=1
        done
        ;;
      pre-compact)
        [ "${value}" = "manual" ] && rotate=1
        ;;
    esac

    if [ "${rotate}" -eq 1 ] || { [ "${2}" = "session-start" ] && [ ! -f "${CURRENT_FILE}" ]; }; then
      open_window >/dev/null
    fi
    exit 0
    ;;
  '')
    usage >&2
    exit 2
    ;;
  *)
    if is_known "$1"; then
      dir="${LOOP_DIR}/marks/$(current_window)"
      mkdir -p "${dir}"
      date +%s >>"${dir}/$1"
    else
      printf 'unknown mark: %s\n' "$1" >&2
      printf 'writable names: %s\n' "${KNOWN_MARKS}" >&2
      exit 2
    fi
    ;;
esac
