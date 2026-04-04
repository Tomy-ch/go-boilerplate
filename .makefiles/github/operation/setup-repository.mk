## リポジトリの初期化
.PHONY: setup-repo ## リポジトリの初期化
.PHONY: setup-replace-module ## node_tool_runnerでGoモジュール名の一括置換を実行
.PHONY: setup-replace-app-metadata ## node_tool_runnerでAPP_NAMEやOpenAPIタイトルの置換を実行
.PHONY: setup-replace-repository-reference ## node_tool_runnerでリポジトリ参照の置換を実行
.PHONY: setup-replace-license-copyright ## node_tool_runnerでLICENSEの著作権表示更新を実行
.PHONY: setup-remove-debug-handlers ## node_tool_runnerでdebugハンドラ一式の削除を実行

SETUP_DRY_RUN_FLAG := $(if $(DRY_RUN),--dry-run,)

setup-repo:
	@echo "🔧 設定を確認中..."

	@if git rev-parse --verify refs/tags/v0.0.0 >/dev/null 2>&1; then \
		echo "❌ タグ 【v0.0.0】 があります。初期化を停止します。"; exit 1; \
	fi

	@echo "✅ 初期化を開始します"

	@echo "🔧 ghコマンドのログインを開始します..."
	@make gh-login
	@echo "✅ ghコマンドのログインが完了しました。"

	@echo "🔧 v0.0.0のタグ打ちを開始します..."
	@git tag -a v0.0.0 -m "Initial boilerplate tag"
	@git push origin v0.0.0
	@echo "✅ v0.0.0のタグ打ちが完了しました。"

	@echo "🔧 ブランチ作成を開始します..."
	@if git show-ref --verify --quiet refs/heads/release/v0.1.0; then \
		echo "🟡 ブランチ 【release/v0.1.0】 は既に存在します。作成処理をスキップします。"; \
	else \
		git branch release/v0.1.0; \
	fi

	@if git show-ref --verify --quiet refs/heads/develop; then \
		echo "🟡 ブランチ 【develop】 は既に存在します。作成処理をスキップします。"; \
	else \
		git branch develop; \
	fi

	@if git show-ref --verify --quiet refs/heads/staging; then \
		echo "🟡 ブランチ 【staging】 は既に存在します。作成処理をスキップします。"; \
	else \
		git branch staging; \
	fi

	@if git show-ref --verify --quiet refs/heads/production; then \
		echo "🟡 ブランチ 【production】 は既に存在します。作成処理をスキップします。"; \
	else \
		git branch production; \
	fi

	@git push origin release/v0.1.0 develop staging production
	@echo "✅ ブランチの作成を終了します。"

	@echo "🔧 デフォルトブランチの設定を開始します..."
	@REPO=$$(gh repo view --json name,owner -q '.owner.login + "/" + .name'); \
		gh api -X PATCH repos/$$REPO -f default_branch=release/v0.1.0

	@git fetch --prune
	@git checkout release/v0.1.0
	@echo "✅ デフォルトブランチの設定を終了します。"

	@echo "🔧 ルールセットの適用を開始します..."
	@make apply-branch-protection
	@echo "✅ ルールセットの適用を終了します。"

	@echo "🔧 ラベルの初期化を開始します..."
	@make delete-all-labels
	@make create-default-labels
	@echo "✅ ラベルの初期化を終了します。"

	@echo "✅ Initialization complete. Default branch: release/v0.1.0"

setup-replace-module:
	@if [ -z "$(OLD_MODULE)" ] || [ -z "$(NEW_MODULE)" ]; then \
		echo "❌ OLD_MODULE と NEW_MODULE を指定してください。例: make setup-replace-module OLD_MODULE=boilerplate-go NEW_MODULE=example-api"; \
		exit 1; \
	fi
	@docker compose run --rm node_tool_runner node scripts/setup/replace-module.cjs $(OLD_MODULE) $(NEW_MODULE) $(SETUP_DRY_RUN_FLAG)

setup-replace-app-metadata:
	@if [ -z "$(APP_NAME)" ] || [ -z "$(OPENAPI_TITLE)" ] || [ -z "$(COPILOT_TITLE)" ]; then \
		echo "❌ APP_NAME, OPENAPI_TITLE, COPILOT_TITLE を指定してください。"; \
		echo "例: make setup-replace-app-metadata APP_NAME='Example API' OPENAPI_TITLE='Example API with Onion Architecture' COPILOT_TITLE='example-api Copilot Instructions'"; \
		exit 1; \
	fi
	@docker compose run --rm node_tool_runner node scripts/setup/replace-app-metadata.cjs \
		--app-name "$(APP_NAME)" \
		--openapi-title "$(OPENAPI_TITLE)" \
		--copilot-title "$(COPILOT_TITLE)" \
		$(SETUP_DRY_RUN_FLAG)

setup-replace-repository-reference:
	@if [ -z "$(REPOSITORY)" ]; then \
		echo "❌ REPOSITORY を指定してください。例: make setup-replace-repository-reference REPOSITORY=example-org/example-api"; \
		exit 1; \
	fi
	@docker compose run --rm node_tool_runner node scripts/setup/replace-repository-reference.cjs $(REPOSITORY) $(SETUP_DRY_RUN_FLAG)

setup-replace-license-copyright:
	@if [ -z "$(COPYRIGHT_HOLDER)" ]; then \
		echo "❌ COPYRIGHT_HOLDER を指定してください。例: make setup-replace-license-copyright COPYRIGHT_HOLDER='Example Inc.' COPYRIGHT_YEAR=2026"; \
		exit 1; \
	fi
	@docker compose run --rm node_tool_runner node scripts/setup/replace-license-copyright.cjs \
		--holder "$(COPYRIGHT_HOLDER)" \
		$(if $(COPYRIGHT_YEAR),--year $(COPYRIGHT_YEAR),) \
		$(SETUP_DRY_RUN_FLAG)

setup-remove-debug-handlers:
	@docker compose run --rm node_tool_runner node scripts/setup/remove-debug-handlers.cjs $(SETUP_DRY_RUN_FLAG)
