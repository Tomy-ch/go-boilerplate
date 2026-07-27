## アプリケーションサーバー関連
.PHONY: serve ## 開発環境(API/mock認証)を起動する（共有インフラは自動起動）
.PHONY: serve-build ## キャッシュを利用したビルド後、開発環境を起動する
.PHONY: serve-build-clean ## キャッシュ無効・base image を pull したクリーンビルド後、開発環境を起動する
.PHONY: serve-stop ## この checkout の app コンテナを停止する（共有インフラは残す）
.PHONY: infra-up ## 共有インフラ(DB/o11y/オブジェクトストレージ)を起動する
.PHONY: infra-down ## 共有インフラを停止する（全 checkout に影響する）
.PHONY: tools ## 開発支援サービス(docs/SQL viewer)を起動する
.PHONY: all ## 全サービス(共有インフラ/開発/ツール)を一括起動する
.PHONY: tool-runners-build ## ツールランナー(go/node/python)をビルドする（キャッシュ利用・起動はしない）
.PHONY: tool-runners-build-clean ## ツールランナーをクリーンビルドする（--no-cache --pull・起動はしない）

# garage_init は完了して終了する one-shot のため --wait（正常終了も失敗と見なす）の対象から外し、
# 終了まで同期ブロックする run で実行する。app はプロジェクトを跨ぐため compose の依存で待てず、
# バケット未プロビジョニングのまま S3 を触ると 503 になる。
infra-up:
	@echo "🔄 共有インフラを起動します... (project=$(INFRA_PROJECT))"
	@$(COMPOSE_INFRA) --profile development up -d --wait $(INFRA_SERVICES)
	@$(COMPOSE_INFRA) --profile development run --rm -T garage_init > /dev/null
	@echo "✅ 共有インフラが起動しています。Grafana: http://localhost:3000"

infra-down:
	@echo "🛑 共有インフラを停止します。全 checkout / worktree に影響します... (project=$(INFRA_PROJECT))"
	@$(COMPOSE_INFRA) down
	@echo "✅ 共有インフラを停止しました（データボリュームは保持されます）。"

serve:
	@echo "🔄 開発環境を起動します。"
	@$(MAKE) infra-up
	@$(COMPOSE_APP) up -d $(APP_SERVICES)
	@# スロット保持時のみ heartbeat を更新する（未取得なら何もしない）。
	@go run ./cmd/ db-slot heartbeat
	@$(LOAD_SLOT); echo "✅ 開発環境の起動が完了しました。API: http://localhost:$${API_HOST_PORT:-8080} (project=$(APP_PROJECT_SH))"

serve-build:
	@echo "🧰 ビルド後、開発環境を起動します。"
	@$(MAKE) infra-up
	@$(COMPOSE_APP) up -d --build $(APP_SERVICES)
	@$(LOAD_SLOT); echo "✅ 開発環境の起動が完了しました。API: http://localhost:$${API_HOST_PORT:-8080}"

serve-build-clean:
	@echo "🧹 クリーンビルド後、開発環境を起動します（--no-cache --pull）。"
	@$(COMPOSE_APP) build --no-cache --pull $(APP_SERVICES)
	@$(MAKE) infra-up
	@$(COMPOSE_APP) up -d $(APP_SERVICES)
	@$(LOAD_SLOT); echo "✅ 開発環境の起動が完了しました。API: http://localhost:$${API_HOST_PORT:-8080}"

serve-stop:
	@$(LOAD_SLOT); echo "🛑 app コンテナを停止します... (project=$(APP_PROJECT_SH))"
	@$(COMPOSE_APP) down
	@echo "✅ app コンテナを停止しました（共有インフラは稼働したままです）。"

tools:
	@echo "🔄 開発ツールを起動します。"
	@$(COMPOSE_INFRA) --profile tools up -d --build
	@echo "✅ 開発ツールの起動が完了しました。SQL editor: http://localhost:7000 / docs: http://localhost:7001"

all:
	@echo "🔄 全サービス(共有インフラ/開発/ツール)を一括起動します。"
	@$(MAKE) tools
	@$(MAKE) serve-build
	@echo "✅ 全サービスの起動が完了しました。Grafana: http://localhost:3000"

tool-runners-build:
	@echo "🧰 ツールランナーをビルドします。"
	@$(LOAD_GH_TOKEN); docker compose build go_tool_runner node_tool_runner python_tool_runner
	@echo "✅ ツールランナーのビルドが完了しました。"

tool-runners-build-clean:
	@echo "🧹 ツールランナーをクリーンビルドします（--no-cache --pull）。"
	@$(LOAD_GH_TOKEN); docker compose build --no-cache --pull go_tool_runner node_tool_runner python_tool_runner
	@echo "✅ ツールランナーのクリーンビルドが完了しました。"
