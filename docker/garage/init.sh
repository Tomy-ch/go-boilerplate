#!/bin/sh
# ローカル開発用 Garage の初期プロビジョニング（冪等）。
# レイアウト割当 → バケット作成 → 固定アクセスキー import → バケット許可 を行う。
# 値は env/.env の OBJECT_STORAGE_* と一致させる（make serve の Go 側接続情報）。
set -eu

CONFIG=/etc/garage.toml
BUCKET="${OBJECT_STORAGE_BUCKET:-gobp-local}"
ACCESS_KEY="${OBJECT_STORAGE_ACCESS_KEY_ID:-gobp-local-access-key}"
SECRET_KEY="${OBJECT_STORAGE_SECRET_ACCESS_KEY:-gobp-local-secret-key}"
KEY_NAME=gobp-local-key

# ノードにロール未割当なら単一ノードのレイアウトを割り当てて適用する（再実行時はスキップ）。
# CLI はノード鍵を共有 meta ボリュームから読む（garage server が生成したもの）。
if garage -c "$CONFIG" layout show 2>/dev/null | grep -q "No nodes currently have a role"; then
  NODE_ID=$(garage -c "$CONFIG" node id -q | cut -d@ -f1)
  garage -c "$CONFIG" layout assign -z dc1 -c 1G "$NODE_ID"
  garage -c "$CONFIG" layout apply --version 1
fi

garage -c "$CONFIG" bucket create "$BUCKET" 2>/dev/null || true

# 固定アクセスキー（既存なら import は失敗するため無視）。
garage -c "$CONFIG" key import --yes -n "$KEY_NAME" "$ACCESS_KEY" "$SECRET_KEY" 2>/dev/null || true

garage -c "$CONFIG" bucket allow --read --write "$BUCKET" --key "$ACCESS_KEY"

echo "garage provisioning done: bucket=${BUCKET} access_key=${ACCESS_KEY}"
