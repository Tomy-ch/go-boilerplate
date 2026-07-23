#!/usr/bin/env bash
# DB スロットプール: 複数の git worktree / 主 checkout が「単一共有 Postgres（ホスト 5432）」を
# per-worktree の論理データベース（wt<N>_local / wt<N>_test）として貸し借りする。DB コンテナは
# 固定 compose プロジェクト gobp-shared に 1 個だけ立て、スロット N はその中のデータベース名ペアを
# 占有する。リースはホスト上のファイルロック（mkdir 原子性）で管理する。
# 詳細は docs/maintenance/db-worktree-pool.md。
set -euo pipefail

POOL_DIR="${GOBP_DB_POOL_DIR:-${TMPDIR:-/tmp}/gobp-db-pool}"
# make serve の並列化用: API サーバー / mock 認証サーバーのホスト公開ポートをスロット毎にずらす。
API_BASE_PORT="${GOBP_API_POOL_BASE:-8080}"
MOCK_AUTH_BASE_PORT="${GOBP_MOCK_AUTH_POOL_BASE:-4000}"
MAX_SLOTS="${GOBP_DB_POOL_MAX:-8}"
TTL_SECONDS="${GOBP_DB_POOL_TTL:-1800}"
# 全 worktree 共有の DB コンテナを載せる固定 compose プロジェクト。worktree はディレクトリ毎に
# compose の既定プロジェクト名が変わるため、共有 DB は明示の固定名に固定する必要がある。
SHARED_PROJECT="${GOBP_DB_SHARED_PROJECT:-gobp-shared}"

ROOT="$(git rev-parse --show-toplevel)"
SLOT_FILE="$ROOT/.gobp-db-slot"

now() { date +%s; }
lock_dir() { echo "$POOL_DIR/slot-$1.lock"; }
api_port_of() { echo $((API_BASE_PORT + $1)); }
mock_auth_port_of() { echo $((MOCK_AUTH_BASE_PORT + $1)); }
db_local_of() { echo "wt$1_local"; }
db_test_of() { echo "wt$1_test"; }
serve_project_of() { echo "gobp-wt-$1"; }

log() { echo "[db-pool] $*" >&2; }

# 共有 DB プロジェクトに対して docker compose を実行する（COMPOSE_PROJECT_NAME を固定）。
dc_db() { COMPOSE_PROJECT_NAME="$SHARED_PROJECT" docker compose "$@"; }

# meta ファイルを書く（owner=worktree 絶対パス, heartbeat=epoch, branch）。
write_meta() {
  local dir="$1" slot="$2"
  {
    echo "owner=$ROOT"
    echo "heartbeat=$(now)"
    echo "branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo '-')"
    echo "slot=$slot"
  } >"$dir/meta"
}

meta_get() { grep "^$2=" "$1/meta" 2>/dev/null | head -1 | cut -d= -f2- || true; }

# lease が stale か（heartbeat が TTL 超過）。owner が自分なら stale 扱いにしない。
is_stale() {
  local dir="$1"
  [ -f "$dir/meta" ] || return 0
  [ "$(meta_get "$dir" owner)" = "$ROOT" ] && return 1
  local hb; hb="$(meta_get "$dir" heartbeat)"
  [ -z "$hb" ] && return 0
  [ $(( $(now) - hb )) -gt "$TTL_SECONDS" ]
}

# 共有 DB コンテナを起動（固定プロジェクト・ホスト 5432）。--wait で healthcheck 完了まで待つ。
# フレッシュな volume は postgres 初期化前に後続の psql が走ると connection refused になるため待ち切る。
ensure_shared_db() {
  dc_db --profile database up -d --wait database >&2
}

db_exists() {
  dc_db exec -T database psql -U postgres -tAc \
    "SELECT 1 FROM pg_database WHERE datname='$1'" 2>/dev/null | grep -q 1
}

# worktree 用のデータベースを作成し（存在ガード）、拡張を流し込む。拡張は init スクリプトが
# local/test 決め打ちで作るため、動的に作る worktree DB には明示的にセットアップする必要がある。
ensure_slot_dbs() {
  local slot="$1" db
  for db in "$(db_local_of "$slot")" "$(db_test_of "$slot")"; do
    db_exists "$db" || dc_db exec -T database psql -U postgres -c "CREATE DATABASE \"$db\"" >&2
    dc_db exec -T database psql -U postgres -d "$db" -c "CREATE EXTENSION IF NOT EXISTS pg_trgm" >&2
  done
}

