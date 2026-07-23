## アプリケーションサーバー関連
.PHONY: serve ## 開発環境(API/DB/otel-lgtm)を起動する
.PHONY: serve-build ## キャッシュを利用したビルド後、開発環境を起動する
.PHONY: serve-build-clean ## キャッシュ無効・base image を pull したクリーンビルド後、開発環境を起動する
.PHONY: tools ## 開発支援サービス(DB/docs/SQL viewer)を起動する
.PHONY: all ## 全サービス(開発/ツール)を一括起動する
.PHONY: tool-runners-build ## ツールランナー(go/node/python)をビルドする（キャッシュ利用・起動はしない）
.PHONY: tool-runners-build-clean ## ツールランナーをクリーンビルドする（--no-cache --pull・起動はしない）

serve:
	@echo "🔄 開発環境を起動します。"
	@# .gobp-db-slot の有無はレシピ内シェルで実行時に判定する（ifeq/wildcard は make の parse 時評価で、
	@# `make db-acquire serve` のように同一起動でチェーンすると acquire がファイルを作る前に分岐が確定し、
	@# serve が誤って非プール分岐へ落ちるため）。プール利用時は共有 DB（COMPOSE_PROJECT_NAME 由来）を参照し、
	@# app コンテナだけ SERVE_PROJECT に分離して起動する。
	@if [ -f .gobp-db-slot ]; then \
		set -a; . ./.gobp-db-slot; set +a; \
		echo "  （DB スロットプール: 共有 DB $$COMPOSE_PROJECT_NAME を参照し、app を $$SERVE_PROJECT に分離）"; \
		COMPOSE_PROJECT_NAME="$$COMPOSE_PROJECT_NAME" docker compose --profile database up -d --wait database; \
		COMPOSE_PROJECT_NAME="$$SERVE_PROJECT" docker compose -f docker-compose.yaml -f docker-compose.pool.yaml \
			--profile development up -d api_server mock_auth_server; \
		go run ./cmd/ db-pool heartbeat; \
		echo "✅ 起動完了。API: http://localhost:$$API_HOST_PORT（o11y/dlv/pprof はプール serve では非分離）"; \
	else \
		docker compose --profile development up -d; \
		echo "✅ 開発環境の起動が完了しました。Grafana: http://localhost:3000"; \
	fi

serve-build:
	@echo "🧰 ビルド後、開発環境を起動します。"
	@docker compose --profile development up -d --build
	@echo "✅ 開発環境の起動が完了しました。"

serve-build-clean:
	@echo "🧹 クリーンビルド後、開発環境を起動します（--no-cache --pull）。"
	@docker compose --profile development build --no-cache --pull
	@docker compose --profile development up -d
	@echo "✅ 開発環境の起動が完了しました。"

tools:
	@echo "🔄 開発ツールを起動します。"
	@docker compose --profile tools up -d --build
	@echo "✅ 開発ツールの起動が完了しました。"

all:
	@echo "🔄 全サービス(開発/ツール)を一括起動します。"
	@docker compose --profile development --profile tools up -d --build
	@echo "✅ 全サービスの起動が完了しました。Grafana: http://localhost:3000"

tool-runners-build:
	@echo "🧰 ツールランナーをビルドします。"
	@docker compose build go_tool_runner node_tool_runner python_tool_runner
	@echo "✅ ツールランナーのビルドが完了しました。"

tool-runners-build-clean:
	@echo "🧹 ツールランナーをクリーンビルドします（--no-cache --pull）。"
	@docker compose build --no-cache --pull go_tool_runner node_tool_runner python_tool_runner
	@echo "✅ ツールランナーのクリーンビルドが完了しました。"
