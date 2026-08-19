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
# A window is one unit of work, not one session, so marks are filed per window rather than per
# checkout. `/clear` and a manual `/compact` end a window and start the next; a resume continues
# the one already open. Why the session is the wrong unit, with the measurements behind it, is
# docs/adr/0010-development-window-as-feedback-unit.md.
#
# Every mark is an EVENT STREAM: one file, one epoch per line, always appended. The reader takes
# the first line, the last, or the count, whichever the question needs — "when did implementation
# start" and "how many times was this reviewed" are then the same recording. Nothing needs to
# decide up front which marks may repeat.
#
#   separate files  Marks are appended from more than one process — a git hook fires while a
#                   session is mid-turn, and two sessions can share one checkout — so appending to
#                   one shared file would interleave writes. Separate files cannot collide. Note
#                   that this protects the MARKS only: the pointer to the current window is a
#                   single shared file, and it needs the lock below.
#   epoch per line  Comparable without parsing, in any language, with no timezone to get wrong.
#
# Marks live under the checkout's ignored `tmp/`. This is the same idiom as `.gobp-db-slot`,
# which `.makefiles/database/pool.mk` reads back the same way.
#
# The set of names is closed. An unknown name is refused rather than silently creating a file,
# because the failure it prevents is a typo becoming a mark nothing ever reads — invisible until
# someone asks why a phase has no data.
#
# A mark stays the primary record even where GitHub also knows the same moment: `gh` is the
# cross-check and the fallback, never the source, and every value carries `source` so that a
# disagreement stays answerable. Why the two are not the same fact, and not equally reliable, is
# docs/adr/0010-development-window-as-feedback-unit.md.
#
# It always exits 0 when invoked as a hook. A window must open whether or not this can answer.

set -eu

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH='' cd -- "${SCRIPT_DIR}/../.." && pwd)
LOOP_DIR="${REPO_ROOT}/tmp/closed-loop"
CURRENT_FILE="${LOOP_DIR}/current"

# The phases the loop reports on. Adding one here is what makes it writable.
KNOWN_MARKS='openedAt planApprovedAt implStartedAt commitAt reviewStartedAt prOpenedAt mergedAt closedAt'

# What ends a window is a person saying "that is done": `/clear`, or a `/compact` they typed.
# `startup` and `resume` continue whatever was already open. Do NOT add `compact` here —
# `SessionStart` reports the automatic and the manual form alike, and only the manual one is a
# boundary (docs/adr/0010-development-window-as-feedback-unit.md), which is why it is caught at
# `PreCompact` instead, whose payload carries `trigger`.
ROTATING_SOURCES='clear'

# The pointer to the current window is one shared file, so the marks' separate-file safety does
# not extend to it. Rotating is read-modify-write across several commands, and two processes doing
# it at once leave one window orphaned — reproduced at 29 of 30 with no timing help at all.
#
# `mkdir` is the lock because it is atomic on every POSIX filesystem and needs no `flock`, which
# macOS does not ship. A stale lock from a killed process would wedge every later run, so the wait
# is bounded and gives up rather than blocking a session — losing a rotation degrades the data,
# while hanging the hook degrades the work.
LOCK_DIR="${LOOP_DIR}/.rotate.lock"
LOCK_HELD=0

acquire_lock() {
  attempt=0
  while [ "${attempt}" -lt 50 ]; do
    if mkdir "${LOCK_DIR}" 2>/dev/null; then
      LOCK_HELD=1
      return 0
    fi
    attempt=$((attempt + 1))
    sleep 0.1
  done
  return 1
}

release_lock() {
  [ "${LOCK_HELD}" -eq 1 ] || return 0
  rmdir "${LOCK_DIR}" 2>/dev/null || :
  LOCK_HELD=0
}

trap release_lock EXIT INT TERM

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
  mkdir -p "${LOOP_DIR}"
  acquire_lock || {
    # ロックを取れなければ、既に別プロセスがローテーションしている。そちらの結果に従うのが
    # 正しく、ここで重ねて開くと窓が二重になる。
    current_window_unsafe
    return 0
  }
  close_window
  id=$(new_window_id)
  mkdir -p "${LOOP_DIR}/marks/${id}"
  date +%s >>"${LOOP_DIR}/marks/${id}/openedAt"
  # ポインタは最後に、一時ファイル経由で差し替える。窓の中身が揃う前にポインタだけが
  # 新窓を指す瞬間を作らないため。同一ディレクトリ内の mv は原子的に置き換わる。
  printf '%s\n' "${id}" >"${CURRENT_FILE}.tmp.$$"
  mv "${CURRENT_FILE}.tmp.$$" "${CURRENT_FILE}"
  release_lock
  printf '%s' "${id}"
}

# Stamping the outgoing window is what makes a window a closed interval. Without it the last
# phase of every window would run to infinity, and no duration could be computed for it.
close_window() {
  id=$(current_window_unsafe) || return 0
  dir="${LOOP_DIR}/marks/${id}"
  [ -d "${dir}" ] || return 0
  [ -f "${dir}/closedAt" ] || date +%s >>"${dir}/closedAt"
}

# 現在の窓を読むだけで、無ければ開かない。診断や一覧のように「見るだけ」の経路が
# 窓を作ってしまうと、誰も打刻しない空の窓がレポートに残り続ける。
current_window_unsafe() {
  [ -f "${CURRENT_FILE}" ] || return 1
  id=$(cat "${CURRENT_FILE}")
  [ -n "${id}" ] || return 1
  printf '%s' "${id}"
}

current_window() {
  current_window_unsafe && return 0
  open_window
}

# 窓 ID を引数で受け取る。呼び出し側が 1 度だけ読んだ ID に対して判定と書き込みの両方を
# 行えるようにするため。読み直すと、その間にローテーションが割り込んだとき、判定した窓と
# 書き込む窓が食い違う。
is_window_closed() {
  [ -n "${1:-}" ] || return 1
  [ -f "${LOOP_DIR}/marks/$1/closedAt" ]
}

# Work arriving after a window closed belongs to the next window, not the last one. A commit that
# lands after the session ended is the start of something, not a late footnote to what finished —
# and filing it under the closed window puts a mark after that window's own end, which is a
# timestamp no interval can be computed from.
#
# The two terminal marks are exempt for opposite reasons: `closedAt` on an already-closed window
# is the same fact twice, and `openedAt` is stamped by opening itself, so rotating and then
# stamping it would record the open twice.
stamp_mark() {
  name=$1
  id=$(current_window)
  if is_window_closed "${id}"; then
    case "${name}" in
      closedAt) return 0 ;;
      openedAt)
        open_window >/dev/null
        return 0
        ;;
      *) id=$(open_window) ;;
    esac
  fi
  dir="${LOOP_DIR}/marks/${id}"
  mkdir -p "${dir}"
  date +%s >>"${dir}/${name}"
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
    id=$(current_window_unsafe) || exit 0
    dir="${LOOP_DIR}/marks/${id}"
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
      stamp_mark "$1"
    else
      printf 'unknown mark: %s\n' "$1" >&2
      printf 'writable names: %s\n' "${KNOWN_MARKS}" >&2
      exit 2
    fi
    ;;
esac
