# docker compose のプロジェクト構成（infra 層 / app 層）
#
# 固定ポートでしか動けないサービス（database 5432 / observability 3000・4317・4318・3200 /
# garage 3900・3903）は全 checkout で 1 インスタンスだけを共有し、INFRA_PROJECT に置く。
# checkout 毎に必要な api_server / mock_auth_server だけを per-checkout のプロジェクトへ分離し、
# docker-compose.attach.yaml で共有インフラへ host-gateway 経由で接続する。
INFRA_PROJECT ?= $(if $(GOBP_DB_SHARED_PROJECT),$(GOBP_DB_SHARED_PROJECT),gobp-shared)
INFRA_SERVICES ?= database observability garage
APP_SERVICES ?= api_server mock_auth_server

# DB ツーリング（go_tool_runner / docker compose exec database）は共有 DB と同じネットワークで
# 動く必要があるため、プロジェクト名を明示しない compose 呼び出しは INFRA_PROJECT に寄せる。
COMPOSE_PROJECT_NAME ?= $(INFRA_PROJECT)
export COMPOSE_PROJECT_NAME

# スロット定義はレシピ内シェルで都度読む。make のパース時に読む pool.mk の -include だけでは、
# slot-acquire がレシピ実行時に書き出した .gobp-db-slot を同一呼び出し内の後続ゴールへ反映できず、
# `make slot-acquire serve` が既定ポート・既定 DB のままサイレントに起動してしまう。
LOAD_SLOT = set -a; if [ -f .gobp-db-slot ]; then . ./.gobp-db-slot; fi; set +a
# スロット未取得時の app 層プロジェクト名。checkout 毎に分けて worktree 間の取り違えを防ぐ。
APP_PROJECT_DEFAULT = gobp-app-$(notdir $(CURDIR))
# ホスト公開ポートの既定値は docker-compose.attach.yaml 側の ${VAR:-...} が持つ（多重定義を避ける）。
APP_PROJECT_SH = $${SERVE_PROJECT:-$(APP_PROJECT_DEFAULT)}

# イメージビルド時、mise は GitHub Releases API でツールを解決する。未認証 60 req/hour（IP 単位）は
# ビルド 1 回分に足りず 403 で落ちるため、ホストの gh からトークンを借りる。
LOAD_GH_TOKEN = export GITHUB_TOKEN="$${GITHUB_TOKEN:-$$(gh auth token 2>/dev/null || true)}"

# 起動対象は常にサービス名で明示するため、profile 指定は対象を絞る用途ではなく有効化のためだけに置く
# （COMPOSE_INFRA は profile 無指定。tools 等でサービス名を省く呼び出しがあり、development を混ぜられない）。
COMPOSE_INFRA = $(LOAD_GH_TOKEN); docker compose -p $(INFRA_PROJECT)
COMPOSE_APP = $(LOAD_SLOT); $(LOAD_GH_TOKEN); docker compose -p "$(APP_PROJECT_SH)" \
	-f docker-compose.yaml -f docker-compose.attach.yaml --profile development