# .gobp-db-slot を worktree ルートへ書き出す（make が -include で読む KEY=VALUE 形式）。
write_slot_file() {
  local slot="$1"
  # COMPOSE_PROJECT_NAME=共有 DB プロジェクト。全 make DB ターゲットがこの共有 DB を指すようにする。
  # serve は server.mk が SERVE_PROJECT で上書きし、app コンテナだけ worktree 毎に分離する。
  {
    echo "SLOT=$slot"
    echo "DB_NAME_LOCAL=$(db_local_of "$slot")"
    echo "DB_NAME_TEST=$(db_test_of "$slot")"
    echo "API_HOST_PORT=$(api_port_of "$slot")"
    echo "MOCK_AUTH_HOST_PORT=$(mock_auth_port_of "$slot")"
    echo "COMPOSE_PROJECT_NAME=$SHARED_PROJECT"
    echo "SERVE_PROJECT=$(serve_project_of "$slot")"
  } >"$SLOT_FILE"
}

cmd_acquire() {
  mkdir -p "$POOL_DIR"
  ensure_shared_db

  # 既に自分がスロットを保持していれば heartbeat 更新して再利用（冪等）。
  if [ -f "$SLOT_FILE" ]; then
    local cur; cur="$(grep '^SLOT=' "$SLOT_FILE" | cut -d= -f2)"
    local d; d="$(lock_dir "$cur")"
    if [ -d "$d" ] && [ "$(meta_get "$d" owner)" = "$ROOT" ]; then
      write_meta "$d" "$cur"
      write_slot_file "$cur"
      ensure_slot_dbs "$cur"
      log "reuse slot $cur (db $(db_local_of "$cur") / $(db_test_of "$cur"))"
      cat "$SLOT_FILE"
      return 0
    fi
  fi

  local slot d
  for slot in $(seq 1 "$MAX_SLOTS"); do
    d="$(lock_dir "$slot")"
    if mkdir "$d" 2>/dev/null; then
      : # 原子取得成功
    elif is_stale "$d"; then
      log "reclaim stale slot $slot"
    else
      continue
    fi
    write_meta "$d" "$slot"
    write_slot_file "$slot"
    ensure_slot_dbs "$slot"
    log "acquired slot $slot (db $(db_local_of "$slot") / $(db_test_of "$slot"))"
    cat "$SLOT_FILE"
    return 0
  done

  log "no free slot (all $MAX_SLOTS in use). release one or raise GOBP_DB_POOL_MAX."
  return 1
}

cmd_release() {
  [ -f "$SLOT_FILE" ] || { log "no slot held by this worktree"; return 0; }
  local slot; slot="$(grep '^SLOT=' "$SLOT_FILE" | cut -d= -f2)"
  local d; d="$(lock_dir "$slot")"
  if [ -d "$d" ] && [ "$(meta_get "$d" owner)" = "$ROOT" ]; then
    rm -rf "$d"
    log "released slot $slot (databases left warm for reuse)"
  fi
  rm -f "$SLOT_FILE"
}

cmd_heartbeat() {
  [ -f "$SLOT_FILE" ] || return 0
  local slot; slot="$(grep '^SLOT=' "$SLOT_FILE" | cut -d= -f2)"
  local d; d="$(lock_dir "$slot")"
  [ -d "$d" ] && [ "$(meta_get "$d" owner)" = "$ROOT" ] && write_meta "$d" "$slot" || true
}

cmd_status() {
  mkdir -p "$POOL_DIR"
  printf '%-5s %-14s %-14s %-6s %-9s %-9s %s\n' SLOT DB_LOCAL DB_TEST API STATE AGE OWNER
  local slot d state age owner hb
  for slot in $(seq 1 "$MAX_SLOTS"); do
    d="$(lock_dir "$slot")"
    if [ -d "$d" ]; then
      owner="$(meta_get "$d" owner)"; hb="$(meta_get "$d" heartbeat)"
      age=$(( $(now) - ${hb:-0} ))
      if is_stale "$d"; then state="stale"; else state="in-use"; fi
    else
      state="free"; owner="-"; age="-"
    fi
    printf '%-5s %-14s %-14s %-6s %-9s %-9s %s\n' \
      "$slot" "$(db_local_of "$slot")" "$(db_test_of "$slot")" "$(api_port_of "$slot")" "$state" "$age" "$owner"
  done
}

case "${1:-}" in
  acquire) cmd_acquire ;;
  release) cmd_release ;;
  heartbeat) cmd_heartbeat ;;
  status) cmd_status ;;
  *) echo "usage: pool.sh {acquire|release|heartbeat|status}" >&2; exit 2 ;;
esac
