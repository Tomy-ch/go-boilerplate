# docker compose のプロジェクト構成（infra 層 / app 層）
#
# 固定ポートでしか動けないサービス（database 5432 / observability 3000・4317・4318・3200 /
# garage 3900・3903）は全 checkout で 1 インスタンスだけを共有し、INFRA_PROJECT に置く。
# checkout 毎に必要な api_server / mock_auth_server だけを APP_PROJECT へ分離し、
# docker-compose.attach.yaml で共有インフラへ host-gateway 経由で接続する。
# DB スロット保持時は APP_PROJECT が gobp-wt-N（SERVE_PROJECT）になり、ホスト公開ポートもずれる。
INFRA_PROJECT ?= gobp-shared
APP_PROJECT ?= $(if $(SERVE_PROJECT),$(SERVE_PROJECT),gobp-app-$(notdir $(CURDIR)))
INFRA_SERVICES ?= database observability garage
APP_SERVICES ?= api_server mock_auth_server

# ホスト公開ポート（DB スロット未取得時の既定値。取得時は .gobp-db-slot が上書きする）。
API_HOST_PORT ?= 8080
MOCK_AUTH_HOST_PORT ?= 4000
DLV_HOST_PORT ?= 2345
PPROF_HOST_PORT ?= 6060
export API_HOST_PORT
export MOCK_AUTH_HOST_PORT
export DLV_HOST_PORT
export PPROF_HOST_PORT

# DB ツーリング（go_tool_runner / docker compose exec database）は共有 DB と同じネットワークで
# 動く必要があるため、プロジェクト名を明示しない compose 呼び出しは INFRA_PROJECT に寄せる。
COMPOSE_PROJECT_NAME ?= $(INFRA_PROJECT)
export COMPOSE_PROJECT_NAME

# 起動対象は常にサービス名で明示するため、profile 指定は対象を絞る用途ではなく有効化のためだけに置く
# （COMPOSE_INFRA は profile 無指定。tools 等でサービス名を省く呼び出しがあり、development を混ぜられない）。
COMPOSE_INFRA = docker compose -p $(INFRA_PROJECT)
COMPOSE_APP = docker compose -p $(APP_PROJECT) \
	-f docker-compose.yaml -f docker-compose.attach.yaml --profile development
