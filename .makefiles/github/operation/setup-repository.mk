## リポジトリの初期化
.PHONY: setup-repo ## リポジトリの初期化
.PHONY: setup-replace-module ## node_tool_runnerでGoモジュール名の一括置換を実行
.PHONY: setup-replace-app-metadata ## node_tool_runnerでAPP_NAMEやOpenAPIタイトルの置換を実行
.PHONY: setup-replace-repository-reference ## node_tool_runnerでリポジトリ参照の置換を実行
.PHONY: setup-replace-license-copyright ## node_tool_runnerでLICENSEの著作権表示更新を実行
.PHONY: setup-replace-codeowners ## node_tool_runnerでCODEOWNERSの所有者の一括置換を実行
.PHONY: setup-remove-sample-api ## サンプルAPI(user/product/order)を一括削除し再生成・検証まで実行 # sample-api:line

SETUP_DRY_RUN_FLAG := $(if $(DRY_RUN),--dry-run,)

# git / gh の手順（タグの作り直し・ブランチ整備・デフォルトブランチ移動・リリースノート整理）は
# scripts/repo-setup（テスト付き）が持つ。取り消しの効かない操作ばかりで、分岐を実地で確かめようと
# するとリポジトリを本当に壊すしかないため、手順をシェルに置かない。
# ラベル・ルールセット・ワークフローの初期化は個別の make ターゲットなので、連鎖はここに残す。
# ホストの認証情報を使うためツールランナーは経由しない（cmd/db-slot と同じ扱い）。
REPO_SETUP := go run ./scripts/repo-setup

setup-repo:
	@if [ -n "$(DRY_RUN)" ]; then echo "❌ setup-repo は DRY_RUN 未対応です（ローカル/リモートを破壊的に変更します）。DRY_RUN を外して実行してください。"; exit 1; fi
	@$(REPO_SETUP) preflight

	@echo "🔧 ghコマンドのログインを開始します..."
	@$(MAKE) gh-login
	@echo "✅ ghコマンドのログインが完了しました。"

	@$(REPO_SETUP) bootstrap

	@echo "🔧 ルールセットの適用を開始します..."
	@$(MAKE) apply-branch-protection
	@echo "✅ ルールセットの適用を終了します。"

	@echo "🔧 ラベルの初期化を開始します..."
	@$(MAKE) delete-all-labels
	@$(MAKE) create-default-labels
	@echo "✅ ラベルの初期化を終了します。"

	@$(REPO_SETUP) prune-release-notes

	@echo "🔧 ワークフローの有効化を開始します..."
	@$(MAKE) enable-workflows
	@echo "✅ ワークフローの有効化を終了します。"

	@git remote remove upstream || true
	@echo "✅ Initialization complete. Default branch: production"

setup-replace-module:
	@if [ -z "$(OLD_MODULE)" ] || [ -z "$(NEW_MODULE)" ]; then \
		echo "❌ OLD_MODULE と NEW_MODULE を指定してください。例: make setup-replace-module OLD_MODULE=go-boilerplate NEW_MODULE=example-api"; \
		exit 1; \
	fi
	@docker compose run --rm node_tool_runner $(TSX) scripts/setup/replace-module $(OLD_MODULE) $(NEW_MODULE) $(SETUP_DRY_RUN_FLAG)

setup-replace-app-metadata:
	@if [ -z "$(APP_NAME)" ] || [ -z "$(OPENAPI_TITLE)" ] || [ -z "$(COPILOT_TITLE)" ]; then \
		echo "❌ APP_NAME, OPENAPI_TITLE, COPILOT_TITLE を指定してください。"; \
		echo "例: make setup-replace-app-metadata APP_NAME='Example API' OPENAPI_TITLE='Example API with Onion Architecture' COPILOT_TITLE='example-api Copilot Instructions'"; \
		exit 1; \
	fi
	@docker compose run --rm node_tool_runner $(TSX) scripts/setup/replace-app-metadata \
		--app-name "$(APP_NAME)" \
		--openapi-title "$(OPENAPI_TITLE)" \
		--copilot-title "$(COPILOT_TITLE)" \
		$(SETUP_DRY_RUN_FLAG)

setup-replace-repository-reference:
	@if [ -z "$(REPOSITORY)" ]; then \
		echo "❌ REPOSITORY を指定してください。例: make setup-replace-repository-reference REPOSITORY=example-org/example-api"; \
		exit 1; \
	fi
	@docker compose run --rm node_tool_runner $(TSX) scripts/setup/replace-repository-reference $(REPOSITORY) $(SETUP_DRY_RUN_FLAG)

setup-replace-license-copyright:
	@if [ -z "$(COPYRIGHT_HOLDER)" ]; then \
		echo "❌ COPYRIGHT_HOLDER を指定してください。例: make setup-replace-license-copyright COPYRIGHT_HOLDER='Example Inc.' COPYRIGHT_YEAR=2026"; \
		exit 1; \
	fi
	@docker compose run --rm node_tool_runner $(TSX) scripts/setup/replace-license-copyright \
		--holder "$(COPYRIGHT_HOLDER)" \
		$(if $(COPYRIGHT_YEAR),--year $(COPYRIGHT_YEAR),) \
		$(SETUP_DRY_RUN_FLAG)

setup-replace-codeowners:
	@if [ -z "$(OWNERS)" ]; then \
		echo "❌ OWNERS を指定してください。例: make setup-replace-codeowners OWNERS='@example-org/tech-leads'"; \
		exit 1; \
	fi
	@docker compose run --rm node_tool_runner $(TSX) scripts/setup/replace-codeowners \
		--owners "$(OWNERS)" \
		$(SETUP_DRY_RUN_FLAG)
# sample-api:begin

# サンプルAPIの削除はコンテナ内（node_tool_runner）で行い、削除後の再生成・整形・検証・DB 再構築は
# Go ツールチェーンが必要なためホスト側の make ターゲットを連鎖させる。
# プレビューは DRY_RUN=1 を付ける（削除も再生成も行わない）。
# make は起動時に makefile を全読込するため、手順1のスクリプトがこの .mk からターゲットを strip（自消滅）
# しても、実行中のレシピは継続し regen まで走る。
# サンプル削除で未使用になる直接依存が go.mod に残ると、後日 go.mod を触った無関係な PR で
# tidy-check が落ちる。tidy-lib は import が確定する gen の後、整理後の状態を lint で検証して
# 終えられるよう fix/lint の前に置く。各手順は && で連鎖し、途中の失敗が完了メッセージに隠れない。
setup-remove-sample-api:
	@docker compose run --rm node_tool_runner $(TSX) scripts/setup/remove-sample-api $(SETUP_DRY_RUN_FLAG)
	@if [ -n "$(DRY_RUN)" ]; then \
		echo "🟡 DRY_RUN のため再生成・整形・検証はスキップしました。"; \
	else \
		echo "🔧 mock-auth-server の固定ユーザーを中立な既定へリセットします..." && \
		$(MAKE) reset-mock-auth-users && \
		echo "🔧 再生成・整形・検証・DB 再構築を実行します..." && \
		$(MAKE) db-local-reinit db-test-reinit && \
		$(MAKE) gen-api gen-query && \
		$(MAKE) tidy-lib && \
		$(MAKE) fix lint && \
		echo "✅ サンプルAPIの削除・再生成・検証が完了しました。"; \
	fi
# sample-api:end
