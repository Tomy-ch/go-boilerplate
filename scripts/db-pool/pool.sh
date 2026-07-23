#!/usr/bin/env bash
# DB スロットプール: 複数の git worktree / 主 checkout が DB コンテナを per-port スロットとして
# 貸し借りする。各スロット N は compose プロジェクト gobp-db-slot-N + ホストポート BASE+N を占有し、
# ホスト上のファイルロック（mkdir 原子性）でリースを管理する。詳細は docs/maintenance/db-worktree-pool.md。
set -euo pipefail

POOL_DIR="${GOBP_DB_POOL_DIR:-${TMPDIR:-/tmp}/gobp-db-pool}"
BASE_PORT="${GOBP_DB_POOL_BASE:-5432}"
# make serve の並列化用: API サーバー / mock 認証サーバーのホスト公開ポートもスロット毎にずらす。
API_BASE_PORT="${GOBP_API_POOL_BASE:-8080}"
MOCK_AUTH_BASE_PORT="${GOBP_MOCK_AUTH_POOL_BASE:-4000}"
MAX_SLOTS="${GOBP_DB_POOL_MAX:-8}"
TTL_SECONDS="${GOBP_DB_POOL_TTL:-1800}"

ROOT="$(git rev-parse --show-toplevel)"
SLOT_FILE="$ROOT/.gobp-db-slot"

now() { date +%s; }
lock_dir() { echo "$POOL_DIR/slot-$1.lock"; }
port_of() { echo $((BASE_PORT + $1)); }
api_port_of() { echo $((API_BASE_PORT + $1)); }
mock_auth_port_of() { echo $((MOCK_AUTH_BASE_PORT + $1)); }
project_of() { echo "gobp-db-slot-$1"; }

log() { echo "[db-pool] $*" >&2; }

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

# スロットのいずれかのホスト公開ポート（DB / API）が「自分のスロット以外」に占有されているか。
foreign_busy() {
  local slot="$1" port names
  for port in "$(port_of "$slot")" "$(api_port_of "$slot")"; do
    names="$(docker ps --filter "publish=$port" --format '{{.Names}}' 2>/dev/null || true)"
    [ -z "$names" ] && continue
    echo "$names" | grep -q "^$(project_of "$slot")-" && continue
    return 0
  done
  return 1
}

# スロットのコンテナを起動（compose プロジェクト分離 + ホストポート割当）。
# --wait で healthcheck 完了まで待つ。フレッシュな volume は postgres 初期化前に
# 後続の reinit（psql）が走ると connection refused になるため、ここで待ち切る。
up_slot() {
  local slot="$1"
  COMPOSE_PROJECT_NAME="$(project_of "$slot")" DB_HOST_PORT="$(port_of "$slot")" \
    docker compose --profile database up -d --wait database >&2
}

# .gobp-db-slot を worktree ルートへ書き出す（make が -include で読む KEY=VALUE 形式）。
write_slot_file() {
  local slot="$1"
  # *_HOST_PORT = ホスト公開ポート（compose の publish と host 側からの接続先）。
  # コンテナ内部ポートは常に固定（DB=5432 / API=8080 / mock_auth=4000）なので、ここには書かない。
  {
    echo "SLOT=$slot"
    echo "DB_HOST_PORT=$(port_of "$slot")"
    echo "API_HOST_PORT=$(api_port_of "$slot")"
    echo "MOCK_AUTH_HOST_PORT=$(mock_auth_port_of "$slot")"
    echo "COMPOSE_PROJECT_NAME=$(project_of "$slot")"
  } >"$SLOT_FILE"
}

cmd_acquire() {
  mkdir -p "$POOL_DIR"

  # 既に自分がスロットを保持していれば heartbeat 更新して再利用（冪等）。
  if [ -f "$SLOT_FILE" ]; then
    local cur; cur="$(grep '^SLOT=' "$SLOT_FILE" | cut -d= -f2)"
    local d; d="$(lock_dir "$cur")"
    if [ -d "$d" ] && [ "$(meta_get "$d" owner)" = "$ROOT" ]; then
      write_meta "$d" "$cur"
      write_slot_file "$cur"
      up_slot "$cur"
      log "reuse slot $cur (port $(port_of "$cur"))"
      cat "$SLOT_FILE"
      return 0
    fi
  fi

  local slot d
  for slot in $(seq 0 $((MAX_SLOTS - 1))); do
    d="$(lock_dir "$slot")"
    if mkdir "$d" 2>/dev/null; then
      : # 原子取得成功
    elif is_stale "$d"; then
      log "reclaim stale slot $slot"
    else
      continue
    fi
    if foreign_busy "$slot"; then
      rmdir "$d" 2>/dev/null || true
      log "slot $slot port busy (foreign), skip"
      continue
    fi
    write_meta "$d" "$slot"
    write_slot_file "$slot"
    up_slot "$slot"
    log "acquired slot $slot (port $(port_of "$slot"), project $(project_of "$slot"))"
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
    log "released slot $slot (container left warm)"
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
  printf '%-5s %-6s %-6s %-10s %-9s %s\n' SLOT DB API STATE AGE OWNER
  local slot d state age owner hb
  for slot in $(seq 0 $((MAX_SLOTS - 1))); do
    d="$(lock_dir "$slot")"
    if [ -d "$d" ]; then
      owner="$(meta_get "$d" owner)"; hb="$(meta_get "$d" heartbeat)"
      age=$(( $(now) - ${hb:-0} ))
      if is_stale "$d"; then state="stale"; else state="in-use"; fi
    else
      state="free"; owner="-"; age="-"
    fi
    docker ps --filter "publish=$(port_of "$slot")" --format '{{.Names}}' 2>/dev/null | grep -q "^$(project_of "$slot")-" && state="$state,up"
    printf '%-5s %-6s %-6s %-10s %-9s %s\n' "$slot" "$(port_of "$slot")" "$(api_port_of "$slot")" "$state" "$age" "$owner"
  done
}

case "${1:-}" in
  acquire) cmd_acquire ;;
  release) cmd_release ;;
  heartbeat) cmd_heartbeat ;;
  status) cmd_status ;;
  *) echo "usage: pool.sh {acquire|release|heartbeat|status}" >&2; exit 2 ;;
esac
