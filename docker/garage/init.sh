#!/bin/sh
# ローカル開発用 Garage の初期プロビジョニング（冪等）。
# レイアウト割当 → バケット作成 → 固定アクセスキー import → バケット許可 → 公開配信の許可 を行う。
# 接続情報は Go 側と同じ env/.env を compose が env_file で渡す。既定値を持たないのは、
# 独自の既定へ黙って流れると Go 側と食い違ったまま起動してしまうためで、未設定なら即座に落とす。
set -eu

CONFIG=/etc/garage.toml
BUCKET="${OBJECT_STORAGE_BUCKET:?}"
ACCESS_KEY="${OBJECT_STORAGE_ACCESS_KEY_ID:?}"
SECRET_KEY="${OBJECT_STORAGE_SECRET_ACCESS_KEY:?}"
KEY_NAME=gobp-local-key

# ノードにロール未割当なら単一ノードのレイアウトを割り当てて適用する（再実行時はスキップ）。
# CLI はノード鍵を共有 meta ボリュームから読む（garage server が生成したもの）。
# インスタンスは全 checkout で共有され、初回ブートストラップ時に複数の checkout が同時に
# ここへ到達しうる。割当は先着が勝てばよいので競合による失敗は無視し、どちらも割り当てられ
# なかった場合は後続の bucket allow が失敗して顕在化する。
if garage -c "$CONFIG" layout show 2>/dev/null | grep -q "No nodes currently have a role"; then
  NODE_ID=$(garage -c "$CONFIG" node id -q | cut -d@ -f1)
  garage -c "$CONFIG" layout assign -z dc1 -c 1G "$NODE_ID" 2>/dev/null || true
  garage -c "$CONFIG" layout apply --version 1 2>/dev/null || true
fi

garage -c "$CONFIG" bucket create "$BUCKET" 2>/dev/null || true

# 固定アクセスキー（既存なら import は失敗するため無視）。
garage -c "$CONFIG" key import --yes -n "$KEY_NAME" "$ACCESS_KEY" "$SECRET_KEY" 2>/dev/null || true

garage -c "$CONFIG" bucket allow --read --write "$BUCKET" --key "$ACCESS_KEY"

# 匿名 read の公開配信（s3_web）を有効化する。write は API の認可のままで、read だけを開く。
garage -c "$CONFIG" bucket website --allow "$BUCKET"

echo "garage provisioning done: bucket=${BUCKET} access_key=${ACCESS_KEY} website=allowed"
