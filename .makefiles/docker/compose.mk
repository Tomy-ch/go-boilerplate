# docker compose のプロジェクト構成（infra 層 / app 層）
#
# 固定ポートでしか動けないサービス（database 5432 / observability 3000・4317・4318・3200 /
# garage 3900・3902 / elasticmq 9324）は全 checkout で 1 インスタンスだけを共有し、
# INFRA_PROJECT に置く。
# checkout 毎に必要な api_server / mock_auth_server だけを per-checkout のプロジェクトへ分離し、
# docker-compose.attach.yaml で共有インフラへ host-gateway 経由で接続する。
INFRA_PROJECT ?= $(if $(GOBP_DB_SHARED_PROJECT),$(GOBP_DB_SHARED_PROJECT),gobp-shared)
INFRA_SERVICES ?= database observability garage elasticmq
APP_SERVICES ?= api_server mock_auth_server

# DB ツーリング（go_tool_runner / docker compose exec database）は共有 DB と同じネットワークで
# 動く必要があるため、プロジェクト名を明示しない compose 呼び出しは INFRA_PROJECT に寄せる。
COMPOSE_PROJECT_NAME ?= $(INFRA_PROJECT)
export COMPOSE_PROJECT_NAME

# スロット定義はレシピ内シェルで都度読む。make のパース時に読む pool.mk の -include だけでは、
# slot-acquire がレシピ実行時に書き出した .gobp-db-slot を同一呼び出し内の後続ゴールへ反映できず、
# `make slot-acquire serve` が既定ポート・既定 DB のままサイレントに起動してしまう。
LOAD_SLOT = set -a; if [ -f .gobp-db-slot ]; then . ./.gobp-db-slot; fi; set +a

# イメージビルド時、mise は GitHub Releases API でツールを解決する。未認証 60 req/hour（IP 単位）は
# ビルド 1 回分に足りず 403 で落ちるため、ホストの gh からトークンを借りる。
LOAD_GH_TOKEN = export GITHUB_TOKEN="$${GITHUB_TOKEN:-$$(gh auth token 2>/dev/null || true)}"

# 起動対象は常にサービス名で明示するため、profile 指定は対象を絞る用途ではなく有効化のためだけに置く
# （COMPOSE_INFRA は profile 無指定。tools 等でサービス名を省く呼び出しがあり、development を混ぜられない）。
COMPOSE_INFRA = $(LOAD_GH_TOKEN); docker compose -p $(INFRA_PROJECT)
COMPOSE_APP = $(LOAD_SLOT); $(DB_SLOT_ENV); $(LOAD_GH_TOKEN); docker compose -p "$$APP_PROJECT" \
	-f docker-compose.yaml -f docker-compose.attach.yaml --profile development

# スロットと git 文脈から導かれる値（DB_LOCAL / DB_TEST / APP_PROJECT / AUTH_ISSUER /
# INFRA_NO_RECREATE）の導出は internal/cli/dbslot だけが持つ。レシピ内で 1 回解決してシェル変数として
# 読む（パース時に置くと make の全呼び出しにビルドが乗るため）。
# make 側に同じ導出を書き写さないこと。両方に置くと片方だけを直したときに黙ってずれ、
# 例えば mock 認証サーバーの既定ポートを変えても make 側のリテラルが追従しない。
# LOAD_SLOT が読むのはスロットの生の値（API_HOST_PORT など）で、こちらは導出済みの値を読む。
# 解決に失敗したときは `exit 1` を eval させて止める。空の解決結果で走らせると、リンク worktree の
# 検出が黙って外れたまま共有インフラを触ることになる。
DB_SLOT_ENV = eval "$$(go run ./cmd/ db-slot env || echo 'exit 1')"

# 共有インフラの稼働中コンテナを作り直させないフラグ。compose の config-hash は bind mount の source と
# build context を解決後の絶対パスで含むため、同一コミットの checkout 同士でもハッシュは一致せず、
# 既定のままだと共有インフラを触るたびに他 checkout の稼働ごと作り直してしまう。
# 渡すのは worktree のときだけ（判定は db-slot が持つ）。単一 checkout には奪い合う相手が居らず、
# compose 本来の「up は定義変更へ再収束する」契約を捨てる理由がないため空になる。独立した clone を
# 複数持つなどこの判定で拾えない構成では、`make infra-up INFRA_NO_RECREATE=--no-recreate` のように
# 明示する（明示された値は無条件に優先し、db-slot の判定へは落とさない）。
INFRA_NO_RECREATE_SH = $(if $(filter undefined,$(origin INFRA_NO_RECREATE)),$${INFRA_NO_RECREATE},$(INFRA_NO_RECREATE))
